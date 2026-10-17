package mangadotnetscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetPageImageURLsUserSource(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
				"chapter": {"id": 888840, "page_count": 3},
				"manga": {"id": 101, "title": "Blue Lock"},
				"images": [
					{
						"url": "/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/001.webp",
						"w": 1244,
						"h": 1920,
						"filename": "001.webp"
					},
					{
						"url": "/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/002.jpg",
						"w": 1244,
						"h": 1920,
						"filename": "002.jpg"
					},
					{
						"url": "/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/003.png",
						"w": 1244,
						"h": 1920,
						"filename": "003.png"
					}
				],
				"prev_chapter_id": null,
				"next_chapter_id": 888841
			}`)
		},
	))
	defer server.Close()

	entryURL := server.URL + "/chapter/888840?source=user"

	imageURLs, err := GetPageImageURLs(entryURL, server.URL, nil)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	// User uploads are served from the /api/uploads endpoint.
	expectedPath := "/api/uploads/888840/images"

	if gotPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, gotPath)
	}

	if len(imageURLs) != 3 {
		t.Fatalf("expected 3 image URLs, got %d", len(imageURLs))
	}

	// Relative paths resolve against the (test) site domain and the
	// API order is preserved.
	expectedFirst := server.URL +
		"/chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/001.webp"

	if imageURLs[0] != expectedFirst {
		t.Errorf("expected URL %q, got %q", expectedFirst, imageURLs[0])
	}

	if !strings.HasSuffix(imageURLs[1], "002.jpg") ||
		!strings.HasSuffix(imageURLs[2], "003.png") {
		t.Errorf("unexpected URLs: %q", imageURLs)
	}
}

func TestGetPageImageURLsScraperSource(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
				"chapter": {
					"id": 798152,
					"page_count": 3,
					"images_folder": "chapter_340_2"
				},
				"manga": {"id": 101, "title": "Blue Lock"},
				"images": [
					{
						"url": "/chapters/manga_101/chapter_340_2/001.png",
						"w": 960,
						"h": 1378
					},
					{
						"url": "https://cdn.example.org/mirror/chapter_340_2/002.png",
						"w": 960,
						"h": 1378
					},
					{
						"url": "javascript:alert(1)"
					},
					{
						"url": ""
					},
					{
						"url": "/chapters/manga_101/chapter_340_2/003.png",
						"w": 960,
						"h": 1378
					}
				]
			}`)
		},
	))
	defer server.Close()

	// Without the source parameter, the chapter is an imported one;
	// its pages are served from the /api/chapters endpoint.
	entryURL := server.URL + "/chapter/798152"

	imageURLs, err := GetPageImageURLs(entryURL, server.URL, nil)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	expectedPath := "/api/chapters/798152/images"

	if gotPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, gotPath)
	}

	if len(imageURLs) != 3 {
		t.Fatalf("expected 3 image URLs, got %d", len(imageURLs))
	}

	// Absolute URLs pass through unchanged.
	expectedSecond := "https://cdn.example.org/mirror/chapter_340_2/002.png"

	if imageURLs[1] != expectedSecond {
		t.Errorf(
			"expected absolute URL pass-through %q, got %q",
			expectedSecond,
			imageURLs[1],
		)
	}
}

func TestGetPageImageURLsEmptyImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
				"chapter": {"id": 888840, "page_count": 0},
				"manga": {"id": 101, "title": "Blue Lock"},
				"images": []
			}`)
		},
	))
	defer server.Close()

	entryURL := server.URL + "/chapter/888840?source=user"

	_, err := GetPageImageURLs(entryURL, server.URL, nil)
	if err == nil {
		t.Fatal("expected an error for an empty image list")
	}

	if !strings.Contains(err.Error(), "no mangadot.net page images") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetPageImageURLsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"service unavailable",
				http.StatusServiceUnavailable,
			)
		},
	))
	defer server.Close()

	entryURL := server.URL + "/chapter/888840?source=user"

	_, err := GetPageImageURLs(entryURL, server.URL, nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 503")
	}
}

func TestParseChapterEntry(t *testing.T) {
	cases := []struct {
		name      string
		entryURL  string
		chapterID string
		user      bool
		wantErr   bool
	}{
		{
			name:      "scraper chapter",
			entryURL:  "https://mangadot.net/chapter/798152",
			chapterID: "798152",
		},
		{
			name:      "user chapter with source parameter",
			entryURL:  "https://mangadot.net/chapter/888840?source=user",
			chapterID: "888840",
			user:      true,
		},
		{
			name:      "ignores other parameters",
			entryURL:  "https://mangadot.net/chapter/888840?page=2&bogus=1",
			chapterID: "888840",
		},
		{
			name:      "trailing slash",
			entryURL:  "https://mangadot.net/chapter/888840/",
			chapterID: "888840",
		},
		{
			name:     "URL without chapter prefix",
			entryURL: "https://mangadot.net/888840",
			wantErr:  true,
		},
		{
			name:     "URL with extra path segments",
			entryURL: "https://mangadot.net/chapter/888840/images",
			wantErr:  true,
		},
		{
			name:     "non-numeric id",
			entryURL: "https://mangadot.net/chapter/abc",
			wantErr:  true,
		},
		{
			name:     "empty id",
			entryURL: "https://mangadot.net/chapter/",
			wantErr:  true,
		},
		{
			name:     "empty URL",
			entryURL: "  ",
			wantErr:  true,
		},
		{
			name:     "invalid URL",
			entryURL: "://not-a-url",
			wantErr:  true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			chapterID, user, err := parseChapterEntry(testCase.entryURL)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf(
						"expected an error for %q",
						testCase.entryURL,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"parseChapterEntry(%q) error: %v",
					testCase.entryURL,
					err,
				)
			}

			if chapterID != testCase.chapterID {
				t.Errorf(
					"expected chapter ID %q, got %q",
					testCase.chapterID,
					chapterID,
				)
			}

			if user != testCase.user {
				t.Errorf("expected user %v, got %v", testCase.user, user)
			}
		})
	}
}
