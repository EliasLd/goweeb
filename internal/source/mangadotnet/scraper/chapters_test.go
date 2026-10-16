package mangadotnetscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	ch1UserID = "888840"
	ch1ScrID  = "798152"
)

func chaptersHandler(t *testing.T, gotPath, gotQuery *string, body string) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		*gotPath = r.URL.Path
		*gotQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}
}

func TestListChaptersGroupFilter(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[
		{
			"id": 890000,
			"chapter_number": 2.5,
			"chapter_title": "Chapter 2.5",
			"language": "en",
			"group_id": 18210,
			"group_name": "K Manga",
			"page_count": 20,
			"source": "user",
			"date_added": "2026-10-13 16:17:51.726423+00",
			"volume_number": null,
			"comment_count": 0
		},
		{
			"id": 889999,
			"chapter_number": 2,
			"chapter_title": "Chapter 2",
			"language": "en",
			"group_id": 18210,
			"group_name": "K Manga",
			"page_count": 18,
			"source": "scraper",
			"date_added": "2026-07-01 08:00:00.000000+00",
			"comment_count": 2
		},
		{
			"id": 888840,
			"chapter_number": 1,
			"chapter_title": null,
			"language": "us",
			"group_id": 18210,
			"group_name": "K Manga",
			"page_count": 21,
			"source": "user",
			"date_added": "2026-06-10 12:00:00.000000+00",
			"comment_count": 1
		}
	]`))
	defer server.Close()

	chapters, err := ListChapters(server.URL, "101", "18210", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	expectedPath := "/api/manga/101/chapters/list"

	if gotPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, gotPath)
	}

	if gotQuery != "group_id=18210" {
		t.Errorf("unexpected query: %q", gotQuery)
	}

	if len(chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(chapters))
	}

	// The API sends rows newest-first; the output is ascending.
	numbers := []string{
		chapters[0].Number.String(),
		chapters[1].Number.String(),
		chapters[2].Number.String(),
	}

	expectedNumbers := []string{"1", "2", "2.5"}

	for i, want := range expectedNumbers {
		if numbers[i] != want {
			t.Errorf(
				"chapter %d: expected number %q, got %q",
				i,
				want,
				numbers[i],
			)
		}
	}

	if chapters[0].Label != "Chapter 1" {
		t.Errorf(
			"expected fallback label, got %q",
			chapters[0].Label,
		)
	}

	expectedUserURL := server.URL +
		"/chapter/888840?source=user"

	if chapters[0].URL != expectedUserURL {
		t.Errorf(
			"expected URL %q, got %q",
			expectedUserURL,
			chapters[0].URL,
		)
	}

	expectedScraperURL := server.URL + "/chapter/889999"

	if chapters[1].URL != expectedScraperURL {
		t.Errorf(
			"expected URL %q, got %q",
			expectedScraperURL,
			chapters[1].URL,
		)
	}
}

func TestListChaptersScanlatorFilter(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[]`))
	defer server.Close()

	_, err := ListChapters(
		server.URL,
		"101",
		"scanlator:Stick",
		nil,
	)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if gotQuery != "scanlator_name=Stick" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
}

func TestListChaptersNoFilter(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[]`))
	defer server.Close()

	_, err := ListChapters(server.URL, "101", "", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if gotQuery != "" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
}

func TestListChaptersDedupeNoFilter(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[
		{
			"id": 900001,
			"chapter_number": 1,
			"chapter_title": "Chapter 1",
			"language": "es",
			"page_count": 78,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 900002,
			"chapter_number": 1,
			"chapter_title": "Chapter 1",
			"language": "en",
			"page_count": 76,
			"source": "user",
			"date_added": "2026-10-13 16:17:51.726423+00"
		},
		{
			"id": 900003,
			"chapter_number": 2,
			"chapter_title": "Chapter 2",
			"language": "en",
			"page_count": 19,
			"source": "user",
			"date_added": "2026-10-14 10:00:00.000000+00"
		}
	]`))
	defer server.Close()

	chapters, err := ListChapters(server.URL, "101", "", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	// Chapter 1 exists twice (es and en uploads); the newest upload
	// is kept and the output is ascending.
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	if chapters[0].Number.String() != "1" {
		t.Errorf("unexpected first number: %q", chapters[0].Number.String())
	}

	// The newest upload (900002) is kept.
	expectedURL := server.URL + "/chapter/900002?source=user"

	if chapters[0].URL != expectedURL {
		t.Errorf(
			"expected URL %q, got %q",
			expectedURL,
			chapters[0].URL,
		)
	}

	if chapters[1].Number.String() != "2" {
		t.Errorf("unexpected second number: %q", chapters[1].Number.String())
	}
}

func TestListChaptersFilterKeepsDuplicates(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[
		{
			"id": 900001,
			"chapter_number": 1,
			"chapter_title": "Chapter 1",
			"language": "es",
			"page_count": 78,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 900002,
			"chapter_number": 1,
			"chapter_title": "Chapter 1",
			"language": "en",
			"page_count": 76,
			"source": "user",
			"date_added": "2026-10-13 16:17:51.726423+00"
		}
	]`))
	defer server.Close()

	// With a scan path selected, rows are kept as sent: the group is
	// authoritative.
	chapters, err := ListChapters(server.URL, "101", "18210", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	expectedURLs := []string{
		server.URL + "/chapter/900001?source=user",
		server.URL + "/chapter/900002?source=user",
	}

	for i, want := range expectedURLs {
		if chapters[i].URL != want {
			t.Errorf(
				"chapter %d: expected URL %q, got %q",
				i,
				want,
				chapters[i].URL,
			)
		}
	}
}

func TestListChaptersSkipsCorruptRows(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[
		{
			"id": "",
			"chapter_number": 1,
			"page_count": 10,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 880001,
			"chapter_number": null,
			"page_count": 10,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 880002,
			"chapter_number": "not-a-number",
			"page_count": 10,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 880003,
			"chapter_number": 3,
			"page_count": 0,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 880004,
			"chapter_number": 4,
			"page_count": 12,
			"source": "user",
			"date_added": "2026-07-16 22:26:36.396671+00"
		}
	]`))
	defer server.Close()

	chapters, err := ListChapters(server.URL, "101", "", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 1 {
		t.Fatalf(
			"expected 1 usable chapter, got %d",
			len(chapters),
		)
	}

	if chapters[0].Number.String() != "4" {
		t.Errorf(
			"unexpected chapter number: %q",
			chapters[0].Number.String(),
		)
	}
}

func TestListChaptersPaddedNumbers(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(t, &gotPath, &gotQuery, `[
		{
			"id": 880010,
			"chapter_number": "7.00",
			"chapter_title": null,
			"page_count": 10,
			"source": "scraper",
			"date_added": "2026-07-16 22:26:36.396671+00"
		},
		{
			"id": 880011,
			"chapter_number": "06.50",
			"chapter_title": null,
			"page_count": 10,
			"source": "scraper",
			"date_added": "2026-07-16 22:26:36.396671+00"
		}
	]`))
	defer server.Close()

	chapters, err := ListChapters(server.URL, "101", "", nil)
	if err != nil {
		t.Fatalf("ListChapters() error: %v", err)
	}

	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}

	if chapters[0].Number.String() != "6.5" {
		t.Errorf(
			"expected padded number parsed as 6.5, got %q",
			chapters[0].Number.String(),
		)
	}

	if chapters[0].Label != "Chapter 6.5" {
		t.Errorf("unexpected label: %q", chapters[0].Label)
	}

	if chapters[1].Number.String() != "7" {
		t.Errorf(
			"expected padded number parsed as 7, got %q",
			chapters[1].Number.String(),
		)
	}
}

func TestListChaptersInvalidScanPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			t.Error("no request expected for an invalid scan path")
		},
	))
	defer server.Close()

	for _, scanPath := range []string{
		"weird:thing",
		"scanlator:",
		"0",
		"-5",
		"1e3",
	} {
		_, err := ListChapters(server.URL, "101", scanPath, nil)
		if err == nil {
			t.Fatalf(
				"expected an error for scan path %q",
				scanPath,
			)
		}

		if !strings.Contains(
			err.Error(),
			"unexpected mangadot.net scan path",
		) {
			t.Errorf(
				"scan path %q: unexpected error: %v",
				scanPath,
				err,
			)
		}
	}
}

func TestListChaptersHTTPError(t *testing.T) {
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

	_, err := ListChapters(server.URL, "101", "18210", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 503")
	}
}

func TestListChaptersInvalidJSON(t *testing.T) {
	var gotPath, gotQuery string

	server := httptest.NewServer(chaptersHandler(
		t,
		&gotPath,
		&gotQuery,
		`{"error": "not a chapter list"}`,
	))
	defer server.Close()

	_, err := ListChapters(server.URL, "101", "18210", nil)
	if err == nil {
		t.Fatal("expected a decode error for non-array JSON")
	}
}
