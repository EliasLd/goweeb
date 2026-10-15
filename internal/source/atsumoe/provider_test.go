package atsumoe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
