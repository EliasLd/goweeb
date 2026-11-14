package toonkotsuscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// listingFixture is the canonical One-Piece-shaped listing: one
// number spread over several rows, alternates on each row, a
// fractional number and unnumbered volume rows. Ids arrive as JSON
// numbers, the way the site delivers them.
const listingFixture = `{"series":{"title":"Jujutsu Kaisen"},"chapters":[` +
	`{"id":4581694,"n":2,"label":"Chapter 2","alts":[{"id":4581695}]},` +
	`{"id":4581691,"n":1,"label":"Chapter 1","alts":[{"id":4581692}]},` +
	`{"id":4581693,"n":1,"label":"","alts":[]},` +
	`{"id":"4581696","n":"271.5","label":"Late","alts":[]},` +
	`{"id":4581697,"n":null,"label":"Volume 112","alts":[]}]}`

func TestListChaptersSortsAscendingAndMergesUploads(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", listingFixture)

	title, chapters, err := ListChapters(
		server.URL,
		"bm0ssx",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if title != "Jujutsu Kaisen" {
		t.Errorf("unexpected title: %q", title)
	}

	if len(chapters) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(chapters))
	}

	if chapters[0].Number.String() != "1" ||
		chapters[1].Number.String() != "2" ||
		chapters[2].Number.String() != "271.5" {
		t.Errorf(
			"expected reading order 1, 2, 271.5, got %s, %s, %s",
			chapters[0].Number.String(),
			chapters[1].Number.String(),
			chapters[2].Number.String(),
		)
	}

	// Every upload of chapter 1 (both rows and the row's
	// alternates) folds into one opaque entry URL, led by the
	// site's first row. Numeric and string ids render identically.
	if chapters[0].URL != server.URL+"/manga/chapter/4581691,4581692,4581693" {
		t.Errorf("unexpected chapter 1 URL: %q", chapters[0].URL)
	}

	if chapters[1].URL != server.URL+"/manga/chapter/4581694,4581695" {
		t.Errorf("unexpected chapter 2 URL: %q", chapters[1].URL)
	}

	if chapters[2].URL != server.URL+"/manga/chapter/4581696" {
		t.Errorf("unexpected chapter 271.5 URL: %q", chapters[2].URL)
	}
}

func TestListChaptersSkipsUnnumberedAndUnusableRows(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", `{"series":{"title":"X"},"chapters":[`+
		`{"id":4581697,"n":null,"label":"Volume 112","alts":[]},`+
		`{"id":4581698,"n":null,"label":"Bonus Manga","alts":[]},`+
		`{"id":4581699,"n":1e5,"label":"Exponent","alts":[]},`+
		`{"id":"","n":3,"label":"No ids","alts":[{"id":""}]},`+
		`{"id":4581700,"n":1,"label":"Chapter 1","alts":[]}]}`)

	_, chapters, err := ListChapters(
		server.URL,
		"bm0ssx",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 1 || chapters[0].Number.String() != "1" {
		t.Errorf(
			"expected only the usable row to survive, got %+v",
			chapters,
		)
	}
}

func TestListChaptersSelectsTheEditionListing(t *testing.T) {
	var requested []string

	mux := http.NewServeMux()

	mux.HandleFunc(seriesPath+"main", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		requested = append(requested, "main")

		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, `{"series":{"title":"Main"},"chapters":[{"id":"m1","n":1}]}`)
	})
	mux.HandleFunc(seriesPath+"color1", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		requested = append(requested, "color1")

		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, `{"series":{"title":"Jujutsu Kaisen Official Colored"},"chapters":[{"id":"c1","n":1}]}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	title, chapters, err := ListChapters(
		server.URL,
		"main",
		"color1",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(requested) != 1 || requested[0] != "color1" {
		t.Errorf("expected only the edition listing, got %v", requested)
	}

	if title != "Jujutsu Kaisen Official Colored" {
		t.Errorf("expected the edition title, got %q", title)
	}

	if len(chapters) != 1 {
		t.Errorf("expected the edition chapter, got %d", len(chapters))
	}
}

func TestListChaptersLabelsFallBackToTheNumber(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", `{"series":{"title":"X"},"chapters":[`+
		`{"id":"r7","n":7,"label":"","alts":[]},`+
		`{"id":"r8","n":8,"label":"  ","alts":[]}]}`)

	_, chapters, err := ListChapters(
		server.URL,
		"bm0ssx",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if chapters[0].Label != "Chapter 7" ||
		chapters[1].Label != "Chapter 8" {
		t.Errorf(
			"expected the fallback rendering, got %q / %q",
			chapters[0].Label,
			chapters[1].Label,
		)
	}
}

func TestListChaptersEmptyListing(t *testing.T) {
	server := newSeriesServer(t, "empty", `{"series":{"title":"X"},"chapters":[]}`)

	title, chapters, err := ListChapters(
		server.URL,
		"empty",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	// A degenerate payload still names the work for the output
	// layout.
	if title != "X" {
		t.Errorf("unexpected title: %q", title)
	}

	if len(chapters) != 0 {
		t.Errorf("expected no chapters, got %d", len(chapters))
	}
}

func TestListChaptersFallsBackToTheSiteName(t *testing.T) {
	server := newSeriesServer(t, "noname", `{"series":{},"chapters":[]}`)

	title, _, err := ListChapters(
		server.URL,
		"noname",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if title != "Toonkotsu" {
		t.Errorf("unexpected fallback title: %q", title)
	}
}

func TestListChaptersMalformedPayload(t *testing.T) {
	server := newSeriesServer(t, "bad", `{"series":{"title":"X"},"chapters":[`)

	_, _, err := ListChapters(server.URL, "bad", "", testDiscardLogger())
	if err == nil {
		t.Fatal("expected a decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode toonkotsu series bad") {
		t.Errorf("expected the decode error, got %v", err)
	}
}
