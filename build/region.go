package main

// region.go — the ONLY place where the ru / com zones differ.
//
// Both profiles are compiled into the binary; REGION (passed by the deploy
// matrix) picks one at startup. Anything zone-specific — domain, links, texts,
// per-app content — belongs here and nowhere else, so this file can be
// reviewed as a whole. Infrastructure (auth URLs, tokens, port) stays in env.

import (
	"log"
	"os"
)

// regionDef is everything that differs between zones.
type regionDef struct {
	Domain string // main domain: base for app links and the site link
}

var regions = map[string]regionDef{
	"ru": {
		Domain: "sh-development.ru",
	},
	"com": {
		Domain: "sh-development.com",
	},
}

// Active profile, selected from REGION by initRegion.
var (
	regionName string
	region     regionDef
)

// initRegion selects the profile from REGION and fills every app's URL as
// https://<Sub>.<Domain>. Unknown or empty REGION is fatal: better a failed
// health check and rollback than a grid of dead links.
func initRegion() {
	regionName = os.Getenv("REGION")
	r, ok := regions[regionName]
	if !ok {
		log.Fatalf("unknown REGION=%q (want one of: ru, com)", regionName)
	}
	region = r

	for i := range apps {
		apps[i].URL = "https://" + apps[i].Sub + "." + region.Domain
	}
	log.Printf("region name=%s domain=%s apps=%d", regionName, region.Domain, len(apps))
}
