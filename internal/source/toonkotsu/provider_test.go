package toonkotsu

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
			expected: "https://toonkotsu.com",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://toonkotsu.com",
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

	if provider.Name() != "toonkotsu" {
		t.Errorf(
			"expected name toonkotsu, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{"items":[`+
				`{"code":"bm0ssx","slug":"jujutsu-kaisen","title":"Jujutsu Kaisen","cover":"/cdn/c/jjk.webp"},`+
				`{"code":"z9k1","slug":"berserk","title":"Berserk","cover":""}]`+
				`,"total":2,"hasNextPage":false}`)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	results, err := provider.Search("jujutsu", nil)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Jujutsu Kaisen" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}

	if results[0].URL != server.URL+"/title/bm0ssx-jujutsu-kaisen" {
		t.Errorf("unexpected url: %q", results[0].URL)
	}

	if results[0].CoverURL != server.URL+"/cdn/c/jjk.webp" {
		t.Errorf("unexpected cover: %q", results[0].CoverURL)
	}

	if results[1].CoverURL != "" {
		t.Errorf("unexpected cover: %q", results[1].CoverURL)
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

	_, err := provider.Search("jujutsu", nil)
	if err == nil {
		t.Fatal("expected an error from the scraper")
	}
}

func TestNotImplementedMethodsStayHonest(t *testing.T) {
	provider := New("")

	if _, err := provider.ListScanPaths("", nil); err == nil ||
		!strings.Contains(err.Error(), "not implemented") {
		t.Errorf("ListScanPaths() error: %v", err)
	}

	if _, _, err := provider.ListEntries("", "", nil); err == nil ||
		!strings.Contains(err.Error(), "not implemented") {
		t.Errorf("ListEntries() error: %v", err)
	}

	if _, err := provider.GetPageImageURLs("", nil); err == nil ||
		!strings.Contains(err.Error(), "not implemented") {
		t.Errorf("GetPageImageURLs() error: %v", err)
	}
}
