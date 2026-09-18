package mangadex

import "fmt"

var languageNames = map[string]string{
	"ar":    "Arabic",
	"bn":    "Bengali",
	"cs":    "Czech",
	"de":    "German",
	"en":    "English",
	"es":    "Spanish",
	"es-la": "Spanish (Latin America)",
	"fr":    "French",
	"hu":    "Hungarian",
	"id":    "Indonesian",
	"it":    "Italian",
	"ja":    "Japanese",
	"ja-ro": "Japanese (Romanized)",
	"kk":    "Kazakh",
	"ko":    "Korean",
	"ko-ro": "Korean (Romanized)",
	"pl":    "Polish",
	"pt":    "Portuguese",
	"pt-br": "Portuguese (Brazil)",
	"ru":    "Russian",
	"th":    "Thai",
	"tr":    "Turkish",
	"uk":    "Ukrainian",
	"uz":    "Uzbek",
	"vi":    "Vietnamese",
	"zh":    "Chinese (Simplified)",
	"zh-hk": "Chinese (Traditional)",
}

func languageLabel(code string) string {
	if name, ok := languageNames[code]; ok {
		return fmt.Sprintf("%s (%s)", name, code)
	}

	return code
}
