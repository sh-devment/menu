package main

// region.go — the ONLY place where the ru / com zones differ.
//
// Both profiles are compiled into the binary; REGION (passed by the deploy
// matrix) picks one at startup. Anything zone-specific — domain, links, texts,
// per-app content — belongs here and nowhere else, so this file can be
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
	GotIt   string // status modal button

	JS jsText // strings used by app.js
}

// jsText is rendered into the page as JSON and read by app.js.
type jsText struct {
	UnavailableTitle string `json:"unavailableTitle"` // popup when an app didn't open within 2s
	UnavailableText  string `json:"unavailableText"`
	EmailCopied      string `json:"emailCopied"`
}

var regions = map[string]regionDef{
	"ru": {
		Domain: "sh-development.ru",
		Lang:   "ru",
		Text: pageText{
			Login: "войти",
			About: "Центральное меню экосистемы sh-development. Единая авторизация — открывайте любое приложение без повторного входа.",

			AppsSub: "приложения",
			OpenApp: "открыть",
			AppInfo: "подробнее о",

			InfoTitle:   "о проекте",
			InfoHello:   "Привет! Меня зовут <strong>Сергей Шумилов</strong>, и это моё видение экосистемы веб-приложений.",
			InfoContact: "Отдельного приложения для поддержки пока нет. Если хотите дать совет или сообщить об ошибке — напишите мне:",
			InfoMore:    "Больше обо мне —",

			Profile: "профиль",
			Close:   "закрыть",
			GotIt:   "понятно",

			JS: jsText{
				UnavailableTitle: "Приложение недоступно",
				UnavailableText:  "Приложение временно недоступно, попробуйте позже.",
				EmailCopied:      "Email скопирован в буфер обмена",
			},
		},
		Apps: map[string]appText{
			"nom-nom": {
				Desc: "Это трекер для колорий и веса",
				Features: []string{
					"Ежедневная статистика прогресса",
					"Учет и калорий по блюдам",
					"AI анализ еды по фото",
				},
			},
			"wgetbash": {
				Desc: "Хранилище для bash скриптов",
				Features: []string{
					"Доставка до сервера в один клик",
					"Группы и быстрый поиск по скриптам",
					"Встроенный просмотр логов",
				},
			},
			"blur": {
				Desc: "Плеер для длинных аудио — книг, подкастов и лекций",
				Features: []string{
					"Удобно выбирать время воспроизведения клавиатурой",
					"Плеер запоминает где вы остановились, даже если приложение закрыто",
					"Можно отключить автовоспроизведение, чтоб плеер сам остановился",
				},
			},
			"qcode": {
				Desc: "Редактор для создания красивых qr codes",
				Features: []string{
					"Есть интеграция с AI",
					"Огромное разнообразие параметров, которые можно изменить",
					"Это бесплатно!",
				},
			},
		},
	},
	"com": {
		Domain: "sh-development.com",
		Lang:   "en",
		Text: pageText{
			Login: "log in",
			About: "The central menu of the sh-development ecosystem. Single sign-on — open any app without logging in again.",

			AppsSub: "apps",
			OpenApp: "open",
			AppInfo: "more about",

			InfoTitle:   "about",
			InfoHello:   "Hi! I'm <strong>Shumilov Sergey</strong>, and this is my vision of a web app ecosystem.",
			InfoContact: "There's no dedicated support app yet. If you have advice or found a bug — write to me:",
			InfoMore:    "More about me —",

			Profile: "profile",
			Close:   "close",
			GotIt:   "got it",

			JS: jsText{
				UnavailableTitle: "App unavailable",
				UnavailableText:  "The app is temporarily unavailable, please try again later.",
				EmailCopied:      "Email copied to clipboard",
			},
		},
		Apps: map[string]appText{
			"nom-nom": {
				Desc: "A calorie and weight tracker",
				Features: []string{
					"Daily progress stats",
					"Calorie tracking per dish",
					"AI food analysis from a photo",
				},
			},
			"wgetbash": {
				Desc: "A store for bash scripts",
				Features: []string{
					"One-click delivery to a server",
					"Groups and fast script search",
					"Built-in log viewer",
				},
			},
			"blur": {
				Desc: "A player for long audio — books, podcasts and lectures",
				Features: []string{
					"Easy keyboard seeking",
					"Remembers where you stopped, even after the app is closed",
					"Autoplay can be turned off so the player stops on its own",
				},
			},
			"qcode": {
				Desc: "An editor for beautiful QR codes",
				Features: []string{
					"AI integration",
					"A huge variety of adjustable parameters",
					"It's free!",
				},
			},
		},
	},
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
