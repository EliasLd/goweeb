package mangadotnet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
		"/api/manga/101/chapters/list": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			// The group uploads 2 chapters in English; the
			// scanlator-only path uploads 2 chapters, one in
			// French and one in English.
			jsonResponse(w, `[
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
					"language": "en",
					"groups": [{"id": 18210, "name": "K Manga"}]
				},
				{
					"id": 3,
					"chapter_number": 1,
					"page_count": 10,
					"language": "fr",
					"scanlator_name": "Stick"
				},
				{
					"id": 4,
					"chapter_number": 2,
					"page_count": 10,
					"language": "en",
					"scanlator_name": "Stick"
				}
			]`)
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

	// At this step the lists keep the API order.
	if items[0].Label != "K Manga (2 chapters, English)" ||
		items[0].Value != "18210" {
		t.Errorf(
			"unexpected first scan path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Label != "Stick (2 chapters, English/French)" ||
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

func TestListEntries(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/101": func(w http.ResponseWriter, _ *http.Request) {
			jsonResponse(w, `{
				"manga": {
					"id": 101,
					"title": "Blue Lock"
				},
				"total_chapters": 3
			}`)
		},
		"/api/manga/101/chapters/list": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			jsonResponse(w, `[
				{
					"id": 890000,
					"chapter_number": 2.5,
					"chapter_title": "Chapter 2.5",
					"page_count": 20,
					"source": "user",
					"date_added": "2026-10-13 16:17:51.726423+00"
				},
				{
					"id": 888840,
					"chapter_number": 1,
					"chapter_title": null,
					"page_count": 21,
					"source": "user",
					"date_added": "2026-06-10 12:00:00.000000+00"
				}
			]`)
		},
	})

	work, entries, err := provider.ListEntries(
		provider.domain+"/manga/101",
		"18210",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if work.Title != "Blue Lock" {
		t.Errorf("unexpected work title: %q", work.Title)
	}

	if work.Kind != sourcetypes.ItemChapter {
		t.Errorf("unexpected work kind: %d", work.Kind)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Number.String() != "1" ||
		entries[0].Label != "Chapter 1" {
		t.Errorf(
			"unexpected first entry: %q / %q",
			entries[0].Number.String(),
			entries[0].Label,
		)
	}

	expectedURL := provider.domain + "/chapter/888840?source=user"

	if entries[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, entries[0].URL)
	}

	if entries[1].Number.String() != "2.5" {
		t.Errorf(
			"unexpected second entry: %q",
			entries[1].Number.String(),
		)
	}
}

func TestListEntriesInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, _, err := provider.ListEntries(
		provider.domain,
		"18210",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestGetPageImageURLs(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/uploads/888840/images": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			jsonResponse(w, `{
				"chapter": {"id": 888840, "page_count": 2},
				"manga": {"id": 101, "title": "Blue Lock"},
				"images": [
					{
						"url": "/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/001.webp",
						"filename": "001.webp"
					},
					{
						"url": "/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/002.webp",
						"filename": "002.webp"
					}
				]
			}`)
		},
	})

	entryURL := provider.domain + "/chapter/888840?source=user"

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

	// Relative image paths are joined against the provider domain.
	expected := provider.domain +
		"/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/001.webp"

	if imageURLs[0] != expected {
		t.Errorf("expected URL %q, got %q", expected, imageURLs[0])
	}
}
