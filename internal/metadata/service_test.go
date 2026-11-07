package metadata

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// stubProvider implements the provider contract enough for the
// ladder tests: a name plus, optionally, rows returned by Search and
// a metadata payload (present only via the optional interface).
type stubProvider struct {
	name      string
	rows      []sourcetypes.SearchResult
	searchErr error
	metadata  sourcetypes.Metadata
	metaErr   error
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Search(
	_ string,
	_ *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	if s.searchErr != nil {
		return nil, s.searchErr
	}

	return s.rows, nil
}

func (s *stubProvider) ListScanPaths(
	_ string,
	_ *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, nil
}

func (s *stubProvider) ListEntries(
	_ string,
	_ string,
	_ *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{Title: "stub"}, nil, nil
}

func (s *stubProvider) GetPageImageURLs(
	_ string,
	_ *logger.Logger,
) ([]string, error) {
	return nil, nil
}

// WorkMetadata is the optional capability; the service's type switch
// only sees it when the concrete provider implements it.
func (s *stubProvider) WorkMetadata(
	_ string,
	_ *logger.Logger,
) (sourcetypes.Metadata, error) {
	return s.metadata, s.metaErr
}

func TestFetchProviderLayerOnly(t *testing.T) {
	primary := &stubProvider{
		name: "richprovider",
		metadata: sourcetypes.Metadata{
			Author:   "Miura Kentarou",
			CoverURL: "https://example.com/cover.jpg",
			Year:     1989,
		},
	}

	service := Service{}

	req := testRequestLog(t)
	req.Provider = primary

	resolved, err := service.Fetch(req)
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if resolved.Author != "Miura Kentarou" {
		t.Errorf("Author = %q, want the provider's author", resolved.Author)
	}

	if resolved.CoverSource != "richprovider" {
		t.Errorf(
			"CoverSource = %q, want the provider's own stamp",
			resolved.CoverSource,
		)
	}
}

// plainProvider is a provider that does NOT implement the optional
// metadata capability — a distinct type, since method sets are
// static.
type plainProvider struct {
	name string
}

func (s *plainProvider) Name() string { return s.name }

func (s *plainProvider) Search(
	_ string,
	_ *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	return nil, fmt.Errorf("no rows")
}

func (s *plainProvider) ListScanPaths(
	_ string,
	_ *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, nil
}

func (s *plainProvider) ListEntries(
	_ string,
	_ string,
	_ *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{Title: "plain"}, nil, nil
}

func (s *plainProvider) GetPageImageURLs(
	_ string,
	_ *logger.Logger,
) ([]string, error) {
	return nil, nil
}

func TestFetchProviderWithoutCapability(t *testing.T) {
	root := &plainProvider{name: "plainprovider"}

	service := Service{}

	req := testRequestLog(t)
	req.Provider = root

	_, err := service.Fetch(req)
	if err == nil {
		t.Fatal("expected a no-metadata error without any capable layer")
	}

	if !strings.Contains(err.Error(), "does not expose work metadata") {
		t.Errorf("error = %s, want the capability reason", err.Error())
	}
}

func TestFetchCrossCoverByExactTitle(t *testing.T) {
	primary := &stubProvider{name: "plainprovider"}

	cross := &stubProvider{
		name: "crossprovider",
		rows: []sourcetypes.SearchResult{
			{Title: "Berserk: Volume 1"},
			{
				Title:    "Berserk",
				CoverURL: "https://example.com/right-cover.jpg",
			},
		},
	}

	service := Service{CrossProviders: []sourcetypes.Provider{cross}}

	req := testRequestLog(t)
	req.Provider = primary

	resolved, err := service.Fetch(req)
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if resolved.CoverURL != "https://example.com/right-cover.jpg" {
		t.Errorf(
			"CoverURL = %q, want the cross-provider cover",
			resolved.CoverURL,
		)
	}

	if resolved.CoverSource != "cross:crossprovider" {
		t.Errorf(
			"CoverSource = %q, want the cross stamp",
			resolved.CoverSource,
		)
	}

	if resolved.Author != "" {
		t.Error("expected no text fields from the cross layer")
	}
}

func TestFetchCrossRefusesFuzzyTitles(t *testing.T) {
	primary := &stubProvider{name: "plainprovider"}

	cross := &stubProvider{
		name: "crossprovider",
		rows: []sourcetypes.SearchResult{
			{Title: "Berserk of Gluttony"},
			{Title: "Otonari Complex"},
		},
	}

	service := Service{CrossProviders: []sourcetypes.Provider{cross}}

	req := testRequestLog(t)
	req.Provider = primary

	_, err := service.Fetch(req)
	if err == nil {
		t.Fatal("expected no metadata: no cross row matches exactly")
	}
}

func TestFetchCrossSearchFailureIsNotFatal(t *testing.T) {
	primary := &stubProvider{name: "plainprovider"}

	cross := &stubProvider{
		name:      "crossprovider",
		searchErr: fmt.Errorf("site unavailable"),
	}

	service := Service{CrossProviders: []sourcetypes.Provider{cross}}

	req := testRequestLog(t)
	req.Provider = primary

	_, err := service.Fetch(req)
	if err == nil {
		t.Fatal("expected the empty-metadata error with the layer reason")
	}

	if !strings.Contains(err.Error(), "site unavailable") {
		t.Errorf("error = %s, want the layer reason included", err.Error())
	}
}

func TestFetchExternalFillsTextWhenNoProviderData(t *testing.T) {
	primary := &stubProvider{name: "plainprovider"}

	service := Service{IncludeExternal: true}

	req := testRequestLog(t)
	req.Provider = primary
	req.WorkTitle = "Childhood Friend Complex"

	serveExternalMatch(t)

	resolved, fetchErr := service.Fetch(req)
	if fetchErr != nil {
		t.Fatalf("Fetch() error: %v", fetchErr)
	}

	// The external layer fills cover and text wholesale: the work
	// had nothing.
	if resolved.CoverURL != "https://s4.anilist.co/cover-real.jpg" ||
		resolved.CoverSource != anilistSource {
		t.Errorf(
			"cover = %q (%q), want the external cover",
			resolved.CoverURL,
			resolved.CoverSource,
		)
	}

	if resolved.Author != "Eunhi" {
		t.Errorf("Author = %q, want the external staff", resolved.Author)
	}

	if !strings.Contains(resolved.Description, "Ha-neul and Mincheol") {
		t.Errorf("Description = %q, want the sanitized external text", resolved.Description)
	}

	if resolved.Year != 2024 {
		t.Errorf("Year = %d, want 2024", resolved.Year)
	}
}

func TestFetchExternalKeepsProviderTextAndFillsCoverOnly(t *testing.T) {
	primary := &stubProvider{
		name: "richprovider",
		metadata: sourcetypes.Metadata{
			Author:      "Miura Kentarou",
			Description: "provider-provided",
		},
	}

	service := Service{IncludeExternal: true}

	req := testRequestLog(t)
	req.Provider = primary
	req.WorkTitle = "Childhood Friend Complex"

	serveExternalMatch(t)

	resolved, fetchErr := service.Fetch(req)
	if fetchErr != nil {
		t.Fatalf("Fetch() error: %v", fetchErr)
	}

	if resolved.CoverURL == "" || resolved.CoverSource != anilistSource {
		t.Error("expected the external cover to fill in")
	}

	if resolved.Author != "Miura Kentarou" ||
		resolved.Description != "provider-provided" {
		t.Errorf(
			"text = %q/%q, want the provider data kept",
			resolved.Author,
			resolved.Description,
		)
	}
}

func TestFetchExternalNeverMixesWhenProviderCarriesText(t *testing.T) {
	primary := &stubProvider{
		name: "richprovider",
		metadata: sourcetypes.Metadata{
			CoverURL: "https://example.com/own.jpg",
			Author:   "Miura Kentarou",
		},
	}

	service := Service{IncludeExternal: true}

	req := testRequestLog(t)
	req.Provider = primary
	req.WorkTitle = "Childhood Friend Complex"

	serveExternalMatch(t)

	resolved, fetchErr := service.Fetch(req)
	if fetchErr != nil {
		t.Fatalf("Fetch() error: %v", fetchErr)
	}

	if resolved.CoverURL != "https://example.com/own.jpg" ||
		resolved.CoverSource != "richprovider" {
		t.Error("expected the provider's own cover to stay")
	}

	if resolved.Year != 0 {
		t.Errorf("Year = %d, want the provider data kept alone", resolved.Year)
	}
}

func testRequestLog(t *testing.T) Request {
	t.Helper()

	return Request{
		WorkTitle: "Berserk",
		WorkURL:   "https://example.com/manga/1",
		Log:       logger.New(&bytes.Buffer{}, logger.LevelInfo),
	}
}

// serveExternalMatch installs the AniList fixture with the real
// probed match for "Childhood Friend Complex" (its synonym noise on
// other rows never matches this query).
func serveExternalMatch(t *testing.T) {
	t.Helper()

	serveFixturePointingAt(t, `{"data":{"page":{"media":[
		{
			"title": {"romaji": "Sokkupchingu Complex", "english": "Childhood Friend Complex", "native": "소꿉친구 컴플렉스"},
			"synonyms": ["Other reading"],
			"coverImage": {"large": "https://s4.anilist.co/cover-real.jpg"},
			"description": "Ha-neul and Mincheol have been childhood friends for all of their lives.<br><br>(Source: Staff)",
			"countryOfOrigin": "KR",
			"startDate": {"year": 2024},
			"genres": ["Comedy", "Romance"],
			"staff": {"edges": [{"role": "Story & Art", "node": {"name": {"full": "Eunhi"}}}]}
		},
		{
			"title": {"romaji": "Otonari Complex", "english": "Otonari Complex"},
			"synonyms": ["Neighborhood Complex"]
		}
	]}}}`, 200)
}
