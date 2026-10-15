package atsumoe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
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

func TestListScanPaths(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/page": func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `{
				"mangaPage": {
					"id": "nh6Ii",
					"scanlators": [
						{
							"id": "S1",
							"name": "Delta"
						},
						{
							"id": "S2",
							"name": "Alpha"
						}
					]
				}
			}`)
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/manga/nh6Ii",
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

	if items[0].Label != "Delta" || items[0].Value != "S1" {
		t.Errorf(
			"unexpected first scan path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Label != "Alpha" || items[1].Value != "S2" {
		t.Errorf(
			"unexpected second scan path: %q / %q",
			items[1].Label,
			items[1].Value,
		)
	}
}

func TestListScanPathsNoScanlators(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/page": func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `{
				"mangaPage": {
					"id": "jIg3F",
					"scanlators": []
				}
			}`)
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/manga/jIg3F",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	// A single item is automatically selectable by the application.
	if len(items) != 1 {
		t.Fatalf(
			"expected 1 scan path, got %d",
			len(items),
		)
	}

	if items[0].Label != "Atsu.moe" {
		t.Errorf("unexpected label: %q", items[0].Label)
	}

	if items[0].Value != "" {
		t.Errorf("unexpected value: %q", items[0].Value)
	}
}

func TestListScanPathsInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, err := provider.ListScanPaths("https://atsu.moe/not-manga/x", nil)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestListEntries(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/info": func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `{
				"id": "nh6Ii",
				"title": "One-Punch Man",
				"type": "Manga",
				"isAdult": false,
				"chapters": []
			}`)
		},
		"/api/manga/allChapters": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			jsonResponse(w, `{
				"chapters": [
					{
						"id": "c2",
						"scanlationMangaId": "S1",
						"title": "# 2",
						"number": 2,
						"index": 2,
						"pageCount": 19
					},
					{
						"id": "c1other",
						"scanlationMangaId": "S2",
						"title": "# 1",
						"number": 1,
						"index": 1,
						"pageCount": 18
					},
					{
						"id": "c1",
						"scanlationMangaId": "S1",
						"title": "# 1",
						"number": 1,
						"index": 1,
						"pageCount": 20
					}
				]
			}`)
		},
	})

	work, entries, err := provider.ListEntries(
		provider.domain+"/manga/nh6Ii",
		"S1",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if work.Title != "One-Punch Man" {
		t.Errorf("unexpected work title: %q", work.Title)
	}

	if work.Kind != sourcetypes.ItemChapter {
		t.Errorf("unexpected work kind: %d", work.Kind)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 entries, got %d",
			len(entries),
		)
	}

	if entries[0].Number.String() != "1" ||
		entries[0].Label != "# 1" {
		t.Errorf(
			"unexpected first entry: %q / %q",
			entries[0].Number.String(),
			entries[0].Label,
		)
	}

	if entries[1].Number.String() != "2" ||
		entries[1].Label != "# 2" {
		t.Errorf(
			"unexpected second entry: %q / %q",
			entries[1].Number.String(),
			entries[1].Label,
		)
	}

	expectedURL := provider.domain +
		"/api/read/chapter?chapterId=c1&mangaId=nh6Ii"

	if entries[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, entries[0].URL)
	}
}

func TestListEntriesAllScanlations(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/info": func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `{
				"id": "nh6Ii",
				"title": "One-Punch Man",
				"chapters": []
			}`)
		},
		"/api/manga/allChapters": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			jsonResponse(w, `{
				"chapters": [
					{
						"id": "c1",
						"scanlationMangaId": "S1",
						"title": "# 1",
						"number": 1,
						"index": 1,
						"pageCount": 20
					},
					{
						"id": "c1other",
						"scanlationMangaId": "S2",
						"title": "# 1",
						"number": 1,
						"index": 1,
						"pageCount": 18
					}
				]
			}`)
		},
	})

	// An empty scan path keeps chapters from every scanlation group.
	_, entries, err := provider.ListEntries(
		provider.domain+"/manga/nh6Ii",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 entries, got %d",
			len(entries),
		)
	}
}

func TestListEntriesInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, _, err := provider.ListEntries(
		provider.domain,
		"S1",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestGetPageImageURLs(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/read/chapter": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			jsonResponse(w, `{
				"readChapter": {
					"id": "ZfvD517i",
					"number": 2,
					"pages": [
						{
							"id": "ZfvD517i-1",
							"image": "/static/pages/jIg3F/ZfvD517i/1.webp",
							"number": 1
						},
						{
							"id": "ZfvD517i-0",
							"image": "/static/pages/jIg3F/ZfvD517i/0.webp",
							"number": 0
						}
					]
				}
			}`)
		},
	})

	entryURL := provider.domain +
		"/api/read/chapter?mangaId=jIg3F&chapterId=ZfvD517i"

	imageURLs, err := provider.GetPageImageURLs(entryURL, nil)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	if len(imageURLs) != 2 {
		t.Fatalf(
			"expected 2 image URLs, got %d",
			len(imageURLs),
		)
	}

	// Relative image paths are joined against the CDN host derived
	// from the provider domain.
	base, err := url.Parse(provider.domain)
	if err != nil {
		t.Fatalf("failed to parse provider domain: %v", err)
	}

	expectedHost := "cdn." + base.Host

	expected := []string{
		"http://" + expectedHost +
			"/static/pages/jIg3F/ZfvD517i/0.webp",
		"http://" + expectedHost +
			"/static/pages/jIg3F/ZfvD517i/1.webp",
	}

	for i, want := range expected {
		if imageURLs[i] != want {
			t.Errorf(
				"image %d: expected %q, got %q",
				i,
				want,
				imageURLs[i],
			)
		}
	}
}
