package atsumoescraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testChaptersPayload = `{
	"chapters": [
		{
			"id": "c29",
			"scanlationMangaId": "S1",
			"title": "# 29",
			"number": 29,
			"index": 29,
			"pageCount": 49,
			"progress": null
		},
		{
			"id": "c2",
			"scanlationMangaId": "S1",
			"title": "# 2",
			"number": 2,
			"index": 2,
			"pageCount": 14
		},
		{
			"id": "c65",
			"scanlationMangaId": "S2",
			"title": "# 6.5",
			"number": 6.5,
			"index": 3,
			"pageCount": 2
		},
		{
			"id": "c1",
			"scanlationMangaId": "S1",
			"title": "",
			"number": 1,
			"index": 1,
			"pageCount": 25
		},
		{
			"id": "c9",
			"scanlationMangaId": "S2",
			"title": "# 9",
			"number": 9,
			"index": 9,
			"pageCount": 10
		},
		{
			"id": "c0",
			"scanlationMangaId": "S1",
			"title": "# 0",
			"number": 0,
			"index": 0,
			"pageCount": 0
		},
		{
			"id": "cnull",
			"scanlationMangaId": "S1",
			"title": "# unknown",
			"number": null,
			"index": 4,
			"pageCount": 5
		}
	]
}`

func chaptersTestServer(
	t *testing.T,
	checks func(path string, mangaID string),
	body string,
	status int,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			checks(
				r.URL.Path,
				r.URL.Query().Get("mangaId"),
			)

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			w.WriteHeader(status)
			_, _ = fmt.Fprint(w, body)
		},
	))
}

func TestListChaptersScanFilter(t *testing.T) {
	var (
		gotPath    string
		gotMangaID string
	)

	checks := func(path string, mangaID string) {
		gotPath = path
		gotMangaID = mangaID
	}

	server := chaptersTestServer(
		t,
		checks,
		testChaptersPayload,
		http.StatusOK,
	)
	defer server.Close()

	chapters, err := ListChapters(server.URL, "CM0wz", "S1", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 3 {
		t.Fatalf(
			"expected 3 chapters, got %d",
			len(chapters),
		)
	}

	// Chapters are sorted from the oldest to the newest and belong to
	// the requested scanlation group only.
	expectedNumbers := []string{
		"1",
		"2",
		"29",
	}

	expectedLabels := []string{
		"Chapter 1",
		"# 2",
		"# 29",
	}

	for i, chapter := range chapters {
		if got := chapter.Number.String(); got != expectedNumbers[i] {
			t.Errorf(
				"chapter %d: expected number %q, got %q",
				i,
				expectedNumbers[i],
				got,
			)
		}

		if got := chapter.Label; got != expectedLabels[i] {
			t.Errorf(
				"chapter %d: expected label %q, got %q",
				i,
				expectedLabels[i],
				got,
			)
		}

		if chapter.URL == "" {
			t.Errorf("chapter %d: expected a URL", i)
		}
	}

	firstURL := chapters[0].URL
	expectedURL := server.URL +
		"/api/read/chapter?chapterId=c1&mangaId=CM0wz"

	if firstURL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, firstURL)
	}

	if gotPath != allChaptersPath {
		t.Errorf("unexpected path: %s", gotPath)
	}

	if gotMangaID != "CM0wz" {
		t.Errorf("unexpected mangaId param: %q", gotMangaID)
	}
}

func TestListChaptersAllScanlations(t *testing.T) {
	server := chaptersTestServer(
		t,
		func(_ string, _ string) {},
		testChaptersPayload,
		http.StatusOK,
	)
	defer server.Close()

	chapters, err := ListChapters(server.URL, "CM0wz", "", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	// Chapters from every scanlation group are kept, sorted by
	// chapter number. Chapters without pages or with an invalid
	// number are skipped.
	expectedNumbers := []string{
		"1",
		"2",
		"6.5",
		"9",
		"29",
	}

	if len(chapters) != len(expectedNumbers) {
		t.Fatalf(
			"expected %d chapters, got %d",
			len(expectedNumbers),
			len(chapters),
		)
	}

	for i, chapter := range chapters {
		if got := chapter.Number.String(); got != expectedNumbers[i] {
			t.Errorf(
				"chapter %d: expected number %q, got %q",
				i,
				expectedNumbers[i],
				got,
			)
		}
	}
}

func TestListChaptersScanFilterNoMatch(t *testing.T) {
	server := chaptersTestServer(
		t,
		func(_ string, _ string) {},
		testChaptersPayload,
		http.StatusOK,
	)
	defer server.Close()

	chapters, err := ListChapters(server.URL, "CM0wz", "S3", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 0 {
		t.Fatalf(
			"expected no chapters, got %d",
			len(chapters),
		)
	}
}

func TestListChaptersHTTPError(t *testing.T) {
	server := chaptersTestServer(
		t,
		func(_ string, _ string) {},
		"service unavailable",
		http.StatusServiceUnavailable,
	)
	defer server.Close()

	_, err := ListChapters(server.URL, "CM0wz", "S1", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 503")
	}
}

func TestListChaptersInvalidJSON(t *testing.T) {
	server := chaptersTestServer(
		t,
		func(_ string, _ string) {},
		"not json",
		http.StatusOK,
	)
	defer server.Close()

	_, err := ListChapters(server.URL, "CM0wz", "S1", nil)
	if err == nil {
		t.Fatal("expected a JSON decoding error")
	}
}

func TestListChaptersInvalidDomain(t *testing.T) {
	_, err := ListChapters("invalid-domain", "CM0wz", "S1", nil)
	if err == nil {
		t.Fatal("expected an error for an invalid domain")
	}
}
