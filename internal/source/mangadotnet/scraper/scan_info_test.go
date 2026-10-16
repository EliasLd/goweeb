package mangadotnetscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListScanPaths(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			w.Header().Set("Content-Type", "application/json")
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
					}
				]
			}`)
		},
	))
	defer server.Close()

	items, err := ListScanPaths(server.URL, "101", nil)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	expectedPath := "/api/manga/101/chapters/filter-options"

	if gotPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, gotPath)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 scan paths, got %d", len(items))
	}

	// Filter option ids are forwarded verbatim, in API order.
	expected := []ScanPath{
		{Label: "K Manga", Value: "18210"},
		{Label: "LeerCapitulo", Value: "18192"},
		{Label: "Stick", Value: "scanlator:Stick"},
	}

	for i, want := range expected {
		if items[i] != want {
			t.Errorf(
				"scan path %d: expected %+v, got %+v",
				i,
				want,
				items[i],
			)
		}
	}
}

func TestListScanPathsSkipsUnexpectedOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
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
			}`)
		},
	))
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
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"languages": ["en"], "groups": []}`)
		},
	))
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
