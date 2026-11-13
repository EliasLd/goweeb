package mangataroscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChapterIDFromEntryURL(t *testing.T) {
	cases := []struct {
		name     string
		entryURL string
		expected string
		wantErr  bool
	}{
		{
			name:     "plain chapter",
			entryURL: "https://mangataro.org/read/berserk/ch398-471959",
			expected: "471959",
		},
		{
			name:     "dashed decimal number",
			entryURL: "https://mangataro.org/read/berserk/ch0-1-32150",
			expected: "32150",
		},
		{
			name:     "trailing slash",
			entryURL: "https://mangataro.org/read/berserk/ch398-471959/",
			expected: "471959",
		},
		{
			name:     "unexpected path",
			entryURL: "https://mangataro.org/manga/berserk/ch398-471959",
			wantErr:  true,
		},
		{
			name:     "no id after the dash",
			entryURL: "https://mangataro.org/read/berserk/ch398-",
			wantErr:  true,
		},
		{
			name:     "no dash at all",
			entryURL: "https://mangataro.org/read/berserk/ch398",
			wantErr:  true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			id, err := ChapterIDFromEntryURL(testCase.entryURL)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", testCase.entryURL)
				}

				return
			}

			if err != nil {
				t.Fatalf("ChapterIDFromEntryURL() error: %v", err)
			}

			if id != testCase.expected {
				t.Errorf("expected id %q, got %q", testCase.expected, id)
			}
		})
	}
}

func TestGetPageImageURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("chapter_id") != "471959" {
				t.Errorf("chapter_id = %q, want 471959", r.URL.Query().Get("chapter_id"))
			}

			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{"success":true,"chapter_id":471959,"chapter_type":"media","images":[`+
				`"https:\/\/mangataro.yachts\/storage\/chapters\/79e5...\/001.webp",`+
				`"https:\/\/mangataro.yachts\/storage\/chapters\/79e5...\/002.webp",`+
				`"https:\/\/mangataro.yachts\/storage\/chapters\/79e5...\/024.webp"]}`)
		},
	))
	defer server.Close()

	imageURLs, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/read/berserk/ch398-471959",
		nil,
	)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	if len(imageURLs) != 3 {
		t.Fatalf("expected 3 image URLs, got %d", len(imageURLs))
	}

	if imageURLs[0] == imageURLs[1] {
		t.Error("expected the pages in the payload's ordered positions")
	}

	if !strings.HasSuffix(imageURLs[2], "024.webp") {
		t.Errorf("unexpected last page: %q", imageURLs[2])
	}
}

func TestGetPageImageURLsFiltersNonHTTPEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{"success":true,"chapter_type":"media","images":[`+
				`"https://mangataro.yachts/storage/x/001.webp",`+
				`"  ",`+
				`"/storage/relative/002.webp",`+
				`"ftp://mangataro.yachts/storage/x/003.webp"]}`)
		},
	))
	defer server.Close()

	imageURLs, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/read/berserk/ch1-1",
		nil,
	)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	if len(imageURLs) != 1 {
		t.Fatalf("expected only the absolute http url to survive, got %v", imageURLs)
	}
}

func TestGetPageImageURLsTextChapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w,
				`{"success":true,"chapter_id":327142,"chapter_type":"text","content":"<p>Only I Level Up</p>"}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/read/solo-leveling-novel/ch270-327142",
		nil,
	)
	if err == nil {
		t.Fatal("expected the text chapter to be refused")
	}

	if !strings.Contains(err.Error(), "text chapter") {
		t.Errorf("expected a text-chapter error, got %v", err)
	}
}

func TestGetPageImageURLsEmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"success":true,"chapter_type":"media","images":[]}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/read/berserk/ch1-1",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for a chapter without pages")
	}

	if !strings.Contains(err.Error(), "no page images") {
		t.Errorf("expected the empty-list error, got %v", err)
	}
}

func TestGetPageImageURLsAllRelativeEntriesFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w,
				`{"success":true,"chapter_type":"media","images":["/storage/x/001.webp"]}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/read/berserk/ch1-1",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error when nothing survives the filter")
	}
}

func TestGetPageImageURLsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"success":false,"message":"Chapter not found"}`)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/read/berserk/ch999-999999",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for the 404")
	}

	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}

func TestGetPageImageURLsInvalidEntryURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("no request expected for an invalid entry URL")
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL,
		"https://mangataro.org/not-a-chapter/471959",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected entry URL")
	}
}
