package metadata

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

func anilistLogger(t *testing.T) *logger.Logger {
	t.Helper()

	return logger.New(&bytes.Buffer{}, logger.LevelDebug)
}

// serveFixturePointingAt installs a test endpoint for the duration
// of one test: lookups hit the local payload instead of the real
// API, restored automatically at cleanup.
func serveFixturePointingAt(
	t *testing.T,
	body string,
	status int,
) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = fmt.Fprint(w, body)
		},
	))
	t.Cleanup(server.Close)

	previous := anilistDefaultEndpoint
	anilistDefaultEndpoint = server.URL

	t.Cleanup(func() { anilistDefaultEndpoint = previous })
}

func TestLookupAniListUniqueEnglishMatch(t *testing.T) {
	serveFixturePointingAt(t, `{"data":{"page":{"media":[
		{"title": {"romaji": "Sokkupchingu Complex", "english": "Childhood Friend Complex", "native": "소꿉친구 컴플렉스"},
		 "synonyms": ["Other reading"],
		 "coverImage": {"extraLarge": "https://s4.anilist.co/file/anilistcdn/media/manga/cover/large/bx97816.jpg"},
		 "description": "Real description<br><br>(Source: Official)",
		 "countryOfOrigin": "KR",
		 "startDate": {"year": 2014},
		 "genres": ["Comedy", "Romance"]}
	]}}}`, http.StatusOK)

	match, err := lookupAniList(
		"Childhood Friend Complex",
		anilistLogger(t),
	)
	if err != nil {
		t.Fatalf("lookupAniList() error: %v", err)
	}

	if want := "Childhood Friend Complex"; match.Title.English != want {
		t.Errorf("Title.English = %q, want %q", match.Title.English, want)
	}

	if match.coverImageURL() == "" {
		t.Error("expected a cover address on the match")
	}

	if match.startDateYear() != 2014 {
		t.Errorf("Year = %d, want 2014", match.startDateYear())
	}

	if got := match.CountryOfOrigin; got != "KR" {
		t.Errorf("CountryOfOrigin = %q, want KR", got)
	}
}

// Probed live (2026-11-07): searching "Childhood Friend Complex"
// surface-ranked "Otonari Complex" first, and its user-curated
// synonyms even contain the exact query. Two distinct works match →
// the lookup must refuse.
func TestLookupAniListAmbiguousTitlesRefuse(t *testing.T) {
	serveFixturePointingAt(t, `{"data":{"page":{"media":[
		{"title": {"romaji": "Otonari Complex", "english": "Otonari Complex", "native": "おとなりコンプレックス"},
		 "synonyms": ["Childhood Friend Complex"],
		 "coverImage": {"extraLarge": "https://s4.anilist.co/wrong.jpg"}},
		{"title": {"romaji": "Sokkupchingu Complex", "english": "Childhood Friend Complex"},
		 "coverImage": {"extraLarge": "https://s4.anilist.co/right.jpg"}}
	]}}}`, http.StatusOK)

	_, err := lookupAniList(
		"Childhood Friend Complex",
		anilistLogger(t),
	)
	if err == nil {
		t.Fatal("expected the ambiguous match to be refused")
	}

	if !strings.Contains(err.Error(), "refusing to guess the cover") {
		t.Errorf("error = %v, want the refusal reason", err)
	}
}

func TestLookupAniListUniqueSynonymMatch(t *testing.T) {
	serveFixturePointingAt(t, `{"data":{"page":{"media":[
		{"title": {"romaji": "Hitoribocchi no Kusuriyubi", "english": "Solitary Finger"},
		 "synonyms": ["Berserk of the Lonely"],
		 "coverImage": {"extraLarge": "https://s4.anilist.co/cover.jpg"}}
	]}}}`, http.StatusOK)

	match, err := lookupAniList(
		"Berserk  of the Lonely!",
		anilistLogger(t),
	)
	if err != nil {
		t.Fatalf("lookupAniList() error: %v", err)
	}

	if match.coverImageURL() != "https://s4.anilist.co/cover.jpg" {
		t.Errorf(
			"cover = %q, want the synonym-matched work",
			match.coverImageURL(),
		)
	}
}

func TestLookupAniListNoMatch(t *testing.T) {
	serveFixturePointingAt(t, `{"data":{"page":{"media":[
		{"title": {"english": "Unrelated Work"}}
	]}}}`, http.StatusOK)

	_, err := lookupAniList("Totally Different Manga", anilistLogger(t))
	if err == nil || !strings.Contains(err.Error(), "no exact title match") {
		t.Fatalf("error = %v, want the no-match reason", err)
	}
}

func TestLookupAniListAPIFailureDegrades(t *testing.T) {
	serveFixturePointingAt(
		t,
		`{"errors":[{"message":"Internal error"}]}`,
		http.StatusInternalServerError,
	)

	_, err := lookupAniList("Berserk", anilistLogger(t))
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("error = %v, want the API status reason", err)
	}
}

func TestLookupAniListQueryErrorsReported(t *testing.T) {
	serveFixturePointingAt(t, `{
		"errors": [{"message": "Cannot query field"}],
		"data": null
	}`, http.StatusOK)

	_, err := lookupAniList("Berserk", anilistLogger(t))
	if err == nil || !strings.Contains(err.Error(), "rejected the query") {
		t.Fatalf("error = %v, want the rejected-query reason", err)
	}
}
