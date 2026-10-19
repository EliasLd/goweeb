package mangadotnetscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fixtureServer dispatches canned JSON payloads by API path.
func fixtureServer(
	filterOptions string,
	chapterList string,
) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			payload := filterOptions

			if strings.HasSuffix(
				r.URL.Path,
				"/chapters/list",
			) {
				if chapterList == "" {
					http.Error(
						w,
						"internal error",
						http.StatusInternalServerError,
					)

					return
				}

				payload = chapterList
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(
				w,
				payload,
			)
		},
	))

	return server
}

func TestListScanPaths(t *testing.T) {
	var (
		gotOptionsPath string
		gotListPath    string
	)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch {
			case strings.HasSuffix(r.URL.Path, "/filter-options"):
				gotOptionsPath = r.URL.Path

				_, _ = fmt.Fprint(w, `{
					"languages": ["en", "es", "fr"],
					"groups": [
						{
							"id": "18210",
							"name": "K Manga",
							"slug": "k-manga",
							"is_scanlator": false
						},
						{
							"id": "18192",
							"name": "LeerCapitulo",
							"slug": "leercapitulo",
							"is_scanlator": false
						},
						{
							"id": "scanlator:Stick",
							"name": "Stick",
							"slug": null,
							"is_scanlator": false
						},
						{
							"id": "99999",
							"name": "Side Group",
							"slug": null,
							"is_scanlator": false
						}
					]
				}`)

			case strings.HasSuffix(r.URL.Path, "/chapters/list"):
				gotListPath = r.URL.Path

				_, _ = fmt.Fprint(w, `[
					{
						"id": 1,
						"chapter_number": 1,
						"page_count": 20,
						"language": "en",
						"groups": [{"id": 18210, "name": "K Manga"}]
					},
					{
						"id": 2,
						"chapter_number": 2,
						"page_count": 20,
						"language": "fr",
						"groups": [
							{"id": 18210, "name": "K Manga"},
							{"id": 99999, "name": "Side Group"}
						]
					},
					{
						"id": 3,
						"chapter_number": 3,
						"page_count": 0,
						"language": "en",
						"groups": [{"id": 18210, "name": "K Manga"}]
					},
					{
						"id": 4,
						"chapter_number": "1.00",
						"page_count": 12,
						"language": "es",
						"groups": [{"id": 18192, "name": "LeerCapitulo"}]
					},
					{
						"id": 5,
						"chapter_number": 1,
						"page_count": 10,
						"language": "en",
						"scanlator_name": "Stick"
					},
					{
						"id": 6,
						"chapter_number": 7,
						"page_count": 15,
						"language": "ja",
						"groups": []
					}
				]`)

			default:
				http.Error(
					w,
					"not found",
					http.StatusNotFound,
				)
			}
		},
	))
	defer server.Close()

	items, err := ListScanPaths(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if gotOptionsPath != "/api/manga/101/chapters/filter-options" {
		t.Errorf("unexpected options path: %q", gotOptionsPath)
	}

	if gotListPath != "/api/manga/101/chapters/list" {
		t.Errorf("unexpected list path: %q", gotListPath)
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 scan paths, got %d", len(items))
	}

	expected := []struct {
		value string
		count int
		label string
	}{
		// Chapter 3 has no page and does not count; a row
		// listing both K Manga and Side Group counts for
		// both; a row with no group and no scanlator counts
		// for nothing.
		{"18210", 2, "K Manga (2 chapters, English/French)"},
		{"18192", 1, "LeerCapitulo (1 chapter, Spanish)"},
		{"scanlator:Stick", 1, "Stick (1 chapter, English)"},
		{"99999", 1, "Side Group (1 chapter, French)"},
	}

	for i, want := range expected {
		got := items[i]

		if got.Value != want.value {
			t.Errorf(
				"scan path %d: value = %q, want %q",
				i,
				got.Value,
				want.value,
			)
		}

		if got.Count != want.count {
			t.Errorf(
				"scan path %d: count = %d, want %d",
				i,
				got.Count,
				want.count,
			)
		}

		if got.Label != want.label {
			t.Errorf(
				"scan path %d: label = %q, want %q",
				i,
				got.Label,
				want.label,
			)
		}
	}
}

func TestListScanPathsSkipsUnexpectedOptions(t *testing.T) {
	server := fixtureServer(`{
		"languages": ["en"],
		"groups": [
			{
				"id": "18210",
				"name": "K Manga"
			},
			{
				"id": "",
				"name": "No ID"
			},
			{
				"id": "weird:thing",
				"name": "Odd"
			},
			{
				"id": "555"
			}
		]
	}`, "")
	defer server.Close()

	items, err := ListScanPaths(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 scan paths, got %d", len(items))
	}

	if items[0].Value != "18210" || items[0].Label != "K Manga" {
		t.Errorf(
			"unexpected first scan path: %+v",
			items[0],
		)
	}

	// A group without a name falls back to its id as label.
	if items[1].Value != "555" || items[1].Label != "555" {
		t.Errorf(
			"unexpected second scan path: %+v",
			items[1],
		)
	}
}

func TestListScanPathsEmptyGroups(t *testing.T) {
	// The chapter list call receives the same options JSON, fails
	// the row decode and degrades: counts stay unknown.
	server := fixtureServer(`{"languages": ["en"], "groups": []}`, "")
	defer server.Close()

	items, err := ListScanPaths(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	// A single item is automatically selectable by the application.
	if len(items) != 1 {
		t.Fatalf(
			"expected the default scan path, got %d item(s)",
			len(items),
		)
	}

	if items[0].Label != "MangaDot" {
		t.Errorf("unexpected label: %q", items[0].Label)
	}

	if items[0].Value != "" {
		t.Errorf("unexpected value: %q", items[0].Value)
	}

	if items[0].Count != -1 {
		t.Errorf("count = %d, want -1 (unknown)", items[0].Count)
	}
}

func TestListScanPathsImplicitPathCountsChapters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if strings.HasSuffix(r.URL.Path, "/chapters/list") {
				_, _ = fmt.Fprint(w, `[
					{
						"id": 1,
						"chapter_number": 1,
						"page_count": 20,
						"language": "en"
					},
					{
						"id": 2,
						"chapter_number": 2,
						"page_count": 20,
						"language": "fr"
					},
					{
						"id": 3,
						"chapter_number": 2,
						"page_count": 20,
						"language": "fr"
					},
					{
						"id": 4,
						"chapter_number": "not a number",
						"page_count": 20,
						"language": "en"
					}
				]`)

				return
			}

			_, _ = fmt.Fprint(w, `{"languages": ["en", "fr"], "groups": []}`)
		},
	))
	defer server.Close()

	items, err := ListScanPaths(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected the implicit scan path, got %d item(s)", len(items))
	}

	// Chapters 1 and 2 are distinct; the duplicate chapter 2 row
	// and the unparsable row 4 do not count.
	if items[0].Label != "MangaDot (2 chapters, English/French)" {
		t.Errorf("unexpected label: %q", items[0].Label)
	}

	if items[0].Count != 2 {
		t.Errorf("count = %d, want 2", items[0].Count)
	}
}

func TestListScanPathsCountFetchFailure(t *testing.T) {
	var gotOptions bool

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if strings.HasSuffix(r.URL.Path, "/filter-options") {
				gotOptions = true

				_, _ = fmt.Fprint(w, `{
					"languages": ["en"],
					"groups": [
						{"id": "18210", "name": "K Manga"},
						{"id": "18192", "name": "LeerCapitulo"}
					]
				}`)

				return
			}

			http.Error(
				w,
				"internal error",
				http.StatusInternalServerError,
			)
		},
	))
	defer server.Close()

	items, err := ListScanPaths(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if !gotOptions {
		t.Fatal("the filter options fetch never happened")
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 scan paths, got %d", len(items))
	}

	// The count fetch failed: the scan paths keep their plain
	// labels and API order.
	if items[0].Label != "K Manga" || items[0].Count != -1 {
		t.Errorf("unexpected first scan path: %+v", items[0])
	}

	if items[1].Label != "LeerCapitulo" || items[1].Count != -1 {
		t.Errorf("unexpected second scan path: %+v", items[1])
	}
}

func TestListScanPathsHTTPError(t *testing.T) {
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

	_, err := ListScanPaths(server.URL, "101", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 503")
	}
}

func TestListScanPathsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")

			_, _ = fmt.Fprint(w, "<html>challenge page</html>")
		},
	))
	defer server.Close()

	_, err := ListScanPaths(server.URL, "101", nil)
	if err == nil {
		t.Fatal("expected a decode error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "invalid character '<'") {
		t.Errorf("unexpected decode error: %v", err)
	}
}

func TestGetMangaTitle(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
				"manga": {
					"id": 101,
					"title": "Blue Lock"
				},
				"total_chapters": 383,
				"latest_chapter_number": 364.2
			}`)
		},
	))
	defer server.Close()

	title, err := GetMangaTitle(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("GetMangaTitle() error: %v", err)
	}

	if title != "Blue Lock" {
		t.Errorf("unexpected title: %q", title)
	}

	if gotPath != "/api/manga/101" {
		t.Errorf("unexpected path: %q", gotPath)
	}
}

func TestGetMangaTitleMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"manga": {"id": 101}}`)
		},
	))
	defer server.Close()

	_, err := GetMangaTitle(server.URL, "101", nil)
	if err == nil {
		t.Fatal("expected an error for a missing title")
	}

	if !strings.Contains(err.Error(), "has no title") {
		t.Errorf("unexpected error: %v", err)
	}
}
