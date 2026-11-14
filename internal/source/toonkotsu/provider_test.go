package toonkotsu

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
			expected: "https://toonkotsu.com",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://toonkotsu.com",
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

	if provider.Name() != "toonkotsu" {
		t.Errorf(
			"expected name toonkotsu, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{"items":[`+
				`{"code":"bm0ssx","slug":"jujutsu-kaisen","title":"Jujutsu Kaisen","cover":"/cdn/c/jjk.webp"},`+
				`{"code":"z9k1","slug":"berserk","title":"Berserk","cover":""}]`+
				`,"total":2,"hasNextPage":false}`)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	results, err := provider.Search("jujutsu", nil)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Jujutsu Kaisen" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}

	if results[0].URL != server.URL+"/title/bm0ssx-jujutsu-kaisen" {
		t.Errorf("unexpected url: %q", results[0].URL)
	}

	if results[0].CoverURL != server.URL+"/cdn/c/jjk.webp" {
		t.Errorf("unexpected cover: %q", results[0].CoverURL)
	}

	if results[1].CoverURL != "" {
		t.Errorf("unexpected cover: %q", results[1].CoverURL)
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

	_, err := provider.Search("jujutsu", nil)
	if err == nil {
		t.Fatal("expected an error from the scraper")
	}
}

func TestGetPageImageURLs(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/api/manga/chapter/thin": seriesRoute(
			`{"chapter":{"pages":[{"src":"/cdn/e/t.bin","k":"tk","n":"tn"}]}}`,
		),
		"/api/manga/chapter/full": seriesRoute(
			`{"chapter":{"pages":[` +
				`{"src":"/cdn/e/f1.bin","k":"fk1","n":"fn1"},` +
				`{"src":"https://other.test/cdn/e/f2.bin","k":"fk2","n":"fn2"}]}}`,
		),
	})

	imageURLs, err := provider.GetPageImageURLs(
		provider.domain+"/manga/chapter/thin,full",
		nil,
	)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	if len(imageURLs) != 2 {
		t.Fatalf("expected the fullest copy's 2 pages, got %d", len(imageURLs))
	}

	if imageURLs[0] != provider.domain+"/cdn/e/f1.bin#k=fk1&n=fn1" {
		t.Errorf("unexpected page url: %q", imageURLs[0])
	}

	// Absolute blob URLs pass through unchanged.
	if imageURLs[1] != "https://other.test/cdn/e/f2.bin#k=fk2&n=fn2" {
		t.Errorf("unexpected page url: %q", imageURLs[1])
	}
}

func TestGetPageImageURLsInvalidEntryURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, err := provider.GetPageImageURLs(
		provider.domain+"/not-a-chapter/471959",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected entry URL")
	}
}

func TestDecodeImageDataPassesThroughPlainPages(t *testing.T) {
	provider := New("")

	decoded, ok := provider.DecodeImageData(
		provider.domain+"/cdn/c/cover.webp",
		[]byte("anything"),
	)

	if ok || decoded != nil {
		t.Errorf(
			"expected the page to pass through, got %v / %v",
			decoded,
			ok,
		)
	}
}

// Compile-time: the optional decode capability is wired.
var _ sourcetypes.ImageDecoder = New("")

func TestImageRequestHeaders(t *testing.T) {
	provider := New("")

	headers := provider.ImageRequestHeaders()

	// The downloader's default bare "Mozilla/5.0" is rejected.
	if headers["User-Agent"] == "Mozilla/5.0" {
		t.Error("expected the bare default UA to be overridden")
	}

	if headers["User-Agent"] == "" {
		t.Error("expected a full browser user agent")
	}
}

// Compile-time: the image request headers capability is wired.
var _ sourcetypes.ImageRequestConfigurer = New("")

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

func seriesRoute(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, body)
	}
}

func TestListScanPaths(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/cdn/v1/series/bm0ssx": seriesRoute(`{"series":{"title":"Jujutsu Kaisen"},"chapters":[` +
			`{"id":1,"n":1,"label":"Chapter 1","alts":[]},` +
			`{"id":2,"n":2,"label":"Chapter 2","alts":[]}]` +
			`,"editions":[` +
			`{"code":"colored1","slug":"jjk-colored","title":"Jujutsu Kaisen Official Colored","chapters":4}]}`),
	})

	items, err := provider.ListScanPaths(
		provider.domain+"/title/bm0ssx-jujutsu-kaisen",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected main + edition, got %d paths", len(items))
	}

	// The main path rides the work's own series code: an empty
	// value would collide with the app's cancellation sentinel.
	if items[0].Label != "Toonkotsu (2 chapters)" || items[0].Value != "bm0ssx" {
		t.Errorf(
			"unexpected main path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Value != "colored1" ||
		items[1].Label != "Jujutsu Kaisen Official Colored (4 chapters)" {
		t.Errorf(
			"unexpected edition path: %q / %q",
			items[1].Label,
			items[1].Value,
		)
	}
}

func TestListScanPathsSingleListing(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/cdn/v1/series/solo": seriesRoute(
			`{"series":{"title":"Solo"},"chapters":[{"id":1,"n":1,"label":"Chapter 1"}]}`,
		),
	})

	// One selectable path auto-selects downstream with the main
	// series code as its value.
	items, err := provider.ListScanPaths(
		provider.domain+"/title/solo-solo",
		nil,
	)
	if err != nil {
		t.Fatalf("ListScanPaths() error: %v", err)
	}

	if len(items) != 1 || items[0].Value != "solo" {
		t.Fatalf("expected one auto-selectable path, got %+v", items)
	}
}

func TestListScanPathsInvalidWorkURL(t *testing.T) {
	provider := newTestProvider(t, nil)

	_, err := provider.ListScanPaths(
		provider.domain+"/not-a-title/bm0ssx",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestListEntries(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/cdn/v1/series/bm0ssx": seriesRoute(`{"series":{"title":"Jujutsu Kaisen"},"chapters":[` +
			`{"id":4581694,"n":2,"label":"Chapter 2","alts":[{"id":4581695}]},` +
			`{"id":4581691,"n":1,"label":"Chapter 1","alts":[{"id":4581692}]},` +
			`{"id":4581693,"n":1,"label":"","alts":[]}]}`),
	})

	work, entries, err := provider.ListEntries(
		provider.domain+"/title/bm0ssx-jujutsu-kaisen",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if work.Title != "Jujutsu Kaisen" {
		t.Errorf("unexpected work title: %q", work.Title)
	}

	if work.Kind != sourcetypes.ItemChapter {
		t.Errorf("unexpected work kind: %d", work.Kind)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Number.String() != "1" || entries[0].Label != "Chapter 1" {
		t.Errorf(
			"unexpected first entry: %q / %q",
			entries[0].Number.String(),
			entries[0].Label,
		)
	}

	// The entry URL stays opaque: it is what the site's URL shape
	// carries and what the pages step resolves back into ids.
	expectedURL := provider.domain + "/manga/chapter/4581691,4581692,4581693"

	if entries[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, entries[0].URL)
	}

	if entries[1].Number.String() != "2" {
		t.Errorf("unexpected second entry: %q", entries[1].Number.String())
	}
}

func TestListEntriesEditionListing(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/cdn/v1/series/bm0ssx": seriesRoute(`{"series":{"title":"Main"},"chapters":[]}`),
		"/cdn/v1/series/colored1": seriesRoute(`{"series":{"title":"Jujutsu Kaisen Official Colored"},"chapters":[` +
			`{"id":"c1","n":1,"label":"Chapter 1 (Official Colored)"}]}`),
	})

	work, entries, err := provider.ListEntries(
		provider.domain+"/title/bm0ssx-jujutsu-kaisen",
		"colored1",
		nil,
	)
	if err != nil {
		t.Fatalf("ListEntries() error: %v", err)
	}

	if work.Title != "Jujutsu Kaisen Official Colored" {
		t.Errorf("expected the edition title, got %q", work.Title)
	}

	if len(entries) != 1 {
		t.Errorf("expected the edition chapter, got %d", len(entries))
	}
}

func TestListEntriesEmptyListing(t *testing.T) {
	provider := newTestProvider(t, map[string]http.HandlerFunc{
		"/cdn/v1/series/solo": seriesRoute(`{"series":{"title":"Solo"},"chapters":[]}`),
	})

	_, entries, err := provider.ListEntries(
		provider.domain+"/title/solo-solo",
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
		provider.domain+"/not-a-title/bm0ssx",
		"",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}
