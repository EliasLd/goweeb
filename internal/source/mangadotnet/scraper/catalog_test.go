package mangadotnetscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSearchCatalog(t *testing.T) {
	var (
		gotPath   string
		gotParams url.Values
	)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotParams = r.URL.Query()

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
				"manga_list": [
					{
						"id": 101,
						"title": "Blue Lock",
						"is_adult": 0,
						"chapter_count": 383,
						"status": "Ongoing"
					},
					{
						"id": 2836,
						"title": "Ya Boy Kongming!",
						"is_adult": 0,
						"chapter_count": 184,
						"status": "Ongoing"
					},
					{
						"id": 9999,
						"title": "Doujinshi",
						"is_adult": 1
					},
					{
						"id": 0,
						"title": "Zero ID"
					},
					{
						"id": 8888,
						"title": "  "
					}
				],
				"pagination": {
					"page": 1,
					"total_items": 5
				},
				"search_engine": "typesense"
			}`)
		},
	))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "  blue lock  ", nil)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if gotPath != searchPath {
		t.Errorf("expected path %q, got %q", searchPath, gotPath)
	}

	if got := gotParams.Get("search"); got != "blue lock" {
		t.Errorf("unexpected search parameter: %q", got)
	}

	if got := gotParams.Get("page"); got != "" {
		t.Errorf("unexpected page parameter: %q", got)
	}

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 results, got %d (%v)",
			len(results),
			results,
		)
	}

	if results[0].ID != "101" || results[0].Title != "Blue Lock" {
		t.Errorf(
			"unexpected first result: %q / %q",
			results[0].ID,
			results[0].Title,
		)
	}

	if results[1].ID != "2836" ||
		results[1].Title != "Ya Boy Kongming!" {
		t.Errorf(
			"unexpected second result: %q / %q",
			results[1].ID,
			results[1].Title,
		)
	}
}

func TestSearchCatalogEmptyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("no request expected for an empty query")
		},
	))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "  ", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
}

func TestSearchCatalogHTTPError(t *testing.T) {
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

	_, err := SearchCatalog(server.URL, "blue lock", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 503")
	}
}

func TestSearchCatalogInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")

			_, _ = fmt.Fprint(w, "<html>unexpected response</html>")
		},
	))
	defer server.Close()

	_, err := SearchCatalog(server.URL, "blue lock", nil)
	if err == nil {
		t.Fatal("expected a decode error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "invalid character '<'") {
		t.Errorf("unexpected decode error: %v", err)
	}
}

func TestMangaIDFromURL(t *testing.T) {
	cases := []struct {
		name     string
		workURL  string
		expected string
		wantErr  bool
	}{
		{
			name:     "manga URL",
			workURL:  "https://mangadot.net/manga/101",
			expected: "101",
		},
		{
			name:     "manga URL with trailing slash",
			workURL:  "https://mangadot.net/manga/101/",
			expected: "101",
		},
		{
			name:     "manga URL with custom domain",
			workURL:  "https://mirror.example.org/manga/2836",
			expected: "2836",
		},
		{
			name:    "URL without manga prefix",
			workURL: "https://mangadot.net/101",
			wantErr: true,
		},
		{
			name:    "URL with extra path segments",
			workURL: "https://mangadot.net/manga/101/chapters",
			wantErr: true,
		},
		{
			name:    "URL without manga ID",
			workURL: "https://mangadot.net/manga/",
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
