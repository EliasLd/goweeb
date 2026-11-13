package mangataroscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListChaptersFiltersByLanguage(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("1", "2", "Chapter 2", "en"),
				chapterRowJSON("2", "1", "Chapter 1", "en"),
				chapterRowJSON("3", "1", "Capítulo 1", "pt-br"),
			},
			3,
			false,
		),
	)

	chapters, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 english chapters, got %d", len(chapters))
	}

	if chapters[0].Number.String() != "1" || chapters[1].Number.String() != "2" {
		t.Errorf(
			"expected ascending order, got %q then %q",
			chapters[0].Number.String(),
			chapters[1].Number.String(),
		)
	}
}

func TestListChaptersSortsDecimalsExactly(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("1", "356.2", "J&ouml;tunn (Part Two)", "en"),
				chapterRowJSON("2", "2", "Chapter 2", "en"),
				chapterRowJSON("3", "0.1", "The Golden Age", "en"),
				chapterRowJSON("4", "356.1", "J&ouml;tunn (Part One)", "en"),
			},
			4,
			false,
		),
	)

	chapters, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	want := []string{"0.1", "2", "356.1", "356.2"}

	if len(chapters) != len(want) {
		t.Fatalf("expected %d chapters, got %d", len(want), len(chapters))
	}

	for i, number := range want {
		if chapters[i].Number.String() != number {
			t.Errorf(
				"chapters[%d].Number = %q, want %q",
				i,
				chapters[i].Number.String(),
				number,
			)
		}
	}

	// Labels prefer the site's own titles, entity-decoded.
	if chapters[3].Label != "Jötunn (Part Two)" {
		t.Errorf("label = %q, want the decoded site title", chapters[3].Label)
	}
}

func TestListChaptersLabelsFallBackToTheNumber(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("1", "7", "", "en"),
				chapterRowJSON("2", "8", "  ", "en"),
			},
			2,
			false,
		),
	)

	chapters, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	if chapters[0].Label != "Chapter 7" {
		t.Errorf("label = %q, want the fallback rendering", chapters[0].Label)
	}

	if chapters[1].Label != "Chapter 8" {
		t.Errorf("label = %q, want the fallback rendering", chapters[1].Label)
	}
}

func TestListChaptersDeduplicatesNumbers(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("first", "100", "Chapter 100", "en"),
				chapterRowJSON("second", "100", "Chapter 100 mirror", "en"),
				chapterRowJSON("third", "101", "Chapter 101", "en"),
			},
			3,
			false,
		),
	)

	chapters, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected the duplicate to lose, got %d chapters", len(chapters))
	}

	if chapters[1].URL == "" ||
		!strings.HasSuffix(chapters[1].URL, "ch101-third") {
		t.Errorf("unexpected survivor url: %q", chapters[1].URL)
	}
}

func TestListChaptersSkipsUnusableRows(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("1", "1", "Chapter 1", "en"),
				`{"id":"","chapter":"2","title":"No id","language":"en","url":"https://x/ch2-no"}`,
				`{"id":"3","chapter":"notanumber","title":"NaN","language":"en","url":"https://x/chnan-3"}`,
				`{"id":"4","chapter":"4","title":"No url","language":"en","url":"  "}`,
			},
			4,
			false,
		),
	)

	chapters, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 1 || chapters[0].Number.String() != "1" {
		t.Errorf("expected only the usable row to survive, got %+v", chapters)
	}
}

func TestListChaptersEmptyListing(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage([]string{}, 0, false),
	)

	chapters, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 0 {
		t.Errorf("expected no chapters, got %d", len(chapters))
	}
}

func TestListChaptersIncompleteWalkFails(t *testing.T) {
	page := listingPage(
		[]string{chapterRowJSON("1", "1", "Chapter 1", "en")},
		99999,
		true,
	)

	pages := make([]chapterListingResponse, 0, maxListingPages)

	for i := 0; i < maxListingPages; i++ {
		pages = append(pages, page)
	}

	fixture := newListingServer(t, pages...)

	_, err := ListChapters(
		fixture.server.URL,
		"32148",
		"en",
		testDiscardLogger(),
	)
	if err == nil {
		t.Fatal("expected the capped walk to be refused")
	}

	if !strings.Contains(err.Error(), "listing is incomplete") {
		t.Errorf("expected the truncation error, got %v", err)
	}
}

func TestListChaptersFailureMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"success":false,"message":"Invalid token"}`)
		},
	))
	defer server.Close()

	_, err := ListChapters(server.URL, "32148", "en", testDiscardLogger())
	if err == nil {
		t.Fatal("expected the failure message to surface")
	}

	if !strings.Contains(err.Error(), "Invalid token") {
		t.Errorf("expected the route's message, got %v", err)
	}
}

func TestListChaptersHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusGone)
		},
	))
	defer server.Close()

	_, err := ListChapters(server.URL, "32148", "en", testDiscardLogger())
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "status 410") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}

func TestListChaptersDecodesMalformedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"success":true,"chapters":`)
		},
	))
	defer server.Close()

	_, err := ListChapters(server.URL, "32148", "en", testDiscardLogger())
	if err == nil {
		t.Fatal("expected a decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode MangaTaro chapter listing") {
		t.Errorf("expected a decode error, got %v", err)
	}
}
