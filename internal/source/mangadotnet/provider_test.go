package mangadotnet

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
			expected: "https://mangadot.net",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://mangadot.net",
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

	if provider.Name() != "mangadotnet" {
		t.Errorf(
			"expected name mangadotnet, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{
				"manga_list": [
					{
						"id": 101,
						"title": "Blue Lock",
						"is_adult": 0
					},
					{
						"id": 9999,
						"title": "Doujinshi",
						"is_adult": 1
					},
					{
						"id": 2836,
						"title": "Ya Boy Kongming!",
						"is_adult": 0
					}
				]
			}`)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	results, err := provider.Search("blue lock", nil)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Blue Lock" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}

	expectedURL := server.URL + "/manga/101"

	if results[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, results[0].URL)
	}

	if results[1].Title != "Ya Boy Kongming!" {
		t.Errorf("unexpected title: %q", results[1].Title)
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

	_, err := provider.Search("blue lock", nil)
	if err == nil {
		t.Fatal("expected an error from the scraper")
	}
}
