package toonkotsuscraper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

func testDiscardLogger() *logger.Logger {
	return logger.New(io.Discard, logger.LevelInfo)
}

// newSearchServer serves one JSON body per /cdn/v1/browse page and
// records the received queries in request order.
func newSearchServer(
	t *testing.T,
	pages map[int]string,
	queries *[]url.Values,
) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			parsed, err := url.ParseRequestURI(r.RequestURI)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			if queries != nil {
				*queries = append(*queries, parsed.Query())
			}

			w.Header().Set("Content-Type", "application/json")

			page := 1
			fmt.Sscanf(parsed.Query().Get("page"), "%d", &page)

			_, _ = fmt.Fprint(w, pages[page])
		},
	))
	t.Cleanup(server.Close)

	return server
}

func TestSearchCatalogBuildsSiteURLs(t *testing.T) {
	server := newSearchServer(t, map[int]string{
		1: `{"items":[` +
			`{"code":"bm0ssx","slug":"jujutsu-kaisen","title":"Jujutsu Kaisen","cover":"/cdn/c/jjk.webp"},` +
			`{"code":"z9k1","slug":"berserk","title":"Berserk","cover":"https://mirror.test/x.webp"}]` +
			`,"page":1,"limit":100,"total":2,"hasNextPage":false}`,
	}, nil)

	results, err := SearchCatalog(
		server.URL,
		"jujutsu",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].URL != server.URL+"/title/bm0ssx-jujutsu-kaisen" {
		t.Errorf("unexpected work URL: %q", results[0].URL)
	}

	// Covers travel as same-origin paths and full URLs alike.
	if results[0].CoverURL != server.URL+"/cdn/c/jjk.webp" {
		t.Errorf("unexpected cover: %q", results[0].CoverURL)
	}

	if results[1].CoverURL != "https://mirror.test/x.webp" {
		t.Errorf("unexpected cover: %q", results[1].CoverURL)
	}
}

func TestSearchCatalogSendsSearchFiltersAndWalksPages(t *testing.T) {
	var queries []url.Values

	server := newSearchServer(t, map[int]string{
		1: `{"items":[{"code":"a","slug":"aa","title":"AA"}],"hasNextPage":true}`,
		2: `{"items":[{"code":"b","slug":"bb","title":"BB"}],"hasNextPage":false}`,
	}, &queries)

	results, err := SearchCatalog(
		server.URL,
		"one piece",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results across pages, got %d", len(results))
	}

	if len(queries) != 2 {
		t.Fatalf("expected 2 walked pages, got %d requests", len(queries))
	}

	first := queries[0]

	if first.Get("q") != "one piece" {
		t.Errorf("query param q = %q", first.Get("q"))
	}

	// Adult works are filtered out on every page.
	if first.Get("sfw") != sfwFilter {
		t.Errorf("query param sfw = %q", first.Get("sfw"))
	}

	if first.Get("limit") != "100" {
		t.Errorf("query param limit = %q", first.Get("limit"))
	}

	if queries[1].Get("page") != "2" {
		t.Errorf("second request page = %q", queries[1].Get("page"))
	}
}

func TestSearchCatalogStopsWhenPagesRepeat(t *testing.T) {
	var queries []url.Values

	server := newSearchServer(t, map[int]string{
		1: `{"items":[{"code":"a","slug":"aa","title":"AA"}],"hasNextPage":true}`,
		2: `{"items":[{"code":"a","slug":"aa","title":"AA"}],"hasNextPage":true}`,
	}, &queries)

	results, err := SearchCatalog(
		server.URL,
		"same",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	// A non-advancing page must not extend the results with its
	// repeated rows.
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSearchCatalogEmptyMatch(t *testing.T) {
	server := newSearchServer(t, map[int]string{
		1: `{"items":[],"total":0,"hasNextPage":false}`,
	}, nil)

	results, err := SearchCatalog(
		server.URL,
		"nothing",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestSearchCatalogSkipsUnusableRows(t *testing.T) {
	server := newSearchServer(t, map[int]string{
		1: `{"items":[` +
			`{"code":"","slug":"aa","title":"No code"},` +
			`{"code":"b","slug":"","title":"No slug"},` +
			`{"code":"c","slug":"cc","title":"  "},` +
			`{"code":"d","slug":"dd","title":"OK"}]` +
			`,"hasNextPage":false}`,
	}, nil)

	results, err := SearchCatalog(
		server.URL,
		"mixed",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 1 || results[0].Title != "OK" {
		t.Errorf("expected only the usable row, got %+v", results)
	}
}

func TestSearchCatalogStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusGone)
		},
	))
	defer server.Close()

	_, err := SearchCatalog(server.URL, "q", testDiscardLogger())
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "status 410") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}
