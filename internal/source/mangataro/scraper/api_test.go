package mangataroscraper

import (
	"strings"
	"testing"
	"time"
)

func TestMangaSlugFromURL(t *testing.T) {
	cases := []struct {
		name     string
		workURL  string
		expected string
		wantErr  bool
	}{
		{
			name:     "plain title URL",
			workURL:  "https://mangataro.org/manga/berserk",
			expected: "berserk",
		},
		{
			name:     "trailing slash",
			workURL:  "https://mangataro.org/manga/berserk/",
			expected: "berserk",
		},
		{
			name:     "slugged title",
			workURL:  "https://mangataro.org/manga/go-with-the-clouds-north-by-northwest",
			expected: "go-with-the-clouds-north-by-northwest",
		},
		{
			name:    "unexpected path",
			workURL: "https://mangataro.org/title/berserk",
			wantErr: true,
		},
		{
			name:    "empty slug",
			workURL: "https://mangataro.org/manga/",
			wantErr: true,
		},
		{
			name:    "unreadable url",
			workURL: "ht!tp://mangataro.org/manga/berserk",
			wantErr: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			slug, err := MangaSlugFromURL(testCase.workURL)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", testCase.workURL)
				}

				return
			}

			if err != nil {
				t.Fatalf("MangaSlugFromURL() error: %v", err)
			}

			if slug != testCase.expected {
				t.Errorf("expected slug %q, got %q", testCase.expected, slug)
			}
		})
	}
}

func TestChapterRequestToken(t *testing.T) {
	token, timestamp := chapterRequestToken()

	before := time.Now().Unix()

	if len(token) != 16 {
		t.Errorf("token = %q, want 16 hex characters", token)
	}

	if timestamp < before-1 || timestamp > before {
		t.Errorf("timestamp = %d, want close to now (%d)", timestamp, before)
	}

	// The site's own check is case-sensitive hexadecimal; a
	// different case or a wrong length is a guaranteed refusal.
	for _, r := range token {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Errorf("token %q carries the non-lowercase-hex character %q", token, r)
		}
	}
}

func TestParseBaseDomain(t *testing.T) {
	cases := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{name: "plain", domain: "https://mangataro.org"},
		{name: "trailing slash", domain: "https://mangataro.org/"},
		{name: "whitespace", domain: "  https://mangataro.org  "},
		{name: "missing scheme", domain: "mangataro.org", wantErr: true},
		{name: "missing host", domain: "https://", wantErr: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			base, err := parseBaseDomain(testCase.domain)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", testCase.domain)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseBaseDomain() error: %v", err)
			}

			if base.Scheme != "https" || base.Host != "mangataro.org" {
				t.Errorf("unexpected base: %q", base)
			}
		})
	}
}
