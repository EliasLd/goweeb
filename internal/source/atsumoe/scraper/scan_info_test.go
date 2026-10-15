package atsumoescraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetScanlators(t *testing.T) {
	var (
		gotPath  string
		gotQuery = make(map[string]string)
	)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			for key, values := range r.URL.Query() {
				gotQuery[key] = values[0]
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"mangaPage": {
					"id": "nh6Ii",
					"title": "One-Punch Man",
					"scanlators": [
						{
							"id": "cmgzli2toc4crm191ejjkdsxw",
							"name": "Alpha"
						},
						{
							"id": "cmod52vbl000065qi1084bnvb",
							"name": "   Delta   "
						},
						{
							"id": "   ",
							"name": "Ghost"
						},
						{
							"id": "cmnoID",
							"name": ""
						}
					]
				}
			}`)
		},
	))
	defer server.Close()

	scanlators, err := GetScanlators(server.URL, "nh6Ii", nil)
	if err != nil {
		t.Fatalf("GetScanlators() error: %v", err)
	}

	if len(scanlators) != 3 {
		t.Fatalf(
			"expected 3 scanlators, got %d",
			len(scanlators),
		)
	}

	// The empty ID is skipped and the missing name falls back to the
	// scanlator ID.
	expected := []Scanlator{
		{ID: "cmgzli2toc4crm191ejjkdsxw", Name: "Alpha"},
		{ID: "cmod52vbl000065qi1084bnvb", Name: "Delta"},
		{ID: "cmnoID", Name: "cmnoID"},
	}

	for i, want := range expected {
		if scanlators[i] != want {
			t.Errorf(
				"scanlator %d: expected %+v, got %+v",
				i,
				want,
				scanlators[i],
			)
		}
	}

	if gotPath != mangaPagePath {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if gotQuery["id"] != "nh6Ii" {
		t.Errorf("unexpected id param: %q", gotQuery["id"])
	}
}

func TestGetScanlatorsNoScanlators(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"mangaPage": {
					"id": "jIg3F",
					"scanlators": []
				}
			}`)
		},
	))
	defer server.Close()

	scanlators, err := GetScanlators(server.URL, "jIg3F", nil)
	if err != nil {
		t.Fatalf("GetScanlators() error: %v", err)
	}

	if len(scanlators) != 0 {
		t.Fatalf(
			"expected no scanlators, got %d",
			len(scanlators),
		)
	}
}

func TestGetScanlatorsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"not found",
				http.StatusNotFound,
			)
		},
	))
	defer server.Close()

	_, err := GetScanlators(server.URL, "jIg3F", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 404")
	}
}

func TestGetScanlatorsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `not json`)
		},
	))
	defer server.Close()

	_, err := GetScanlators(server.URL, "jIg3F", nil)
	if err == nil {
		t.Fatal("expected a JSON decoding error")
	}
}

func TestGetMangaTitle(t *testing.T) {
	var (
		gotPath  string
		gotQuery = make(map[string]string)
	)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			for key, values := range r.URL.Query() {
				gotQuery[key] = values[0]
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"id": "jIg3F",
				"title": "  Solanin  ",
				"type": "Manga",
				"medium": "Comic",
				"isAdult": false,
				"chapters": []
			}`)
		},
	))
	defer server.Close()

	title, err := GetMangaTitle(server.URL, "jIg3F", nil)
	if err != nil {
		t.Fatalf("GetMangaTitle() error: %v", err)
	}

	if title != "Solanin" {
		t.Errorf("unexpected title: %q", title)
	}

	if gotPath != mangaInfoPath {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if gotQuery["mangaId"] != "jIg3F" {
		t.Errorf("unexpected mangaId param: %q", gotQuery["mangaId"])
	}
}

func TestGetMangaTitleMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"id": "jIg3F",
				"title": "   "
			}`)
		},
	))
	defer server.Close()

	_, err := GetMangaTitle(server.URL, "jIg3F", nil)
	if err == nil {
		t.Fatal("expected an error when the title is missing")
	}
}

func TestGetMangaTitleHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		},
	))
	defer server.Close()

	_, err := GetMangaTitle(server.URL, "jIg3F", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 500")
	}
}
