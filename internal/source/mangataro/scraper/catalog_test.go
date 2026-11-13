package mangataroscraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

func rowJSON(id string, title string, url string) string {
	return fmt.Sprintf(
		`{"id":%q,"title":%q,"url":%q,"cover":"https://mangataro.org/content/media/%s.jpg"}`,
		id,
		title,
		url,
		id,
	)
}

func completeCatalogPage() []string {
	rows := make([]string, 0, catalogPageSize)

	for i := 0; i < catalogPageSize; i++ {
		rows = append(rows, rowJSON(
			fmt.Sprintf("id-%d", i),
			fmt.Sprintf("Manga %d", i),
			fmt.Sprintf("https://mangataro.org/manga/manga-%d", i),
		))
	}

	return rows
}

// captureCatalog holds what the fake catalog route received.
type captureCatalog struct {
	requests []catalogRequest
}

func (c *captureCatalog) handler(t *testing.T, pages [][]string) http.HandlerFunc {
	t.Helper()

	for _, page := range pages {
		if len(page) > catalogPageSize {
			t.Fatalf("test page carries %d rows, route answers at most %d",
				len(page), catalogPageSize)
		}
	}

	var cursor atomic.Int32

	return func(w http.ResponseWriter, r *http.Request) {
		index := int(cursor.Add(1)) - 1

		if index >= len(pages) {
			t.Errorf("unexpected extra catalog request #%d", index+1)

			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("catalog method = %s, want POST", r.Method)
		}

		if r.URL.Path != "/wp-json/manga/v1/load" {
			t.Errorf("catalog path = %q, want /wp-json/manga/v1/load", r.URL.Path)
		}

		var payload catalogRequest

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body %q: %v", body, err)
		}

		c.requests = append(c.requests, payload)

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, "["+strings.Join(pages[index], ",")+"]")
	}
}

func testDiscardLogger() *logger.Logger {
	return logger.New(io.Discard, logger.LevelError)
}

func TestSearchCatalogWalksPages(t *testing.T) {
	second := []string{
		rowJSON("100", "Berserk", "https://mangataro.org/manga/berserk"),
		rowJSON("101", "The Berserker&#039;s Second Playthrough", "https://mangataro.org/manga/berserker-2"),
	}

	capture := &captureCatalog{}
	server := httptest.NewServer(capture.handler(t, [][]string{
		completeCatalogPage(),
		second,
	}))
	defer server.Close()

	results, err := SearchCatalog(
		server.URL,
		"berserk",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != catalogPageSize+2 {
		t.Fatalf("expected %d results, got %d", catalogPageSize+2, len(results))
	}

	last := results[len(results)-1]

	if last.Title != "The Berserker's Second Playthrough" {
		t.Errorf("expected the entity-decoded title, got %q", last.Title)
	}

	if last.URL != "https://mangataro.org/manga/berserker-2" {
		t.Errorf("unexpected url: %q", last.URL)
	}

	if last.CoverURL != "https://mangataro.org/content/media/101.jpg" {
		t.Errorf("unexpected cover: %q", last.CoverURL)
	}

	if len(capture.requests) != 2 {
		t.Fatalf("sent %d catalog requests, want 2", len(capture.requests))
	}

	// The payload must mirror the site's own catalog POSTs: the
	// filters are JSON strings, the sort is the site default, and
	// the page walks forward.
	for i, request := range capture.requests {
		if request.Page != i+1 {
			t.Errorf("request %d page = %d", i, request.Page)
		}

		if request.Search != "berserk" {
			t.Errorf("request %d search = %q", i, request.Search)
		}

		if request.Years != "[]" || request.Genres != "[]" ||
			request.Types != "[]" || request.Statuses != "[]" {
			t.Errorf("request %d filters = %q/%q/%q/%q, want literal \"[]\"",
				i, request.Years, request.Genres, request.Types, request.Statuses)
		}

		if request.Sort != "popular" {
			t.Errorf("request %d sort = %q", i, request.Sort)
		}
	}
}

func TestSearchCatalogStopsWhenPageComesBackShort(t *testing.T) {
	rows := []string{
		rowJSON("1", "Berserk", "https://mangataro.org/manga/berserk"),
		rowJSON("2", "Dazzling Breath", "https://mangataro.org/manga/dazzling-breath"),
	}

	capture := &captureCatalog{}
	server := httptest.NewServer(capture.handler(t, [][]string{rows}))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "berserk", testDiscardLogger())
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if len(capture.requests) != 1 {
		t.Errorf("sent %d catalog requests, a short page must stop the walk", len(capture.requests))
	}
}

func TestSearchCatalogCapsAtMaxPages(t *testing.T) {
	page := completeCatalogPage()

	capture := &captureCatalog{}
	server := httptest.NewServer(capture.handler(t, [][]string{
		page, page, page,
	}))
	defer server.Close()

	if _, err := SearchCatalog(server.URL, "berserk", testDiscardLogger()); err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(capture.requests) != maxCatalogPages {
		t.Errorf("sent %d catalog requests, want the %d-page cap", len(capture.requests), maxCatalogPages)
	}
}

func TestSearchCatalogEmptyResult(t *testing.T) {
	capture := &captureCatalog{}
	server := httptest.NewServer(capture.handler(t, [][]string{{}}))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "zzzzqqqq", testDiscardLogger())
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestSearchCatalogSkipsIncompleteRows(t *testing.T) {
	rows := []string{
		rowJSON("1", "Berserk", "https://mangataro.org/manga/berserk"),
		`{"id":"","title":"No id","url":"https://mangataro.org/manga/no-id","cover":""}`,
		`{"id":"3","title":"","url":"https://mangataro.org/manga/no-title","cover":""}`,
		`{"id":"4","title":"No url","url":"","cover":""}`,
	}

	server := httptest.NewServer((&captureCatalog{}).handler(t, [][]string{rows}))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "berserk", testDiscardLogger())
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 1 || results[0].Title != "Berserk" {
		t.Errorf("expected the single complete row to survive, got %+v", results)
	}
}

func TestSearchCatalogHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		},
	))
	defer server.Close()

	_, err := SearchCatalog(server.URL, "berserk", testDiscardLogger())
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "status 503") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}

func TestSearchCatalogInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"code":`)
		},
	))
	defer server.Close()

	_, err := SearchCatalog(server.URL, "berserk", testDiscardLogger())
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "failed to decode MangaTaro catalog search") {
		t.Errorf("expected a decode error, got %v", err)
	}
}
