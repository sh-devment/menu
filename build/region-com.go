package main

// The com zone. See region.go for what a region holds; every string here has a
// twin in region-ru.go.
var regionCOM = regionDef{
	Domain: "sh-development.com",
	Lang:   "en",
	Text: pageText{
		Login: "log in",
		About: "The central menu of the sh-development ecosystem. Single sign-on — open any app without logging in again.",

		TabApps: "apps",
		TabInfo: "info",
		Readme:  "readme",
		Logout:  "log out",

		AppsSub: "apps",
		OpenApp: "open",
		AppInfo: "more about",

		InfoTitle:   "about",
		InfoHello:   "Hi! I'm <strong>Shumilov Sergey</strong>, and this is my vision of a web app ecosystem.",
		InfoContact: "There's no dedicated support app yet. If you have advice or found a bug — write to me:",
		InfoMore:    "More about me —",

		Profile: "profile",
		Close:   "close",

		JS: jsText{
			EmailCopied: "Email copied to clipboard",
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
}
