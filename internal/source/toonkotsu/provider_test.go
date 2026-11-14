package toonkotsu

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestNotImplementedMethodsStayHonest(t *testing.T) {
	provider := New("")

	if _, _, err := provider.ListEntries("", "", nil); err == nil ||
		!strings.Contains(err.Error(), "not implemented") {
		t.Errorf("ListEntries() error: %v", err)
	}

	if _, err := provider.GetPageImageURLs("", nil); err == nil ||
		!strings.Contains(err.Error(), "not implemented") {
		t.Errorf("GetPageImageURLs() error: %v", err)
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
