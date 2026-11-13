package mangataro

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	cases := []struct {
		name         string
		customDomain string
		expected     string
	}{
		{
			name:     "default domain",
			expected: "https://mangataro.org",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://mangataro.org",
		},
		{
			name:         "custom domain is trimmed",
			customDomain: "  https://mirror.example.org  ",
			expected:     "https://mirror.example.org",
		},
		{
			name:         "trailing slash is trimmed",
			customDomain: "https://mirror.example.org/",
			expected:     "https://mirror.example.org",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := New(testCase.customDomain)

			if provider.domain != testCase.expected {
				t.Errorf(
					"expected domain %q, got %q",
					testCase.expected,
					provider.domain,
				)
			}
		})
	}
}

func TestName(t *testing.T) {
	provider := New("")

	if provider.Name() != "mangataro" {
		t.Errorf(
			"expected name mangataro, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `[
				{"id":"32148","title":"Berserk","url":"https://mangataro.org/manga/berserk","cover":"https://mangataro.org/content/media/berserk.jpg"},
				{"id":"635694","title":"The Berserker&#039;s Second Playthrough","url":"https://mangataro.org/manga/the-berserker-039-s-second-playthrough","cover":""}
			]`)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	results, err := provider.Search("berserk", nil)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Berserk" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}

	if results[0].URL != "https://mangataro.org/manga/berserk" {
		t.Errorf("unexpected url: %q", results[0].URL)
	}

	if results[0].CoverURL != "https://mangataro.org/content/media/berserk.jpg" {
		t.Errorf("unexpected cover: %q", results[0].CoverURL)
	}

	// The catalog payloads embed HTML entities in their titles.
	if results[1].Title != "The Berserker's Second Playthrough" {
		t.Errorf("expected the entity-decoded title, got %q", results[1].Title)
	}
}

func TestSearchErrorPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"service unavailable",
				http.StatusServiceUnavailable,
			)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	_, err := provider.Search("berserk", nil)
	if err == nil {
		t.Fatal("expected an error from the scraper")
	}
}

func TestListScanPathsNotImplementedYet(t *testing.T) {
	provider := New("")

	_, err := provider.ListScanPaths("https://mangataro.org/manga/berserk", nil)
	if err == nil {
		t.Fatal("expected the stub to reject the call")
	}
}

func TestListEntriesNotImplementedYet(t *testing.T) {
	provider := New("")

	_, _, err := provider.ListEntries(
		"https://mangataro.org/manga/berserk",
		"en",
		nil,
	)
	if err == nil {
		t.Fatal("expected the stub to reject the call")
	}
}

func TestGetPageImageURLsNotImplementedYet(t *testing.T) {
	provider := New("")

	_, err := provider.GetPageImageURLs(
		"https://mangataro.org/read/berserk/ch398-471959",
		nil,
	)
	if err == nil {
		t.Fatal("expected the stub to reject the call")
	}
}
