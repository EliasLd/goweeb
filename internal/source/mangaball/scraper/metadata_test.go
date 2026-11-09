package mangaballscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

// Fixture shapes come from real API payloads (probed 2026-11-07),
// trimmed to what the code reads. The author rows intentionally
// include the duplicates the API actually sends, and the description
// carries the markdown residue the site stores (bold markers,
// separator rules).
const richDetailFixture = `{
	"code": 200,
	"status": "success",
	"data": {
		"id": "68516a86702284f83417c943",
		"name": "Childhood Friend Complex",
		"slug": "childhood-friend-complex",
		"alternateName": ["Sokkupchingu Complex"],
		"image": {
			"cdn_mangadex": "https://uploads.mangadex.org/covers/eb211cad/c8505a45.jpg",
			"cdn_mangaupdate": null,
			"file": null,
			"cover": {"name": "cover_1751259607070.webp", "path": "68516a86702284f83417c943/cover_1751259607070.webp"}
		},
		"originalLanguage": "ko",
		"status": "ongoing",
		"date_published": 2024,
		"description": ["**Ha-neul and Mincheol** have been childhood friends for all of their lives.", "___", "Although they vowed never to rom"],
		"tags": [
			{"id": "1", "name": "Long Strip", "slug": "long-strip"},
			{"id": "2", "name": "Romance", "slug": "romance"}
		],
		"author": [
			{"id": "a1", "name": "Eunhi", "slug": "eunhi"},
			{"id": "a1", "name": "Eunhi", "slug": "eunhi"},
			{"id": "a2", "name": "MHL", "slug": "mhl"}
		]
	}
}`

func mbLogger(t *testing.T) *logger.Logger {
	t.Helper()

	return logger.New(&bytes.Buffer{}, logger.LevelDebug)
}

func detailServer(t *testing.T, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, body)
		},
	))
	t.Cleanup(server.Close)

	return server
}

func TestGetTitleDetailRich(t *testing.T) {
	server := detailServer(t, richDetailFixture)

	detail, err := GetTitleDetail(
		server.URL,
		"68516a86702284f83417c943",
		mbLogger(t),
	)
	if err != nil {
		t.Fatalf("GetTitleDetail() error: %v", err)
	}

	if detail.Name != "Childhood Friend Complex" {
		t.Errorf("Name = %q, want the title", detail.Name)
	}

	metadata := detail.Metadata

	if author := metadata.Author; author != "Eunhi, MHL" {
		t.Errorf("Author = %q, want the distinct author rows", author)
	}

	// The site-hosted cover path has no reachable host, so the
	// mangadex mirror wins.
	wantCover := "https://uploads.mangadex.org/covers/eb211cad/c8505a45.jpg"
	if metadata.CoverURL != wantCover {
		t.Errorf("CoverURL = %q, want the mangadex mirror", metadata.CoverURL)
	}

	wantDescription := "Ha-neul and Mincheol have been childhood friends for all of their lives.\nAlthough they vowed never to rom"
	if metadata.Description != wantDescription {
		t.Errorf("Description = %q, want the joined, cleaned paragraphs", metadata.Description)
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 names", metadata.Tags)
	}

	if metadata.OriginalLanguage != "ko" {
		t.Errorf("OriginalLanguage = %q, want ko", metadata.OriginalLanguage)
	}

	if metadata.Year != 2024 {
		t.Errorf("Year = %d, want 2024", metadata.Year)
	}
}

func TestGetTitleDetailCoverFallsBackToMangaupdatesMirror(t *testing.T) {
	server := detailServer(t, `{"code":200,"status":"success","data":{
		"name": "Some Title",
		"image": {
			"cdn_mangadex": null,
			"cdn_mangaupdate": "https://cdn.mangaupdates.com/image/i521271.jpg"
		}
	}}`)

	detail, err := GetTitleDetail(
		server.URL,
		"68516a86702284f83417c943",
		mbLogger(t),
	)
	if err != nil {
		t.Fatalf("GetTitleDetail() error: %v", err)
	}

	want := "https://cdn.mangaupdates.com/image/i521271.jpg"
	if detail.Metadata.CoverURL != want {
		t.Errorf("CoverURL = %q, want the mangaupdates mirror", detail.Metadata.CoverURL)
	}
}

func TestGetTitleDetailToleratesSparseAndOddShapes(t *testing.T) {
	// A minimal detail like the one the name tests use, plus shapes
	// that could appear: a single description string, a string year,
	// and junk description rows. None of these may fail a decode —
	// the listing flow rides the same endpoint.
	server := detailServer(t, `{"code":200,"status":"success","data":{
		"name": "Sparse Title",
		"image": null,
		"description": "One paragraph only",
		"date_published": "2024-05",
		"tags": [{"name": ""}],
		"author": "unexpected"
	}}`)

	detail, err := GetTitleDetail(
		server.URL,
		"68516a86702284f83417c943",
		mbLogger(t),
	)
	if err != nil {
		t.Fatalf("GetTitleDetail() error: %v", err)
	}

	if detail.Name != "Sparse Title" {
		t.Errorf("Name = %q, want the title", detail.Name)
	}

	metadata := detail.Metadata

	if metadata.Description != "One paragraph only" {
		t.Errorf("Description = %q, want the single paragraph", metadata.Description)
	}

	if metadata.Year != 0 {
		t.Errorf("Year = %d, want 0 for a non-integer publication date", metadata.Year)
	}

	if len(metadata.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", metadata.Tags)
	}

	if metadata.Author != "" {
		t.Errorf("Author = %q, want empty for an unexpected shape", metadata.Author)
	}
}

// The listing flow relies on this error message shape.
func TestGetTitleDetailKeepsNameError(t *testing.T) {
	server := detailServer(t, `{"code":200,"status":"success","data":{"name": ""}}`)

	_, err := GetTitleDetail(
		server.URL,
		"68516a86702284f83417c943",
		mbLogger(t),
	)

	if err == nil || !strings.Contains(err.Error(), "has no name") {
		t.Fatalf("error = %v, want the unnamed-title failure", err)
	}
}

func TestPlainDescriptionCleansTheSiteFormatting(t *testing.T) {
	input := []string{
		"**Childhood Friend Complex** est un manhwa de Eunhi.",
		"___",
		"- Un lycée, deux amis d'enfance, un secret.",
		"***",
		"",
		"Ha-neul et Mincheol partagent le même secret.",
		" --- ",
	}

	want := "Childhood Friend Complex est un manhwa de Eunhi.\n" +
		"- Un lycée, deux amis d'enfance, un secret.\n" +
		"Ha-neul et Mincheol partagent le même secret."

	if got := plainDescription(input); got != want {
		t.Errorf("plainDescription() = %q, want %q", got, want)
	}
}

func TestPlainDescriptionDegradesToEmpty(t *testing.T) {
	cases := []struct {
		name  string
		input []string
	}{
		{"nil", nil},
		{"blank rows", []string{"", "  "}},
		{"pure decoration", []string{"**", " *** ", "___", "---"}},
	}

	for _, testCase := range cases {
		if got := plainDescription(testCase.input); got != "" {
			t.Errorf(
				"plainDescription(%s) = %q, want empty (%s)",
				testCase.name,
				got,
				testCase.name,
			)
		}
	}
}

func TestPlainDescriptionPassesTextThrough(t *testing.T) {
	// Single-marker shapes and legitimate text keep every other
	// character: only the offered `**` pair and separator-only
	// lines are touched.
	input := []string{
		"Salut — ça va ?",
		"*italic stays*",
		"- a list marker with text survives",
	}

	want := "Salut — ça va ?\n" +
		"*italic stays*\n" +
		"- a list marker with text survives"

	if got := plainDescription(input); got != want {
		t.Errorf("plainDescription() = %q, want %q", got, want)
	}
}
