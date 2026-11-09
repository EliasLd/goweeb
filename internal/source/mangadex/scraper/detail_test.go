package mangadexscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

// tLogger silences the debug logging the scrapers emit.
func tLogger(t *testing.T) *logger.Logger {
	t.Helper()

	return logger.New(&bytes.Buffer{}, logger.LevelDebug)
}

// Fixture shapes come from real API payloads (probed 2026-11-07,
// description residue probed 2026-11-09), trimmed to the fields the
// code reads. The description carries the separator line the site
// mirrors from its sources. Any structural drift in these rows is a
// live-data risk worth pinning.
const detailFixture = `{
	"result": "ok",
	"data": {
		"id": "801513ba-a712-498c-8f57-cae55b38cc92",
		"type": "manga",
		"attributes": {
			"title": {"ja-ro": "Berserk", "en": "Berserk"},
			"description": {
				"bg": "...",
				"en": "Guts, known as the Black Swordsman...\n___\nWon the 6th Osamu Tezuka Cultural Prize Excellence Award in 2002."
			},
			"tags": [
				{"attributes": {"name": {"en": "Award Winning"}}},
				{"attributes": {"name": {"en": "Monsters"}}}
			],
			"year": 1989,
			"status": "ongoing",
			"originalLanguage": "ja",
			"chapterNumbersResetOnNewVolume": false
		},
		"relationships": [
			{"id": "a1", "type": "author", "attributes": {"name": "Miura Kentarou"}},
			{"id": "a2", "type": "author", "attributes": {"name": "Mori Kouji"}},
			{"id": "a3", "type": "author", "attributes": {"name": "Miura Kentarou"}},
			{"id": "b1", "type": "artist", "attributes": {"name": "Miura Kentarou"}},
			{"id": "b2", "type": "artist", "attributes": {"name": "Studio Gaga"}},
			{"id": "c1", "type": "cover_art", "attributes": {"fileName": "81e1c82d-6672-400c-8c58-4ff9bfb89031.jpg"}},
			{"id": "d1", "type": "manga"},
			{"id": "d2", "type": "manga"}
		]
	}
}`

func TestGetMangaDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, detailFixture)
		},
	))
	defer server.Close()

	detail, err := GetMangaDetail(
		server.URL+"/manga/801513ba",
		server.URL+"/manga/801513ba-a712-498c-8f57-cae55b38cc92",
		tLogger(t),
	)
	if err != nil {
		t.Fatalf("GetMangaDetail() error: %v", err)
	}

	if detail.Name != "Berserk" {
		t.Errorf("Name = %q, want Berserk", detail.Name)
	}

	metadata := detail.Metadata

	if metadata.Author != "Miura Kentarou, Mori Kouji" {
		t.Errorf(
			"Author = %q, want distinct authors joined",
			metadata.Author,
		)
	}

	// "Miura Kentarou" is already listed as an author; only the rest
	// of the artist set shows up as the artist.
	if metadata.Artist != "Studio Gaga" {
		t.Errorf(
			"Artist = %q, want the artist-minus-author set",
			metadata.Artist,
		)
	}

	wantCover := "https://uploads.mangadex.org/covers/801513ba-a712-498c-8f57-cae55b38cc92/81e1c82d-6672-400c-8c58-4ff9bfb89031.jpg"
	if metadata.CoverURL != wantCover {
		t.Errorf(
			"CoverURL = %q, want the upload CDN address",
			metadata.CoverURL,
		)
	}

	// The separator rule the source mirrors is not stored text.
	if metadata.Description != "Guts, known as the Black Swordsman...\nWon the 6th Osamu Tezuka Cultural Prize Excellence Award in 2002." {
		t.Errorf("Description = %q, want the en locale cleaned of the separator rule", metadata.Description)
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 localized tags", metadata.Tags)
	}

	if metadata.OriginalLanguage != "ja" {
		t.Errorf("OriginalLanguage = %q, want ja", metadata.OriginalLanguage)
	}

	if metadata.Year != 1989 {
		t.Errorf("Year = %d, want 1989", metadata.Year)
	}
}

func TestGetMangaDetailRequestsIncludes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query()["includes[]"]; len(got) == 0 {
				t.Error("expected includes[] parameters to be requested")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, detailFixture)
		},
	))
	defer server.Close()

	if _, err := GetMangaDetail(
		server.URL,
		server.URL+"/manga/801513ba-a712-498c-8f57-cae55b38cc92",
		tLogger(t),
	); err != nil {
		t.Fatalf("GetMangaDetail() error: %v", err)
	}
}

func TestGetMangaDetailWorkURLWithoutMangaID(t *testing.T) {
	if _, err := GetMangaDetail(
		"https://api.mangadex.org",
		"https://api.mangadex.org/author/something",
		tLogger(t),
	); err == nil {
		t.Error("expected an error for a URL without a manga id")
	}
}

func TestGetMangaDetailTitleFallsBackToID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"result":"ok","data":{
				"id": "801513ba",
				"type": "manga",
				"attributes": {},
				"relationships": []
			}}`)
		},
	))
	defer server.Close()

	detail, err := GetMangaDetail(
		server.URL,
		server.URL+"/manga/801513ba",
		tLogger(t),
	)
	if err != nil {
		t.Fatalf("GetMangaDetail() error: %v", err)
	}

	if detail.Name != "801513ba" {
		t.Errorf("Name = %q, want the manga id fallback", detail.Name)
	}

	if detail.Metadata.Known() {
		t.Error("expected empty metadata for a bare work")
	}
}

func TestSearchCoverURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("includes[]"); got != "cover_art" {
				t.Errorf("search includes[] = %q, want cover_art", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{
				"result": "ok",
				"data": [
					{
						"id": "801513ba",
						"type": "manga",
						"attributes": {"title": {"en": "Berserk"}},
						"relationships": [
							{"id": "c1", "type": "cover_art", "attributes": {"fileName": "81e1c82d-6672-400c-8c58-4ff9bfb89031.jpg"}}
						]
					},
					{
						"id": "2222222222222222222222222222222",
						"type": "manga",
						"attributes": {"title": {"en": "No Cover"}}
					},
					{
						"id": "3333333333333333333333333333333",
						"type": "manga",
						"attributes": {"title": {}}
					}
				]
			}`)
		},
	))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "berserk", tLogger(t))
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	if results[0].CoverURL == "" {
		t.Error("expected the cover to travel with the search result")
	}

	if results[1].CoverURL != "" {
		t.Errorf(
			"CoverURL = %q, want empty without a cover_art relationship",
			results[1].CoverURL,
		)
	}
}

func TestPlainDescriptionCleansTheResidue(t *testing.T) {
	cases := []struct {
		name        string
		description string
		want        string
	}{
		{
			name:        "separator rule inside the text",
			description: "Guts, known as the Black Swordsman...\n___\nWon the 6th Osamu Tezuka Cultural Prize.",
			want:        "Guts, known as the Black Swordsman...\nWon the 6th Osamu Tezuka Cultural Prize.",
		},
		{
			name:        "blank lines stay",
			description: "Paragraph one.\n\nParagraph two.",
			want:        "Paragraph one.\n\nParagraph two.",
		},
		{
			name:        "bold markers ride along",
			description: "**Bold** start, then plain text.",
			want:        "Bold start, then plain text.",
		},
		{
			name:        "decoration only",
			description: "___\n---\n***",
			want:        "",
		},
		{
			name:        "surrounding decoration trimmed",
			description: "\n___\nA clean text.\n\n\n",
			want:        "A clean text.",
		},
	}

	for _, testCase := range cases {
		if got := plainDescription(testCase.description); got != testCase.want {
			t.Errorf(
				"plainDescription(%s) = %q, want %q",
				testCase.name,
				got,
				testCase.want,
			)
		}
	}
}
