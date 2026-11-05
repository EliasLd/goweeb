package mangaballscraper

// languageNames maps the language codes observed in MangaBall payloads
// to their display names. Codes missing from the map fall back to the
// code itself, so a new language still gets a usable label.
var languageNames = map[string]string{
	"ar":    "Arabic",
	"bg":    "Bulgarian",
	"cs":    "Czech",
	"da":    "Danish",
	"de":    "German",
	"el":    "Greek",
	"en":    "English",
	"es":    "Spanish",
	"es-la": "Spanish (Latin America)",
	"fi":    "Finnish",
	"fr":    "French",
	"he":    "Hebrew",
	"hi":    "Hindi",
	"hu":    "Hungarian",
	"id":    "Indonesian",
	"it":    "Italian",
	"ja":    "Japanese",
	"ka":    "Georgian",
	"ko":    "Korean",
	"la":    "Latin",
	"lt":    "Lithuanian",
	"mn":    "Mongolian",
	"ms":    "Malay",
	"my":    "Burmese",
	"nl":    "Dutch",
	"no":    "Norwegian",
	"pl":    "Polish",
	"pt":    "Portuguese",
	"pt-br": "Portuguese (Brazil)",
	"ro":    "Romanian",
	"ru":    "Russian",
	"sv":    "Swedish",
	"th":    "Thai",
	"tl":    "Tagalog",
	"tr":    "Turkish",
	"uk":    "Ukrainian",
	"vi":    "Vietnamese",
	"zh":    "Chinese",
	"zh-hk": "Chinese (Hong Kong)",
}

func languageName(code string) string {
	if name, ok := languageNames[code]; ok {
		return name
	}

	return code
}
