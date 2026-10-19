package mangadotnetscraper

import "testing"

func TestLanguageName(t *testing.T) {
	testCases := []struct {
		code     string
		expected string
	}{
		{"en", "English"},
		{"fr", "French"},
		{"es-la", "Spanish (Latin America)"},
		{"pt-br", "Portuguese (Brazil)"},
		{"zh-hk", "Chinese (Hong Kong)"},
		{"jp", "jp"}, // unknown codes render as-is
		{"", ""},
	}

	for _, testCase := range testCases {
		if got := languageName(testCase.code); got != testCase.expected {
			t.Errorf(
				"languageName(%q) = %q, want %q",
				testCase.code,
				got,
				testCase.expected,
			)
		}
	}
}

func TestScanPathLabel(t *testing.T) {
	testCases := []struct {
		name      string
		count     int
		languages []string
		expected  string
	}{
		{
			name:      "K Manga",
			count:     1,
			languages: []string{"English"},
			expected:  "K Manga (1 chapter, English)",
		},
		{
			name:     "Bored College Students",
			count:    0,
			expected: "Bored College Students (0 chapters)",
		},
		{
			name:      "Anime-Sama",
			count:     202,
			languages: []string{"French"},
			expected:  "Anime-Sama (202 chapters, French)",
		},
		{
			name:  "MANGA MILLION",
			count: 3519,
			// Aggregator groups: the label shows the three
			// first languages at most.
			languages: []string{
				"Arabic", "Bulgarian", "Burmese", "Chinese",
				"Czech", "Danish", "Dutch", "English", "French",
			},
			expected: "MANGA MILLION (3519 chapters, Arabic/Bulgarian/Burmese/…)",
		},
	}

	for _, testCase := range testCases {
		if got := scanPathLabel(
			testCase.name,
			testCase.count,
			testCase.languages,
		); got != testCase.expected {
			t.Errorf(
				"scanPathLabel(%q) = %q, want %q",
				testCase.name,
				got,
				testCase.expected,
			)
		}
	}
}
