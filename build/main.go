package main

import (
	"embed"
	"fmt"
	"hash/fnv"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var buildTime = "unknown"

//go:embed web
var webFiles embed.FS

var (
	authURL      string
	authInternal string
	appURL       string
	appToken     string
	tmpl         *template.Template
	httpClient   = &http.Client{}
)

type pageData struct {
	User   *User
	Error  string
	Apps   []appDef
	Region regionDef
}

// appDef is the single source of truth for an app in the grid.
// The grid (template) and the open handler both derive from it.
type appDef struct {
	Slug string // url segment, e.g. "blur"
	Name string // display name shown on the card
	Sub  string // subdomain; the URL is https://<Sub>.<region.Domain>
	URL  string // filled at startup by initRegion (region.go) — don't set by hand
	Icon string // icon filename under web/icons/ (e.g. "blur.svg"); empty → first-letter tile

	// Info-modal content, filled at startup by initRegion from region.Apps
	// (region.go) — don't set by hand.
	Desc     string
	Features []string
}

var apps = []appDef{
	{Slug: "nom-nom", Name: "nom-nom", Sub: "nom-nom", Icon: "nom-nom.svg"},
	{Slug: "wgetbash", Name: "wget-bash", Sub: "wgetbash", Icon: "wget-bash.svg"},
	{Slug: "blur", Name: "blur", Sub: "blur", Icon: "blur.svg"},
	{Slug: "qcode", Name: "qcode", Sub: "qcode", Icon: "qcode.svg"},
}

// assetVer is a short content hash of everything embedded under web/. The
// template appends it to static URLs as ?v=…, so a rebuilt binary serves new
// URLs and the long cacheStatic TTL below can never pin a stale asset. Without
// it, replacing an icon stayed invisible for up to 30 days to anyone who had
// already loaded the page.
var assetVer = "0"

func initAssetVer() {
	h := fnv.New64a()
	err := fs.WalkDir(webFiles, "web", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := webFiles.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path)) //nolint:errcheck
		h.Write(b)            //nolint:errcheck
		return nil
	})
	if err != nil {
		log.Printf("asset-version error=%v", err)
		return
	}
	assetVer = strconv.FormatUint(h.Sum64(), 36)
}

// cacheStatic wraps a handler with a long-lived cache header. Use for media that
// only changes on deploy (background, icons, favicon) so browsers don't re-fetch it.
// Safe because every URL it serves carries ?v=assetVer.
func cacheStatic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30 days
		h.ServeHTTP(w, r)
	})
}

// appBySlug returns the app with the given slug, or nil.
func appBySlug(slug string) *appDef {
	for i := range apps {
		if apps[i].Slug == slug {
			return &apps[i]
		}
	}
	return nil
}

func initTemplate() {
	src, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		log.Fatalf("web/index.html not found: %v", err)
	}
	tmpl = template.Must(template.New("index").Funcs(template.FuncMap{
		"join": strings.Join,
		"ver":  func() string { return assetVer },
	}).Parse(string(src)))
}

// ── request logging ───────────────────────────────────────────────────────────

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(status int) {
	sw.status = status
	sw.ResponseWriter.WriteHeader(status)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

// ── app routes ────────────────────────────────────────────────────────────────

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if code := r.URL.Query().Get("code"); code != "" {
		handleCallback(w, r, code)
		return
	}
	var user *User
	if uid := sessionUserID(r); uid != 0 {
		user, _ = getUserByID(uid)
	}
	tmpl.Execute(w, pageData{User: user, Apps: apps, Region: region}) //nolint:errcheck
}

// handleOpen issues a cross-app delegate redirect for /open/{slug}.
func handleOpen(w http.ResponseWriter, r *http.Request) {
	uid := sessionUserID(r)
	if uid == 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	app := appBySlug(r.PathValue("slug"))
	if app == nil {
		http.NotFound(w, r)
		return
	}
	code, err := delegateCode(uid)
	if err != nil {
		log.Printf("open-%s uid=%d error=%v", app.Slug, uid, err)
		http.Error(w, "could not open app", http.StatusInternalServerError)
		return
	}
	log.Printf("open-%s uid=%d", app.Slug, uid)
	http.Redirect(w, r, app.URL+"/?code="+code, http.StatusFound)
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "--info") {
		fmt.Printf("menu built: %s\n", buildTime)
		os.Exit(0)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	godotenv.Load() //nolint:errcheck

	authURL = os.Getenv("AUTH_URL")
	authInternal = os.Getenv("AUTH_INTERNAL")
	appURL = os.Getenv("APP_URL")
	appToken = os.Getenv("APP_TOKEN")

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		secretKey = "dev-secret"
	}
	jwtSecret = []byte(secretKey)

	initRegion()
	initDB()
	initAssetVer()
	initTemplate()

	webFS, _ := fs.Sub(webFiles, "web")
	fileServer := http.FileServer(http.FS(webFS))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /login", handleLogin)
	mux.HandleFunc("GET /logout", handleLogout)
	mux.HandleFunc("GET /open/{slug}", handleOpen)
	mux.Handle("GET /favicon.svg", cacheStatic(fileServer))
	mux.Handle("GET /background.webp", cacheStatic(fileServer))
	mux.Handle("GET /icons/{file}", cacheStatic(fileServer))
	mux.Handle("GET /shell.css", fileServer)
	mux.Handle("GET /shell.js", fileServer)
	mux.Handle("GET /app.css", fileServer)
	mux.Handle("GET /app.js", fileServer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8890"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, logMiddleware(mux)))
}
