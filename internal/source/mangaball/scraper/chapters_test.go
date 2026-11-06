package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newChaptersServer wires a chapter listing and a title detail
// endpoint under one server so the scraper talks to a single host.
func newChaptersServer(
	t *testing.T,
	total int,
	rowsJSON string,
	titleName string,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc(
		"/api/v1/title/chapter-listing",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, listingResponse(rowsJSON, total))
		},
	)

	mux.HandleFunc(
		"/api/v1/title/detail",
		func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, fmt.Sprintf(
				`{"status":"success","code":200,"data":{"name":%q}}`,
				titleName,
			))
		},
	)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func TestGetTitleName(t *testing.T) {
	server := newChaptersServer(t, 0, `[]`, "Berserk")

	name, err := GetTitleName(server.URL, "6851544a702284f8341782f2", nil)
	if err != nil {
		t.Fatalf("title: %v", err)
	}

	if name != "Berserk" {
		t.Errorf("name = %q, want Berserk", name)
	}
}

func TestGetTitleNameEmpty(t *testing.T) {
	server := newChaptersServer(t, 0, `[]`, "")

	_, err := GetTitleName(server.URL, "6851544a702284f8341782f2", nil)

	if err == nil || !strings.Contains(err.Error(), "has no name") {
		t.Fatalf("error = %v, want the unnamed-title failure", err)
	}
}

func TestListChaptersDedupsSourcesAndSorts(
	t *testing.T,
) {
	// Rows arrive number-descending (site order) with the same
	// chapter crawled from several sites.
	rows := `[
		{"id":"id-5","number":5,"name":"Chapter 5 - finale","lang":"en","site":"mangadex"},
		{"id":"id-35","number":3.5,"name":"Chapter 3.5","lang":"en","site":"mangadex"},
		{"id":"id-3-b","number":3,"name":"Chapter 3 (other site)","lang":"en","site":"comick"},
		{"id":"id-3-a","number":3,"name":"Chapter 3","lang":"en","site":"mangadex"},
		{"id":"id-2","number":2,"name":"","lang":"en","site":"mangadex"},
		{"id":"id-1","number":1,"name":"Chapter 1","lang":"en","site":"mangadex"}
	]`

	server := newChaptersServer(t, 6, rows, "Berserk")

	chapters, err := ListChapters(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("chapters: %v", err)
	}

	if len(chapters) != 5 {
		t.Fatalf("chapters = %d, want 5 (one per distinct number)", len(chapters))
	}

	wantOrder := []string{"1", "2", "3", "3.5", "5"}
	for i, want := range wantOrder {
		if got := chapters[i].Number.String(); got != want {
			t.Errorf("chapter %d = %s, want %s (ascending)", i, got, want)
		}
	}

	// Chapter 3 carries every source in the order the API listed
	// them (comick copy first here).
	if want := server.URL + "/chapter-detail/id-3-b,id-3-a"; chapters[2].URL != want {
		t.Errorf("chapter 3 url = %q, want %q", chapters[2].URL, want)
	}

	// The site label wins when present; blank rows fall back.
	if chapters[0].Label != "Chapter 1" {
		t.Errorf("chapter 1 label = %q", chapters[0].Label)
	}

	if chapters[1].Label != "Chapter 2" {
		t.Errorf("chapter 2 label = %q, want the fallback label", chapters[1].Label)
	}

	if chapters[3].Label != "Chapter 3.5" {
		t.Errorf("chapter 3.5 label = %q", chapters[3].Label)
	}
}

func TestListChaptersSkipsUnusableRows(
	t *testing.T,
) {
	// 1e2-style exponent numbers and id-less rows are skipped; the
	// rest still builds a clean ascending list.
	rows := `[
		{"id":"id-2","number":2,"name":"Chapter 2","lang":"en","site":"mangadex"},
		{"id":"","number":3,"name":"Chapter 3","lang":"en","site":"mangadex"},
		{"id":"id-1","number":1e2,"name":"Chapter 100","lang":"en","site":"mangadex"},
		{"id":"id-0","number":0.5,"name":"Chapter 0.5","lang":"en","site":"mangadex"}
	]`

	server := newChaptersServer(t, 4, rows, "Berserk")

	chapters, err := ListChapters(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("chapters: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(chapters))
	}

	if chapters[0].Number.String() != "0.5" || chapters[1].Number.String() != "2" {
		t.Errorf("chapters = %s, %s", chapters[0].Number.String(), chapters[1].Number.String())
	}
}

func TestListChaptersIncompleteListing(
	t *testing.T,
) {
	// The declared total exceeds one page's worth of rows while
	// the walk cap stops early: the listing is unusable for a
	// download, so this must fail loudly.
	pages := make(map[int]chapterListingResponse)

	for p := 1; p <= maxListingPages; p++ {
		pages[p] = chapterListingResponse{
			Data:               []chapterRow{row(fmt.Sprintf("c%d", p), "1", "en")},
			AvailableLanguages: []string{"en"},
			Pagination:         paginationOf(p, maxListingPages+1, maxListingPages+1),
		}
	}

	capture := newListingServer(t, pages)

	_, err := ListChapters(
		capture.server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)

	if err == nil || !strings.Contains(err.Error(), "listing is incomplete") {
		t.Fatalf("error = %v, want the incomplete-listing failure", err)
	}
}

func TestListChaptersEmptyListing(
	t *testing.T,
) {
	server := newChaptersServer(t, 0, `[]`, "Berserk")

	chapters, err := ListChapters(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("chapters: %v", err)
	}

	if len(chapters) != 0 {
		t.Fatalf("chapters = %d, want 0", len(chapters))
	}
}

func TestListChaptersHTTPError(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "forbidden", http.StatusForbidden)
			},
		),
	)
	defer server.Close()

	_, err := ListChapters(server.URL, "6851544a702284f8341782f2", "en", nil)

	if err == nil || !strings.Contains(err.Error(), "returned status 403") {
		t.Fatalf("error = %v, want the listing status failure", err)
	}
}
