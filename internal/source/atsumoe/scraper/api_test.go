package atsumoescraper

import (
	"testing"
)

func TestMangaIDFromURL(t *testing.T) {
	cases := []struct {
		name     string
		workURL  string
		expected string
		wantErr  bool
	}{
		{
			name:     "manga URL",
			workURL:  "https://atsu.moe/manga/jIg3F",
			expected: "jIg3F",
		},
		{
			name:     "manga URL with trailing slash",
			workURL:  "https://atsu.moe/manga/jIg3F/",
			expected: "jIg3F",
		},
		{
			name:     "manga URL with custom domain",
			workURL:  "https://mirror.example.org/manga/CM0wz",
			expected: "CM0wz",
		},
		{
			name:    "URL without manga prefix",
			workURL: "https://atsu.moe/jIg3F",
			wantErr: true,
		},
		{
			name:    "URL with extra path segments",
			workURL: "https://atsu.moe/manga/jIg3F/chapters",
			wantErr: true,
		},
		{
			name:    "URL without manga ID",
			workURL: "https://atsu.moe/manga/",
			wantErr: true,
		},
		{
			name:    "empty URL",
			workURL: "  ",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			workURL: "://not-a-url",
			wantErr: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mangaID, err := MangaIDFromURL(testCase.workURL)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf(
						"expected an error for %q",
						testCase.workURL,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"MangaIDFromURL(%q) error: %v",
					testCase.workURL,
					err,
				)
			}

			if mangaID != testCase.expected {
				t.Errorf(
					"expected manga ID %q, got %q",
					testCase.expected,
					mangaID,
				)
			}
		})
	}
}
