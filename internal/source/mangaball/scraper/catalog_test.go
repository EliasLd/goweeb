package mangaballscraper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveSearch(
	t *testing.T,
	response string,
) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(
					"Content-Type",
					"application/json",
				)
				w.Write([]byte(response))
			},
		),
	)

	t.Cleanup(server.Close)

	return server
}

func TestSearchCatalogSendsSearchPayload(
	t *testing.T,
) {
	var (
		gotMethod  string
		gotPath    string
		gotPayload searchRequest
		gotUA      string
		gotAccept  string
	)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotUA = r.Header.Get("User-Agent")
				gotAccept = r.Header.Get("Accept")

				if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
					t.Errorf("request payload: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"code":200,"status":"success","data":[` +
					`{"id":"6851544a702284f8341782f2","name":"Berserk","slug":"berserk"},` +
					`{"id":"685270ea6908629fc2c82502","name":" Otonari Complex ","slug":"otomari-complex"}]}`))
			},
		),
	)
	defer server.Close()

	results, err := SearchCatalog(server.URL, " berserk ", nil)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (the search is client-side over a POST API)", gotMethod)
	}

	if gotPath != "/api/v1/title/search" {
		t.Errorf("path = %q, want /api/v1/title/search", gotPath)
	}

	if gotPayload.Keyword != "berserk" {
		t.Errorf("keyword = %q, want the trimmed query", gotPayload.Keyword)
	}

	if gotPayload.Limit != maxSearchResults {
		t.Errorf("limit = %d, want %d", gotPayload.Limit, maxSearchResults)
	}

	if gotUA != "Mozilla/5.0" {
		t.Errorf("user agent = %q, want the browser one the API answers", gotUA)
	}

	if gotAccept != "application/json" {
		t.Errorf("accept = %q, want application/json", gotAccept)
	}

	if results[0].Title != "Berserk" {
		t.Errorf("title = %q, want Berserk", results[0].Title)
	}

	if want := server.URL + "/title-detail/6851544a702284f8341782f2-berserk"; results[0].URL != want {
		t.Errorf("url = %q, want %q", results[0].URL, want)
	}

	// Titles are trimmed but kept verbatim otherwise.
	if results[1].Title != "Otonari Complex" {
		t.Errorf("title = %q, want the trimmed site title", results[1].Title)
	}
}

func TestSearchCatalogSkipsIncompleteRows(
	t *testing.T,
) {
	server := serveSearch(t, `{"code":200,"status":"success","data":[`+
		`{"id":"","name":"No title id","slug":"no-title-id"},`+
		`{"id":"685270ea6908629fc2c82502","name":"No slug","slug":""},`+
		`{"id":"6851544a702284f8341782f2","name":"","slug":"no-name"},`+
		`{"id":"685270ea6908629fc2c82502","name":"Berserk","slug":"berserk"}]}`)

	results, err := SearchCatalog(server.URL, "berserk", nil)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}

	if results[0].Title != "Berserk" {
		t.Errorf("title = %q, want Berserk", results[0].Title)
	}
}

func TestSearchCatalogEmptyQuery(
	t *testing.T,
) {
	server := serveSearch(t, `{"code":200,"status":"success","data":[]}`)

	results, err := SearchCatalog(server.URL, "   ", nil)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("results = %d, want 0 (a blank query must not reach the API)", len(results))
	}
}

func TestSearchCatalogHTTPError(
	t *testing.T,
) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "blocked", http.StatusInternalServerError)
			},
		),
	)
	defer server.Close()

	_, err := SearchCatalog(server.URL, "berserk", nil)

	if err == nil || !strings.Contains(err.Error(), "returned status 500") {
		t.Fatalf("error = %v, want a brand-named status failure", err)
	}
}

func TestSearchCatalogInvalidJSON(
	t *testing.T,
) {
	server := serveSearch(t, "200 not json")

	_, err := SearchCatalog(server.URL, "berserk", nil)

	if err == nil || !strings.Contains(err.Error(), "failed to decode MangaBall search response") {
		t.Fatalf("error = %v, want a decode failure", err)
	}
}

func TestSearchCatalogInvalidDomain(
	t *testing.T,
) {
	_, err := SearchCatalog("", "berserk", nil)

	if err == nil || !strings.Contains(err.Error(), "invalid MangaBall domain") {
		t.Fatalf("error = %v, want an invalid domain failure", err)
	}
}
