package mangataroscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// listingServer answers the chapters route with pre-built pages and
// records which query parameters each walk page carried.
type listingServer struct {
	server *httptest.Server

	pages []chapterListingResponse

	offsets    []string
	tokens     []string
	timestamps []string
}

func newListingServer(
	t *testing.T,
	pages ...chapterListingResponse,
) *listingServer {
	t.Helper()

	fixture := &listingServer{pages: pages}

	mux := http.NewServeMux()
	mux.HandleFunc(chaptersPath, func(w http.ResponseWriter, r *http.Request) {
		offsetParam := r.URL.Query().Get("offset")

		page := 0
		if offset, err := strconv.Atoi(offsetParam); err == nil {
			page = offset / listingPageLimit
		}

		if page >= len(fixture.pages) {
			t.Errorf("unexpected listing page %d (offset %q)", page, offsetParam)

			return
		}

		fixture.offsets = append(fixture.offsets, offsetParam)
		fixture.tokens = append(fixture.tokens, r.URL.Query().Get("_t"))
		fixture.timestamps = append(fixture.timestamps, r.URL.Query().Get("_ts"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fixture.pages[page])
	})

	fixture.server = httptest.NewServer(mux)
	t.Cleanup(fixture.server.Close)

	return fixture
}

func chapterRowJSON(id string, chapter string, title string, language string) string {
	return fmt.Sprintf(
		`{"id":%q,"chapter":%q,"title":%q,"chapter_type":"media","language":%q,"url":"https://mangataro.org/read/berserk/ch%s-%s"}`,
		id,
		chapter,
		title,
		language,
		strings.ReplaceAll(chapter, ".", "-"),
		id,
	)
}

func listingPage(
	rows []string,
	total int,
	hasMore bool,
) chapterListingResponse {
	payload := "[" + strings.Join(rows, ",") + "]"

	var response chapterListingResponse

	// Re-decode through the real shape so the fixtures keep the
	// exact key names the route answers with.
	if err := json.Unmarshal([]byte(fmt.Sprintf(
		`{"success":true,"chapters":%s,"total":%d,"offset":0,"limit":%d,"has_more":%t}`,
		payload,
		total,
		listingPageLimit,
		hasMore,
	)), &response); err != nil {
		panic(err)
	}

	return response
}

func TestFetchChapterRowsWalksUntilExhausted(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("1", "398", "Chapter 398", "en"),
				chapterRowJSON("2", "397", "Chapter 397", "en"),
			},
			3,
			true,
		),
		listingPage(
			[]string{chapterRowJSON("3", "396", "Chapter 396", "en")},
			3,
			false,
		),
	)

	listing, err := fetchChapterRows(fixture.server.URL, "32148", testDiscardLogger())
	if err != nil {
		t.Fatalf("fetchChapterRows() error: %v", err)
	}

	if len(listing.rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(listing.rows))
	}

	if !listing.complete {
		t.Error("expected the walk to be complete")
	}

	if listing.declaredTotal != 3 {
		t.Errorf("declaredTotal = %d, want 3", listing.declaredTotal)
	}

	if len(fixture.offsets) != 2 {
		t.Fatalf("expected 2 walk pages, got %d", len(fixture.offsets))
	}

	// The second page resumes at the first page's size, one row
	// short of the 500-row offset stride the site's frontend uses.
	if got, want := fixture.offsets[1], strconv.Itoa(listingPageLimit); got != want {
		t.Errorf("second page offset = %q, want %q", got, want)
	}
}

func TestFetchChapterRowsSendsTheToken(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{chapterRowJSON("1", "1", "Chapter 1", "en")},
			1,
			false,
		),
	)

	if _, err := fetchChapterRows(fixture.server.URL, "32148", testDiscardLogger()); err != nil {
		t.Fatalf("fetchChapterRows() error: %v", err)
	}

	if len(fixture.tokens) != 1 || len(fixture.timestamps) != 1 {
		t.Fatalf("expected one tokened request, got %d", len(fixture.tokens))
	}

	// The route rejects empty or malformed credentials, so the
	// requests must carry the locally minted token and timestamp.
	if len(fixture.tokens[0]) != 16 {
		t.Errorf("token = %q, want the 16-character hex form", fixture.tokens[0])
	}

	if fixture.timestamps[0] == "" || fixture.timestamps[0] == "0" {
		t.Errorf("timestamp = %q, want unix seconds", fixture.timestamps[0])
	}
}

func TestFetchChapterRowsCapsHugeWalks(t *testing.T) {
	// Every page claims more; the safety cap stops the walk after
	// maxListingPages pages without spinning forever.
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

	listing, err := fetchChapterRows(fixture.server.URL, "32148", testDiscardLogger())
	if err != nil {
		t.Fatalf("fetchChapterRows() error: %v", err)
	}

	if listing.complete {
		t.Error("expected the capped walk to be incomplete")
	}

	if len(fixture.tokens) != maxListingPages {
		t.Errorf("walked %d pages, want the cap of %d", len(fixture.tokens), maxListingPages)
	}
}

func TestGetLanguagePathsCountsDistinctNumbers(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage(
			[]string{
				chapterRowJSON("1", "1", "Chapter 1", "en"),
				// A duplicate row of chapter 1 must not count.
				chapterRowJSON("2", "1", "Chapter 1", "en"),
				chapterRowJSON("3", "2", "Chapter 2", "en"),
				chapterRowJSON("4", "1", "Capítulo 1", "pt-br"),
			},
			4,
			false,
		),
	)

	paths, err := GetLanguagePaths(fixture.server.URL, "32148", testDiscardLogger())
	if err != nil {
		t.Fatalf("GetLanguagePaths() error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}

	// Labels sort alphabetically; counts come from the listing.
	if paths[0].Code != "en" || paths[0].Label != "English (2 chapters)" {
		t.Errorf("unexpected first path: %+v", paths[0])
	}

	if paths[1].Code != "pt-br" || paths[1].Label != "Portuguese (Brazil) (1 chapter)" {
		t.Errorf("unexpected second path: %+v", paths[1])
	}
}

func TestGetLanguagePathsEmptyListing(t *testing.T) {
	fixture := newListingServer(
		t,
		listingPage([]string{}, 0, false),
	)

	paths, err := GetLanguagePaths(fixture.server.URL, "32148", testDiscardLogger())
	if err != nil {
		t.Fatalf("GetLanguagePaths() error: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("expected no paths, got %+v", paths)
	}
}

func TestGetLanguagePathsCappedWalkDegradesLabels(t *testing.T) {
	// A walk that hit the cap must not present partial numbers as
	// real counts; labels degrade to the bare language name.
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

	paths, err := GetLanguagePaths(fixture.server.URL, "32148", testDiscardLogger())
	if err != nil {
		t.Fatalf("GetLanguagePaths() error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	if paths[0].Label != "English" {
		t.Errorf("label = %q, want the bare language name of a capped walk", paths[0].Label)
	}
}

func TestGetLanguagePathsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		},
	))
	defer server.Close()

	_, err := GetLanguagePaths(server.URL, "32148", testDiscardLogger())
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "status 403") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}
