package atsumoescraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSearchCatalog(t *testing.T) {
	var gotParams url.Values

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != searchPath {
				t.Errorf(
					"unexpected path: %s",
					r.URL.Path,
				)
			}

			gotParams = r.URL.Query()

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"found": 3,
				"hits": [
					{
						"document": {
							"id": "jIg3F",
							"title": "Solanin",
							"englishTitle": "Solanin",
							"isAdult": false
						}
					},
					{
						"document": {
							"id": "fallback",
							"title": "",
							"englishTitle": "English title",
							"isAdult": false
						}
					},
					{
						"document": {
							"id": "adult",
							"title": "Adult title",
							"isAdult": true
						}
					}
				]
			}`)
		},
	))
	defer server.Close()

	results, err := SearchCatalog(
		server.URL,
		"  solanin  ",
		nil,
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf(
			"expected 2 results, got %d",
			len(results),
		)
	}

	if results[0].Title != "Solanin" {
		t.Errorf(
			"unexpected title: %q",
			results[0].Title,
		)
	}

	if results[0].URL != server.URL+"/manga/jIg3F" {
		t.Errorf(
			"unexpected URL: %q",
			results[0].URL,
		)
	}

	if results[1].Title != "English title" {
		t.Errorf(
			"expected English title fallback, got %q",
			results[1].Title,
		)
	}

	if got := gotParams.Get("q"); got != "solanin" {
		t.Errorf("unexpected query: %q", got)
	}

	if got := gotParams.Get("page"); got != "1" {
		t.Errorf("unexpected page: %q", got)
	}

	if got := gotParams.Get("per_page"); got != "40" {
		t.Errorf("unexpected per_page: %q", got)
	}

	if got := gotParams.Get("query_by"); got !=
		"title,englishTitle,otherNames,authors,acronyms" {
		t.Errorf("unexpected query_by: %q", got)
	}

	if got := gotParams.Get("filter_by"); got !=
		"isAdult:=false && views:>0 && hidden:!=true" {
		t.Errorf("unexpected filter_by: %q", got)
	}
}

func TestSearchCatalogEmptyQuery(t *testing.T) {
	results, err := SearchCatalog(
		"invalid-domain",
		"  ",
		nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Fatalf(
			"expected no results, got %d",
			len(results),
		)
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

	_, err := SearchCatalog(server.URL, "solanin", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 503")
	}
}
