package mangadotnetscraper

// languageNames maps the language codes observed in mangadot.net
// payloads to their display names.
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

// languageName returns the display name of a language code; unknown
// codes render as-is.
func languageName(code string) string {
	if name, ok := languageNames[code]; ok {
		return name
	}

	return code
}
