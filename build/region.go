package main

// region.go — the type and the loader of the ru / com zone profiles; the
// profiles themselves live in region-ru.go and region-com.go. These three files
// are the ONLY place where the zones differ.
//
// Both profiles are compiled into the binary; REGION (passed by the deploy
// matrix) picks one at startup. Anything zone-specific — domain, links, texts,
// per-app content — belongs here and nowhere else, so the three files can be
// reviewed as a whole. Infrastructure (auth URLs, tokens, port) stays in env.
// Technical text (logs, http.Error bodies) is English in both zones and does
// not live here.

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"reflect"
)

// regionDef is everything that differs between zones.
type regionDef struct {
	Domain string             // main domain: base for app links and the site link
	Lang   string             // <html lang="…">
	Text   pageText           // every user-facing string of the page
	Apps   map[string]appText // slug → info-modal content; every app in `apps` needs an entry
}

// appText is the per-zone content of an app's info modal.
type appText struct {
	Desc     string   // short description at the top of the modal
	Features []string // bullet list under Desc; empty hides the list
}

// pageText holds the page strings. Fields, not a map: a typo in the template
// fails at render instead of silently printing nothing.
type pageText struct {
	// login screen
	Login string // login button
	About string // text under the login card

	// header (ru keeps these English on purpose)
	TabApps string
	TabInfo string
	Readme  string
	Logout  string

	// apps tab
	AppsSub string // subtitle under "sh-development"
	OpenApp string // aria-label prefix: "<OpenApp> <app name>"
	AppInfo string // aria-label prefix of the (i) button: "<AppInfo> <app name>"

	// info tab
	InfoTitle   string
	InfoHello   template.HTML // may contain <strong>
	InfoContact string
	InfoMore    string // before the domain link at the bottom: "<InfoMore> sh-development.xx"

	// chrome
	Profile string // aria-label of the profile button
	Close   string // aria-label of the modal close button

	JS jsText // strings used by app.js
}

// jsText is rendered into the page as JSON and read by app.js.
type jsText struct {
	EmailCopied string `json:"emailCopied"`
}

var regions = map[string]regionDef{
	"ru":  regionRU,
	"com": regionCOM,
}

// Active profile, selected from REGION by initRegion.
var (
	regionName string
	region     regionDef
)

// initRegion selects the profile from REGION and fills every app's URL as
// https://<Sub>.<Domain> plus its Desc/Features. Unknown or empty REGION, or
// a string missing in any zone, is fatal: better a failed health check and
// rollback than a grid of dead links or blank text.
func initRegion() {
	for name, r := range regions {
		if err := checkRegion(r); err != nil {
			log.Fatalf("region %s: %v", name, err)
		}
	}

	regionName = os.Getenv("REGION")
	r, ok := regions[regionName]
	if !ok {
		log.Fatalf("unknown REGION=%q (want one of: ru, com)", regionName)
	}
	region = r

	for i := range apps {
		t := region.Apps[apps[i].Slug]
		apps[i].URL = "https://" + apps[i].Sub + "." + region.Domain
		apps[i].Desc = t.Desc
		apps[i].Features = t.Features
	}
	log.Printf("region name=%s domain=%s apps=%d", regionName, region.Domain, len(apps))
}

// checkRegion reports the first empty string in a profile — e.g. a text added
// to ru but forgotten in com — and any app without content.
func checkRegion(r regionDef) error {
	if err := checkFilled(reflect.ValueOf(r), "regionDef"); err != nil {
		return err
	}
	for _, a := range apps {
		t, ok := r.Apps[a.Slug]
		if !ok {
			return fmt.Errorf("Apps[%q] missing", a.Slug)
		}
		if err := checkFilled(reflect.ValueOf(t), "Apps["+a.Slug+"]"); err != nil {
			return err
		}
	}
	for slug := range r.Apps {
		if appBySlug(slug) == nil {
			return fmt.Errorf("Apps[%q] has no app in main.go", slug)
		}
	}
	return nil
}

// checkFilled walks structs and slices and fails on the first empty string.
func checkFilled(v reflect.Value, path string) error {
	switch v.Kind() {
	case reflect.String:
		if v.Len() == 0 {
			return fmt.Errorf("%s is empty", path)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := checkFilled(v.Field(i), path+"."+v.Type().Field(i).Name); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if err := checkFilled(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
