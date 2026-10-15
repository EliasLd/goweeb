package atsumoescraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetPageImageURLs(t *testing.T) {
	var (
		gotPath  string
		gotManga = make(map[string]string)
	)

	checks := func(r *http.Request) {
		gotPath = r.URL.Path

		for key, values := range r.URL.Query() {
			gotManga[key] = values[0]
		}
	}

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			checks(r)

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"readChapter": {
					"id": "ZfvD517i",
					"number": 2,
					"title": "# 2",
					"scanlationMangaId": "cmgzlbgb59vevm191tbi7vll8",
					"pages": [
						{
							"id": "ZfvD517i-1",
							"image": "/static/pages/jIg3F/ZfvD517i/1.webp",
							"number": 1
						},
						{
							"id": "ZfvD517i-0",
							"image": "/static/pages/jIg3F/ZfvD517i/0.webp",
							"number": 0
						},
						{
							"id": "ZfvD517i-2",
							"image": "/static/pages/jIg3F/ZfvD517i/2.webp",
							"number": 2
						},
						{
							"id": "ZfvD517i-3",
							"image": "   ",
							"number": 3
						},
						{
							"id": "ZfvD517i-4",
							"image": "ftp://bad.example.net/static/pages/jIg3F/ZfvD517i/4.webp",
							"number": 4
						},
						{
							"id": "ZfvD517i-5",
							"image": "https://mirror.example.net/static/pages/jIg3F/ZfvD517i/5.webp",
							"number": 5
						}
					]
				}
			}`)
		},
	))
	defer server.Close()

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	entryURL := server.URL +
		"/api/read/chapter?mangaId=jIg3F&chapterId=ZfvD517i"

	imageURLs, err := GetPageImageURLs(entryURL, server.URL, nil)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	// Pages are ordered from the first to the last one. Relative
	// image paths point to the CDN host; empty and unsupported
	// image entries are skipped; absolute URLs are kept as-is.
	expected := []string{
		"http://cdn." + base.Host +
			"/static/pages/jIg3F/ZfvD517i/0.webp",
		"http://cdn." + base.Host +
			"/static/pages/jIg3F/ZfvD517i/1.webp",
		"http://cdn." + base.Host +
			"/static/pages/jIg3F/ZfvD517i/2.webp",
		"https://mirror.example.net/static/pages/jIg3F/ZfvD517i/5.webp",
	}

	if len(imageURLs) != len(expected) {
		t.Fatalf(
			"expected %d image URLs, got %d",
			len(expected),
			len(imageURLs),
		)
	}

	for i, want := range expected {
		if imageURLs[i] != want {
			t.Errorf(
				"image %d: expected %q, got %q",
				i,
				want,
				imageURLs[i],
			)
		}
	}

	if gotPath != readChapterPath {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if gotManga["mangaId"] != "jIg3F" {
		t.Errorf("unexpected mangaId param: %q", gotManga["mangaId"])
	}

	if gotManga["chapterId"] != "ZfvD517i" {
		t.Errorf("unexpected chapterId param: %q", gotManga["chapterId"])
	}
}

func TestGetPageImageURLsMissingIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"readChapter": {"pages": []}}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL+"/api/read/chapter?chapterId=ZfvD517i",
		server.URL,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error when mangaId is missing")
	}

	_, err = GetPageImageURLs(
		server.URL+"/api/read/chapter?mangaId=jIg3F",
		server.URL,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error when chapterId is missing")
	}
}

func TestGetPageImageURLsUnexpectedEntryURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL+"/manga/jIg3F",
		server.URL,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected entry URL")
	}
}

func TestGetPageImageURLsNoPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = fmt.Fprint(w, `{
				"readChapter": {
					"id": "ZfvD517i",
					"pages": []
				}
			}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL+"/api/read/chapter?mangaId=jIg3F&chapterId=ZfvD517i",
		server.URL,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error when no page images exist")
	}
}

func TestGetPageImageURLsHTTPError(t *testing.T) {
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

	_, err := GetPageImageURLs(
		server.URL+"/api/read/chapter?mangaId=jIg3F&chapterId=ZfvD517i",
		server.URL,
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for HTTP 404")
	}
}

func TestGetPageImageURLsInvalidJSON(t *testing.T) {
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

	_, err := GetPageImageURLs(
		server.URL+"/api/read/chapter?mangaId=jIg3F&chapterId=ZfvD517i",
		server.URL,
		nil,
	)
	if err == nil {
		t.Fatal("expected a JSON decoding error")
	}
}

func TestGetPageImageURLsInvalidDomain(t *testing.T) {
	_, err := GetPageImageURLs(
		"https://atsu.moe/api/read/chapter?mangaId=jIg3F&chapterId=ZfvD517i",
		"invalid-domain",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an invalid domain")
	}
}
