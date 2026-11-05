package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// listingResponse wraps chapter rows in the API envelope a listing
// page answers with. total must match the served rows (the real API
// reports 0 for an empty listing).
func listingResponse(rowsJSON string, total int) string {
	return fmt.Sprintf(
		`{"status":"success","code":200,"data":`+rowsJSON+
			`,"available_languages":["en","fr"],`+
			`"pagination":{"page":1,"limit":5000,"total":%d,"total_pages":1}}`,
		total,
	)
}

// paginationOf builds the listing metadata a server page answers with.
func paginationOf(
	page int,
	total int,
	totalPages int,
) listingPagination {
	return listingPagination{
		Page:       page,
		Total:      total,
		TotalPages: totalPages,
	}
}

// listingServer serves a chapter listing split over pages. Each page
// is produced by its own function; the requested page number is
// checked per request.
type listingServer struct {
	server      *httptest.Server
	gotQueries  []url.Values
	gotTitleIDs []string
}

func newListingServer(
	t *testing.T,
	pages map[int]chapterListingResponse,
) *listingServer {
	t.Helper()

	captured := &listingServer{}

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				captured.gotQueries = append(
					captured.gotQueries,
					query,
				)
				captured.gotTitleIDs = append(
					captured.gotTitleIDs,
					query.Get("title_id"),
				)

				page, err := strconv.Atoi(query.Get("page"))
				if err != nil {
					page = 1
				}

				response, ok := pages[page]
				if !ok {
					http.NotFound(w, r)
					return
				}

				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				_ = json.NewEncoder(w).Encode(response)
			},
		),
	)

	t.Cleanup(server.Close)
	captured.server = server

	return captured
}

func row(id string, number string, lang string) chapterRow {
	return chapterRow{
		ID:     id,
		Number: json.Number(number),
		Name:   fmt.Sprintf("Chapter %s", number),
		Lang:   lang,
		Site:   "mangadex",
	}
}

func TestGetLanguagePathsCountsDistinctNumbers(
	t *testing.T,
) {
	capture := newListingServer(t, map[int]chapterListingResponse{
		1: {
			Data: []chapterRow{
				row("c1", "1", "en"),
				row("c2", "1", "en"), // duplicate number
				row("c3", "2", "en"),
				row("c4", "1", "es"),
				row("c5", "2", "es"),
			},
			AvailableLanguages: []string{"en", "es"},
			Pagination:         paginationOf(1, 5, 1),
		},
	})

	paths, err := GetLanguagePaths(capture.server.URL, "6851544a702284f8341782f2", nil)
	if err != nil {
		t.Fatalf("languages: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("paths = %d, want 2", len(paths))
	}

	// Two distinct numbers for English, two for Spanish; the
	// duplicate row must not inflate the count.
	if paths[0].Code != "en" || paths[0].Label != "English (2 chapters)" {
		t.Errorf("first path = %q / %q", paths[0].Code, paths[0].Label)
	}

	if paths[1].Code != "es" || paths[1].Label != "Spanish (2 chapters)" {
		t.Errorf("second path = %q / %q", paths[1].Code, paths[1].Label)
	}

	if len(capture.gotQueries) != 1 {
		t.Fatalf("requests = %d, want 1 (single page listing)", len(capture.gotQueries))
	}

	query := capture.gotQueries[0]

	if got := query.Get("title_id"); got != "6851544a702284f8341782f2" {
		t.Errorf("title_id = %q", got)
	}

	if got := query.Get("limit"); got != strconv.Itoa(listingPageLimit) {
		t.Errorf("limit = %q, want the largest page the API accepts", got)
	}

	if got := query.Get("language"); got != "" {
		t.Errorf("language = %q, want unset (unfiltered listing)", got)
	}

	if got := query.Get("page"); got != "" {
		t.Errorf("page = %q, want unset on the first request", got)
	}
}

func TestGetLanguagePathsWalksPages(
	t *testing.T,
) {
	capture := newListingServer(t, map[int]chapterListingResponse{
		1: {
			Data: []chapterRow{
				row("c1", "100", "en"),
			},
			AvailableLanguages: []string{"en"},
			Pagination:         paginationOf(1, 2, 2),
		},
		2: {
			Data: []chapterRow{
				row("c2", "1", "en"),
				row("c3", "1", "fr"),
			},
			AvailableLanguages: nil,
			Pagination:         paginationOf(2, 3, 2),
		},
	})

	paths, err := GetLanguagePaths(capture.server.URL, "6851544a702284f8341782f2", nil)
	if err != nil {
		t.Fatalf("languages: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("paths = %d, want 2", len(paths))
	}

	// French has one distinct number: singular label. Row-only
	// languages (declared on page 2 with no available_languages
	// refresh) still count.
	if paths[0].Label != "English (2 chapters)" {
		t.Errorf("english label = %q", paths[0].Label)
	}

	if paths[1].Code != "fr" || paths[1].Label != "French (1 chapter)" {
		t.Errorf("french path = %q / %q", paths[1].Code, paths[1].Label)
	}

	if len(capture.gotQueries) != 2 {
		t.Fatalf("requests = %d, want 2", len(capture.gotQueries))
	}

	if got := capture.gotQueries[1].Get("page"); got != "2" {
		t.Errorf("second request page = %q, want 2", got)
	}
}

func TestGetLanguagePathsDegradesWhenCapped(
	t *testing.T,
) {
	// The API declares more data than the walk is allowed to take
	// (maxListingPages > 1 page requested here by declaring
	// total_pages just beyond the cap for this test is not needed:
	// a total larger than the delivered rows marks incompleteness).
	pages := make(map[int]chapterListingResponse)

	for p := 1; p <= maxListingPages; p++ {
		pages[p] = chapterListingResponse{
			Data: []chapterRow{
				row(fmt.Sprintf("c%d", p), "1", "en"),
			},
			AvailableLanguages: []string{"en"},
			Pagination:         paginationOf(p, maxListingPages+1, maxListingPages+1),
		}
	}

	capture := newListingServer(t, pages)

	paths, err := GetLanguagePaths(capture.server.URL, "6851544a702284f8341782f2", nil)
	if err != nil {
		t.Fatalf("languages: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("paths = %d, want 1", len(paths))
	}

	// The count lies about 21 chapters when only 20 are seen: the
	// label must degrade to the plain language name.
	if paths[0].Label != "English" {
		t.Errorf("label = %q, want a count-less label when the walk was capped", paths[0].Label)
	}
}

func TestGetLanguagePathsUnknownCodeFallback(
	t *testing.T,
) {
	capture := newListingServer(t, map[int]chapterListingResponse{
		1: {
			Data: []chapterRow{
				row("c1", "1", "zz"),
			},
			AvailableLanguages: []string{"zz"},
			Pagination:         paginationOf(1, 1, 1),
		},
	})

	paths, err := GetLanguagePaths(capture.server.URL, "6851544a702284f8341782f2", nil)
	if err != nil {
		t.Fatalf("languages: %v", err)
	}

	if len(paths) != 1 || paths[0].Label != "zz (1 chapter)" {
		t.Fatalf("paths = %+v, want the raw code as name", paths)
	}
}

func TestGetLanguagePathsNoChapters(
	t *testing.T,
) {
	capture := newListingServer(t, map[int]chapterListingResponse{
		1: {
			Data:               nil,
			AvailableLanguages: nil,
			Pagination:         paginationOf(1, 0, 0),
		},
	})

	paths, err := GetLanguagePaths(capture.server.URL, "6851544a702284f8341782f2", nil)
	if err != nil {
		t.Fatalf("languages: %v", err)
	}

	if len(paths) != 0 {
		t.Fatalf("paths = %d, want 0", len(paths))
	}
}

func TestGetLanguagePathsHTTPError(
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

	_, err := GetLanguagePaths(server.URL, "6851544a702284f8341782f2", nil)

	if err == nil || !strings.Contains(err.Error(), "returned status 403") {
		t.Fatalf("error = %v, want a brand-named status failure", err)
	}
}

func TestTitleIDFromURL(t *testing.T) {
	tests := []struct {
		name    string
		workURL string
		want    string
		wantErr bool
	}{
		{
			name:    "canonical title page",
			workURL: "https://mangaball.com/title-detail/6851544a702284f8341782f2-berserk",
			want:    "6851544a702284f8341782f2",
		},
		{
			name:    "slug with dashes",
			workURL: "https://mangaball.com/title-detail/68516a86702284f83417c943-childhood-friend-complex",
			want:    "68516a86702284f83417c943",
		},
		{
			name:    "no slug",
			workURL: "https://mangaball.com/title-detail/68516a86702284f83417c943",
			want:    "68516a86702284f83417c943",
		},
		{
			name:    "trailing slash",
			workURL: "https://mangaball.com/title-detail/68516a86702284f83417c943-berserk/",
			want:    "68516a86702284f83417c943",
		},
		{
			name:    "wrong first segment",
			workURL: "https://mangaball.com/chapter-detail/68516a86702284f83417c943",
			wantErr: true,
		},
		{
			name:    "missing id",
			workURL: "https://mangaball.com/title-detail/",
			wantErr: true,
		},
		{
			name:    "garbage url",
			workURL: "://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TitleIDFromURL(tt.workURL)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("TitleIDFromURL(%q) = %q, want error", tt.workURL, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("TitleIDFromURL(%q): %v", tt.workURL, err)
			}

			if got != tt.want {
				t.Errorf("TitleIDFromURL(%q) = %q, want %q", tt.workURL, got, tt.want)
			}
		})
	}
}
