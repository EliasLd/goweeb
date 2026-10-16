package mangadotnet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestProvider(
	t *testing.T,
	handlers map[string]http.HandlerFunc,
) *Provider {
	t.Helper()

	mux := http.NewServeMux()

	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return New(server.URL)
}

func jsonResponse(
	w http.ResponseWriter,
	body string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	_, _ = fmt.Fprint(w, body)
}

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

func TestListScanPaths(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/101/chapters/filter-options": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			jsonResponse(w, `{
				"languages": ["en", "es", "fr"],
				"groups": [
					{
						"id": "18210",
						"name": "K Manga",
						"slug": "k-manga",
						"is_scanlator": false
					},
					{
						"id": "scanlator:Stick",
						"name": "Stick",
						"slug": null,
						"is_scanlator": false
					}
				]
			}`)
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/manga/101",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf(
			"expected 2 scan paths, got %d",
			len(items),
		)
	}

	if items[0].Label != "K Manga" || items[0].Value != "18210" {
		t.Errorf(
			"unexpected first scan path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Label != "Stick" ||
		items[1].Value != "scanlator:Stick" {
		t.Errorf(
			"unexpected second scan path: %q / %q",
			items[1].Label,
			items[1].Value,
		)
	}
}

func TestListScanPathsInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, err := provider.ListScanPaths(
		provider.domain+"/not-manga/101",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}
