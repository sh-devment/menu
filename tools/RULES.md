# RULES.md

Project rules for **menu** — for humans and for Claude Code alike (the local, gitignored
`CLAUDE.md` just imports this file). Single source of truth: update it together with the code.

## What this is

**menu** is an app-portal for the sh-development ecosystem. Authenticated users see a grid of available apps and can open any of them without re-logging in (cross-app SSO via delegate codes). It is built on the `example/` auth-client template.

Auth center: one per zone — `https://auth-center.sh-development.ru` / `https://auth-center.sh-development.com`.

## Rules

- Go may be run natively (dev machine is linux/amd64, same as prod) — `go build`, `go vet`,
  `gofmt` directly in `build/`. Docker stays available as a tool (dev hot-reload, release build).
- Need something installed on the dev machine? Ask first, install only after agreement.
- `example/` is a read-only reference template. Do not modify it.

## Commands

**Local dev (hot-reload via Air):**
```bash
cp .env.example .env   # fill in variables once
docker-compose -f dev-compose.yml up --remove-orphans
```
Force rebuild after Dockerfile changes: add `--build`.

**Quick check (native):**
```bash
cd build && gofmt -l . && go vet ./... && go build -o /dev/null .
```

**Production binary (linux/amd64, committed to git):**
```bash
docker-compose -f prod-compose.yml run --rm release
# binary lands in bin/menu
```
Force no-cache: `docker-compose -f prod-compose.yml build --no-cache release && docker-compose -f prod-compose.yml run --rm release`

**Deploy:** GitHub Actions → `Deploy` (manual `workflow_dispatch`). Matrix over `ru` / `com`,
each on its own self-hosted runner + environment; runs `bin/deploy.sh`, which installs
`bin/$APP`, writes `/opt/$APP/$APP.env` and the systemd unit, restarts, health-checks and
rolls back on failure (previous state kept in `/opt/$APP/last-deploy-backup`). No build step
on the server — commit the rebuilt `bin/menu` before deploying. `APP` is a required repo
variable; `REGION` comes from the matrix.

**Nightly backup:** `bin/backup.sh` (generic, takes `$APP` as its argument). Every deploy
installs it as `/opt/$APP/backup.sh` and, if there's no such entry yet, adds to worker's crontab:
`0 1 * * * flock /var/lock/backups.lock /opt/$APP/backup.sh $APP >> /backup/$APP/backup.log 2>&1`.
Every app uses the same time, and the shared lock runs them one after another. It keeps the 7
latest `/backup/$APP/$APP-<date>.db`. One-time server setup: `/backup` must be writable by worker.

## File roles

```
build/
  main.go        — server setup, env loading, all app routes/handlers  ← edit this
  auth-human.go  — JWT sessions, /login, /logout, handleCallback       ← do not edit
  auth-server.go — server-to-server calls to auth-center (delegateCode) ← do not edit
  db.go          — SQLite init, users table, core queries               ← do not edit
  app_db.go      — app-specific migrations (appMigrate func)           ← edit this
  region.go      — ALL ru/com zone differences (regionDef, initRegion) ← edit, review together
  web/
    shell.css / shell.js  — shared chrome: navbar, profile popover     ← do not edit
    app.css / app.js      — app-specific styles and logic              ← edit these
    index.html            — Go template: {{if .User}} / {{else}}       ← edit this
```

## Auth flow

```
GET /login → redirect to auth-center → redirect back to /?code=<one-time-code>
GET /       → handleCallback(code) → POST AUTH_INTERNAL/exchange → upsertUser → set JWT cookie → redirect /
```

- `sessionUserID(r)` — returns internal DB `id` (int64), 0 if not logged in
- `requireAuth(handler)` — middleware that redirects to `/login` if not logged in
- JWT cookie: 30-day expiry, HttpOnly, SameSite=Lax, signed with `SECRET_KEY`

## App definitions (single source of truth)

All apps live in one `apps []appDef` slice in `main.go`. The grid template and the open
handler both derive from it — add an app in one place:

```go
var apps = []appDef{
    {Slug: "blur", Name: "blur", Sub: "blur", Icon: "blur.svg"},
    ...
}
```

`appDef{Slug, Name, Sub, URL, Icon, Desc, Features}`. `URL`, `Desc`, `Features` are **not**
set by hand. `initRegion()` (`region.go`) fills `URL` as `https://<Sub>.<region.Domain>` and takes
`Desc`/`Features` from `region.Apps[slug]`. A new app needs an entry here **and** in `Apps` of
every region. `appBySlug(slug)` looks one up.

## Regions

Two main domains — `sh-development.ru` and `sh-development.com` — each with its own
subdomains (one per app). Every app exists in both zones, but its content and links may
differ slightly between them. SSO works only within one zone. Both profiles are compiled into the binary (`regions` map → `regionDef`);
`REGION` env picks one at startup, unknown/empty is fatal. The deploy workflow passes
`REGION: ${{ matrix.region }}`, so it always matches the runner/environment.

- **All zone differences live in one file: `build/region.go`.** Domain, links, texts,
  per-app content/availability — only there, never in `main.go`, templates or JS. It is one
  of the most critical nodes of the system: any change to `region.go` is reviewed together
  with the owner before commit.
- Infrastructure (auth URLs, tokens, port) → env.
- Template gets the profile as `.Region` (e.g. `{{.Region.Domain}}`) — never hardcode the domain.

### Language

Language follows the region: `ru` is in Russian (a few English UI words like `apps`, `info`,
`log out` stay as they are), `com` is strictly English. There is no browser-language detection.

- Every user-facing string lives in `regionDef.Text` (`pageText`, named fields) and
  `regionDef.Apps` (per-app `Desc`/`Features`). Templates use `{{.Region.Text.X}}`, and
  `<html lang>` comes from `Region.Lang`.
- JS strings live in `Text.JS` (`jsText`). They are rendered into `<script id="texts">` as JSON,
  and `app.js` reads them into `TEXT`. No text literals in `app.js` or `index.html`.
- **Completeness check at startup:** `initRegion()` checks **all** regions. Any empty string, or
  an app missing from `Apps`, is fatal (health check fails → rollback). A text added in `ru` must
  be added in `com` too.
- Nobody checks for Cyrillic in `com`. That's up to the developer.
- Technical text (logs, `http.Error` bodies) is English in both zones and doesn't go in `region.go`.

## Cross-app redirect (delegate flow)

One generic handler serves every app via `GET /open/{slug}` (Go 1.22 path value):

```go
func handleOpen(w http.ResponseWriter, r *http.Request) {
    uid := sessionUserID(r)
    if uid == 0 { http.Redirect(w, r, "/login", http.StatusFound); return }
    app := appBySlug(r.PathValue("slug"))
    if app == nil { http.NotFound(w, r); return }
    code, err := delegateCode(uid)
    // ... → redirect app.URL + "/?code=" + code
}
```

No per-app handlers — registering a new app needs only an `apps` entry.

## Availability

No reachability probes — neither on page load nor on the server. A card is a plain link to
`/open/{slug}`. On click `app.js` starts a 2s timer (`OPEN_TIMEOUT`); if the page is still
there when it fires, the redirect didn't happen: it calls `window.stop()` and shows the
"app temporarily unavailable" popup (`Text.JS.UnavailableTitle/Text`).

## App grid

Current apps (subdomains, on the active region's domain): `nom-nom`, `wgetbash`, `blur`, `qcode`.

Each card shows: icon tile, name, and an info button (`i`) that opens a modal with the
app's `Desc`/`Features`.

## Navigation (two tabs)

1. **Apps** — main page, app grid
2. **Info** — hardcoded static info page

Profile popover contains: auth provider, name, uid, README link (`https://github.com/shumilovsergey/menu#`), logout button. No extra items.

## Web layer

All static files must be registered as explicit `GET` routes in `main.go` (Go 1.22+ method-prefix mux). Files are embedded via `//go:embed web`.

Template receives `pageData{User *User, Error string, Apps []appDef, Region regionDef}`:
- `{{if .User}}` — navbar + `<main class="app-content">`
- `{{else}}` — login screen with `.login-card` and `.app-about`

Use CSS variables from `shell.css` only — no hardcoded colors in `app.css`. Variables: `--bg`, `--card`, `--border`, `--border-active`, `--text`, `--text-dim`, `--accent`, `--neon`.

## Database

SQLite via `modernc.org/sqlite` (no CGO). Core `users` table auto-created on startup. Add app-specific tables in `app_db.go → appMigrate()`. Always FK to `users.id`, never `auth_id`.

New columns on existing DBs:
```go
db.Exec(`ALTER TABLE users ADD COLUMN my_col TEXT`) //nolint:errcheck
```

## Logging

One `log.Printf` per meaningful user action, `key=value` format:
```
login uid=1 method=google name="Сергей Шумилов" new=false
logout uid=1
open-blur uid=1
```
HTTP request logging is handled by the middleware — don't add per-route request logging.

## Environment variables

| Variable | Default | Notes |
|---|---|---|
| `REGION` | — | `ru` \| `com`, required; set from deploy matrix |
| `AUTH_URL` | — | Public auth-center URL (browser-facing) |
| `AUTH_INTERNAL` | — | Internal auth-center URL (server-to-server) |
| `APP_URL` | — | Public URL of this app |
| `APP_TOKEN` | — | Must be in auth-center's `APP_TOKENS` |
| `SECRET_KEY` | `dev-secret` | JWT signing key — always set in prod |
| `DB_PATH` | `menu.db` | SQLite file path |
| `PORT` | `8890` | HTTP listen port |
