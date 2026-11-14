package toonkotsuscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newSeriesServer serves one series payload under its code path.
func newSeriesServer(
	t *testing.T,
	code string,
	body string,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc(seriesPath+code, func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, body)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func TestGetScanPathsMainFirstThenEditions(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", `{"series":{"title":"Jujutsu Kaisen"},"chapters":[`+
		`{"id":"c1","n":2,"label":"Chapter 2","alts":[]},`+
		`{"id":"c2","n":1,"label":"Chapter 1","alts":[]},`+
		`{"id":"c3","n":1,"label":"Chapter 1 mirror","alts":[]},`+
		`{"id":"c4","n":"271.5","label":"Late","alts":[]},`+
		`{"id":"c5","n":null,"label":"Volume 3","alts":[]}`+
		`],"editions":[`+
		`{"code":"colored1","kind":"color","slug":"jjk-colored","title":"Jujutsu Kaisen Official Colored","chapters":4},`+
		`{"code":"","kind":"color","title":"Broken","chapters":1}]}`)

	items, err := GetScanPaths(
		server.URL,
		server.URL+"/title/bm0ssx-jujutsu-kaisen",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetScanPaths() error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 scan paths, got %d", len(items))
	}

	// Main first, counted as distinct chapter numbers: the
	// mirrored row and the unnumbered volume row never count.
	if items[0].Label != "Toonkotsu (3 chapters)" || items[0].Value != "bm0ssx" {
		t.Errorf(
			"unexpected main scan path: %q / %q",
			items[0].Label,
			items[0].Value,
		)
	}

	if items[1].Label != "Jujutsu Kaisen Official Colored (4 chapters)" ||
		items[1].Value != "colored1" {
		t.Errorf(
			"unexpected edition scan path: %q / %q",
			items[1].Label,
			items[1].Value,
		)
	}
}

func TestGetScanPathsSingleListingAutoSelects(t *testing.T) {
	server := newSeriesServer(t, "solo", `{"series":{"title":"Solo"},"chapters":[`+
		`{"id":"c1","n":1,"label":"Chapter 1","alts":[]}]}`)

	items, err := GetScanPaths(
		server.URL,
		server.URL+"/title/solo-solo",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetScanPaths() error: %v", err)
	}

	// One selectable path auto-selects downstream with the main
	// series code as its value.
	if len(items) != 1 || items[0].Value != "solo" {
		t.Fatalf("expected one auto-selectable path, got %+v", items)
	}
}

func TestGetScanPathsInvalidWorkURL(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", "{}")

	_, err := GetScanPaths(
		server.URL,
		server.URL+"/not-a-title/bm0ssx",
		testDiscardLogger(),
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestGetScanPathsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "blocked", http.StatusForbidden)
		},
	))
	defer server.Close()

	_, err := GetScanPaths(
		server.URL,
		server.URL+"/title/bm0ssx-jujutsu-kaisen",
		testDiscardLogger(),
	)
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "status 403") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}
