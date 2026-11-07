package metadata

import (
	"strings"
	"unicode"
)

// normalizeTitle reduces a display title to a comparable key:
// letters and digits kept (case-folded), punctuation ignored, and
// whitespace runs collapsed. Matching stays exact-equality on the
// result — never fuzzy, because a wrong cover is worse than no
// cover.
func normalizeTitle(title string) string {
	var builder strings.Builder

	builder.Grow(len(title))

	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))

		case unicode.IsSpace(r) || isSeparator(r):
			builder.WriteRune(' ')
		}
	}

	return strings.Join(strings.Fields(builder.String()), " ")
}

// isSeparator reports whether a punctuation rune acts as a title
// word separator worth normalizing to a space.
func isSeparator(r rune) bool {
	switch r {
	case '/', '-', ':', ',', '.', '&', '_', '(', ')', '[', ']':
		return true
	}

	return false
}

// LanguageISO renders a scan-path language code as the ComicInfo
// LanguageISO value (ISO 639 language plus an uppercase region when
// one travels with it, e.g. "pt-br" becomes "pt-BR"). Anything that
// is not shaped like a language code yields empty: display labels
// and provider-specific values are not language codes.
func LanguageISO(scanPath string) string {
	scanPath = strings.TrimSpace(scanPath)

	language, region, found := strings.Cut(
		strings.ToLower(scanPath),
		"-",
	)
	if len(language) < 2 || len(language) > 3 {
		return ""
	}

	for _, r := range language {
		if r < 'a' || r > 'z' {
			return ""
		}
	}

	if found {
		// Region or script subtag, e.g. "pt-br" or "zh-hk"/"zh-hans".
		if len(region) < 2 || len(region) > 4 {
			return ""
		}

		// Digit regions ("419") and script tags ("hans") stay as
		// decoded; only the classic two-letter regions go uppercase.
		if len(region) == 2 {
			region = strings.ToUpper(region)
		}
	}

	if !found {
		return language
	}

	return language + "-" + region
}
