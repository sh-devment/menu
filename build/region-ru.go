package main

// The ru zone. See region.go for what a region holds; every string here has a
// twin in region-com.go.
var regionRU = regionDef{
	Domain: "sh-development.ru",
	Lang:   "ru",
	Text: pageText{
		Login: "войти",
		About: "Центральное меню экосистемы sh-development. Единая авторизация — открывайте любое приложение без повторного входа.",

		TabApps: "apps",
		TabInfo: "info",
		Readme:  "readme",
		Logout:  "log out",

		AppsSub: "приложения",
		OpenApp: "открыть",
		AppInfo: "подробнее о",

		InfoTitle:   "о проекте",
		InfoHello:   "Привет! Меня зовут <strong>Сергей Шумилов</strong>, и это моё видение экосистемы веб-приложений.",
		InfoContact: "Отдельного приложения для поддержки пока нет. Если хотите дать совет или сообщить об ошибке — напишите мне:",
		InfoMore:    "Больше обо мне —",

		Profile: "профиль",
		Close:   "закрыть",

		JS: jsText{
			EmailCopied: "Email скопирован в буфер обмена",
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
}
