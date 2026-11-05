package mangaball

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
			expected: "https://mangaball.com",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://mangaball.com",
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

	if provider.Name() != "mangaball" {
		t.Errorf(
			"expected name mangaball, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{"code":200,"status":"success","data":[`+
				`{"id":"6851544a702284f8341782f2","name":"Berserk","slug":"berserk"},`+
				`{"id":"685270ea6908629fc2c82502","name":"Otonari Complex","slug":"otomari-complex"}]}`)
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

	expectedURL := server.URL +
		"/title-detail/6851544a702284f8341782f2-berserk"

	if results[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, results[0].URL)
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

func listingResponse(
	rowsJSON string,
) string {
	return `{"status":"success","code":200,"data":` +
		rowsJSON +
		`,"available_languages":["en","fr"],` +
		`"pagination":{"page":1,"limit":5000,"total":1,"total_pages":1}}`
}

func TestListScanPaths(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/v1/title/chapter-listing": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, listingResponse(
				`[
					{"id":"c1","number":1,"name":"Chapter 1","lang":"en","site":"mangadex"},
					{"id":"c2","number":1,"name":"Chapter 1","lang":"en","site":"comick"},
					{"id":"c3","number":2,"name":"Chapter 2","lang":"en","site":"comick"},
					{"id":"c4","number":1,"name":"Chapitre 1","lang":"fr","site":"mangadex"}
				]`,
			))
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/title-detail/6851544a702284f8341782f2-berserk",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 scan paths, got %d", len(items))
	}

	// The duplicate English row dedups the number count to 2, and
	// the paths sort by label.
	if items[0].Label != "English (2 chapters)" || items[0].Value != "en" {
		t.Errorf(
			"unexpected first scan path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Label != "French (1 chapter)" || items[1].Value != "fr" {
		t.Errorf(
			"unexpected second scan path: %q / %q",
			items[1].Label,
			items[1].Value,
		)
	}
}

func TestListScanPathsFallbackWithoutChapters(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/v1/title/chapter-listing": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, listingResponse(`[]`))
		},
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/title-detail/6851544a702284f8341782f2-empty",
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
		provider.domain+"/not-a-title/6851544a702284f8341782f2",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestListEntries(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/v1/title/detail": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w,
				`{"status":"success","code":200,"data":{"name":"Berserk"}}`)
		},
		"/api/v1/title/chapter-listing": func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, listingResponse(
				`[
					{"id":"id-2","number":2,"name":"Chapter 2","lang":"en","site":"mangadex"},
					{"id":"id-1-b","number":1,"name":"Chapter 1","lang":"en","site":"comick"},
					{"id":"id-1-a","number":1,"name":"Chapter 1","lang":"en","site":"mangadex"}
				]`,
			))
		},
	})

	work, entries, err := provider.ListEntries(
		provider.domain+"/title-detail/6851544a702284f8341782f2-berserk",
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

	if entries[0].Number.String() != "1" ||
		entries[0].Label != "Chapter 1" {
		t.Errorf(
			"unexpected first entry: %q / %q",
			entries[0].Number.String(),
			entries[0].Label,
		)
	}

	// Duplicate sources join the entry URL; the page step picks
	// the fullest copy later.
	expectedURL := provider.domain + "/chapter-detail/id-1-b,id-1-a"

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

func TestListEntriesInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, _, err := provider.ListEntries(
		provider.domain+"/not-a-title/6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}
