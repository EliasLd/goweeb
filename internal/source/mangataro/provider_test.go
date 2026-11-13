package mangataro

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func TestNew(t *testing.T) {
	cases := []struct {
		name         string
		customDomain string
		expected     string
	}{
		{
			name:     "default domain",
			expected: "https://mangataro.org",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://mangataro.org",
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

	if provider.Name() != "mangataro" {
		t.Errorf(
			"expected name mangataro, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `[
				{"id":"32148","title":"Berserk","url":"https://mangataro.org/manga/berserk","cover":"https://mangataro.org/content/media/berserk.jpg"},
				{"id":"635694","title":"The Berserker&#039;s Second Playthrough","url":"https://mangataro.org/manga/the-berserker-039-s-second-playthrough","cover":""}
			]`)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	results, err := provider.Search("berserk", nil)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Berserk" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}

	if results[0].URL != "https://mangataro.org/manga/berserk" {
		t.Errorf("unexpected url: %q", results[0].URL)
	}

	if results[0].CoverURL != "https://mangataro.org/content/media/berserk.jpg" {
		t.Errorf("unexpected cover: %q", results[0].CoverURL)
	}

	// The catalog payloads embed HTML entities in their titles.
	if results[1].Title != "The Berserker's Second Playthrough" {
		t.Errorf("expected the entity-decoded title, got %q", results[1].Title)
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

	_, err := provider.Search("berserk", nil)
	if err == nil {
		t.Fatal("expected an error from the scraper")
	}
}

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

// titlePageHTML renders the SSR markers the provider resolves the
// numeric manga id from.
const titlePageHTML = "<html><body class=\"bg-neutral-900\"" +
	" data-manga-id=\"32148\">" +
	"<h1>Berserk</h1>" +
	"</body></html>"

func TestListScanPaths(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/manga/berserk": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "text/html")

			_, _ = fmt.Fprint(
				w,
				"<html><body class=\"bg-neutral-900\""+
					" data-manga-id=\"32148\">"+
					"<h1>Berserk</h1>"+
					"</body></html>",
			)
		},
		"/auth/manga-chapters": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w,
				`{"success":true,"chapters":[`+
					`{"id":"1","chapter":"1","title":"Chapter 1","chapter_type":"media","language":"en","url":"https://mangataro.org/read/berserk/ch1-1"},`+
					`{"id":"2","chapter":"2","title":"Chapter 2","chapter_type":"media","language":"en","url":"https://mangataro.org/read/berserk/ch2-2"},`+
					`{"id":"3","chapter":"1","title":"Capítulo 1","chapter_type":"media","language":"pt-br","url":"https://mangataro.org/read/berserk/ch1-3"}`+
					`],"total":3,"offset":0,"limit":500,"has_more":false}`)
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/manga/berserk",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 scan paths, got %d", len(items))
	}

	// Counts come from the listing (2 distinct numbers in en),
	// and the labels sort alphabetically.
	if items[0].Label != "English (2 chapters)" || items[0].Value != "en" {
		t.Errorf(
			"unexpected first scan path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Label != "Portuguese (Brazil) (1 chapter)" || items[1].Value != "pt-br" {
		t.Errorf(
			"unexpected second scan path: %q / %q",
			items[1].Label,
			items[1].Value,
		)
	}
}

func TestListScanPathsFallbackWithoutChapters(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/manga/berserk": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "text/html")

			_, _ = fmt.Fprint(w, titlePageHTML)
		},
		"/auth/manga-chapters": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w,
				`{"success":true,"chapters":[],"total":0,"offset":0,"limit":500,"has_more":false}`)
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/manga/berserk",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 1 || items[0].Value != "" {
		t.Fatalf(
			"expected one auto-selectable fallback path, got %+v",
			items,
		)
	}
}

func TestListScanPathsInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, err := provider.ListScanPaths(
		provider.domain+"/not-a-manga/berserk",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestListEntries(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/manga/berserk": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, titlePageHTML)
		},
		"/auth/manga-chapters": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w,
				`{"success":true,"chapters":[`+
					`{"id":"2","chapter":"2","title":"Chapter 2","chapter_type":"media","language":"en","url":"https://mangataro.org/read/berserk/ch2-2"},`+
					`{"id":"1","chapter":"1","title":"The Golden Age (1)","chapter_type":"media","language":"en","url":"https://mangataro.org/read/berserk/ch1-1"},`+
					`{"id":"3","chapter":"1","title":"Capítulo 1","chapter_type":"media","language":"pt-br","url":"https://mangataro.org/read/berserk/ch1-3"}`+
					`],"total":3,"offset":0,"limit":500,"has_more":false}`)
		},
	})

	work, entries, err := provider.ListEntries(
		provider.domain+"/manga/berserk",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if work.Title != "Berserk" {
		t.Errorf("unexpected work title: %q", work.Title)
	}

	if work.Kind != sourcetypes.ItemChapter {
		t.Errorf("unexpected work kind: %d", work.Kind)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Number.String() != "1" || entries[0].Label != "The Golden Age (1)" {
		t.Errorf(
			"unexpected first entry: %q / %q",
			entries[0].Number.String(),
			entries[0].Label,
		)
	}

	// The entry URL is the row's own, passed through unchanged: it
	// is what the site lists and what the pages step resolves back
	// to a chapter id.
	expectedURL := "https://mangataro.org/read/berserk/ch1-1"

	if entries[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, entries[0].URL)
	}

	if entries[1].Number.String() != "2" {
		t.Errorf(
			"unexpected second entry: %q",
			entries[1].Number.String(),
		)
	}
}

func TestListEntriesEmptyListing(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/manga/berserk": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, titlePageHTML)
		},
		"/auth/manga-chapters": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w,
				`{"success":true,"chapters":[],"total":0,"offset":0,"limit":500,"has_more":false}`)
		},
	})

	_, entries, err := provider.ListEntries(
		provider.domain+"/manga/berserk",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected no entries, got %d", len(entries))
	}
}

func TestListEntriesInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, _, err := provider.ListEntries(
		provider.domain+"/not-a-manga/berserk",
		"en",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestGetPageImageURLsNotImplementedYet(t *testing.T) {
	provider := New("")

	_, err := provider.GetPageImageURLs(
		"https://mangataro.org/read/berserk/ch398-471959",
		nil,
	)
	if err == nil {
		t.Fatal("expected the stub to reject the call")
	}
}
