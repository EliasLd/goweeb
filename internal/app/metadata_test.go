package app

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/metadata"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

type fakeMetadataProvider struct {
	name string

	lookups int

	payload sourcetypes.Metadata
	failErr error
}

func (f *fakeMetadataProvider) Name() string { return f.name }

func (f *fakeMetadataProvider) Search(
	_ string,
	_ *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	return nil, nil
}

func (f *fakeMetadataProvider) ListScanPaths(
	_ string,
	_ *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, nil
}

func (f *fakeMetadataProvider) ListEntries(
	_ string,
	_ string,
	_ *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{Title: f.name}, nil, nil
}

func (f *fakeMetadataProvider) GetPageImageURLs(
	_ string,
	_ *logger.Logger,
) ([]string, error) {
	return nil, nil
}

// WorkMetadata is the optional capability under test.
func (f *fakeMetadataProvider) WorkMetadata(
	_ string,
	_ *logger.Logger,
) (sourcetypes.Metadata, error) {
	f.lookups++

	return f.payload, f.failErr
}

// serveCover answers every request with a few JPEG magic bytes, the
// way a cover CDN does.
func serveCover(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte{
				0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F',
			})
		},
	))
	t.Cleanup(server.Close)

	return server
}

// testLogger keeps the flow's log lines out of the test output.
func testLogger() *logger.Logger {
	return logger.New(&bytes.Buffer{}, logger.LevelInfo)
}

// testLedger opens the manifest the way a run would.
func testLedger(
	t *testing.T,
	dir string,
) *ledger.Ledger {
	t.Helper()

	led, err := ledger.New(
		ledgerPath(LayoutSeries, dir, "One Piece"),
		ledger.Identity{
			Series:   "One Piece",
			Provider: "fakeprovider",
			WorkURL:  "https://example.com/manga/1",
			Kind:     ledger.KindChapter,
		},
	)
	if err != nil {
		t.Fatalf("ledger.New() error: %v", err)
	}

	return &led
}

func TestFetchWorkMetadataRecordsIntoLedger(t *testing.T) {
	dir := t.TempDir()

	provider := &fakeMetadataProvider{
		name: "fakeprovider",
		payload: sourcetypes.Metadata{
			Author: "Miura Kentarou",
		},
	}

	led := testLedger(t, dir)

	resolved := fetchWorkMetadata(
		metadata.Service{},
		metadata.Request{
			Provider:  provider,
			WorkTitle: "One Piece",
			Log:       testLogger(),
		},
		led,
		coverBasePath(LayoutSeries, dir, "One Piece"),
		testLogger(),
	)

	if resolved.Author != "Miura Kentarou" {
		t.Errorf("Author = %q, want the provider payload", resolved.Author)
	}

	if stored := led.Metadata(); stored == nil ||
		stored.Author != "Miura Kentarou" {
		t.Errorf(
			"ledger metadata = %+v, want the payload recorded",
			stored,
		)
	}
}

func TestFetchWorkMetadataSkipsLookupsOnCacheHit(t *testing.T) {
	dir := t.TempDir()

	provider := &fakeMetadataProvider{
		name: "fakeprovider",
		payload: sourcetypes.Metadata{
			Author: "Miura Kentarou",
		},
	}

	led := testLedger(t, dir)

	request := metadata.Request{
		Provider:  provider,
		WorkTitle: "One Piece",
		Log:       testLogger(),
	}

	coverBase := coverBasePath(LayoutSeries, dir, "One Piece")

	first := fetchWorkMetadata(
		metadata.Service{},
		request,
		led,
		coverBase,
		testLogger(),
	)

	if first.Author != "Miura Kentarou" {
		t.Fatalf("Author = %q, want the payload", first.Author)
	}

	provider.lookups = 0

	again := fetchWorkMetadata(
		metadata.Service{},
		request,
		led,
		coverBase,
		testLogger(),
	)

	if provider.lookups != 0 {
		t.Errorf(
			"lookups = %d, want 0 for a cached work",
			provider.lookups,
		)
	}

	if again.Author != first.Author {
		t.Errorf(
			"cached metadata diverged: %q != %q",
			again.Author,
			first.Author,
		)
	}
}

func TestFetchWorkMetadataDegradesOnFailure(t *testing.T) {
	dir := t.TempDir()

	provider := &fakeMetadataProvider{
		name:    "fakeprovider",
		failErr: fmt.Errorf("the site answered 503"),
	}

	led := testLedger(t, dir)

	resolved := fetchWorkMetadata(
		metadata.Service{},
		metadata.Request{
			Provider:  provider,
			WorkTitle: "One Piece",
			Log:       testLogger(),
		},
		led,
		coverBasePath(LayoutSeries, dir, "One Piece"),
		testLogger(),
	)

	if resolved.Known() {
		t.Errorf("metadata = %+v, want a clean degrade to empty", resolved)
	}

	if led.Metadata() != nil {
		t.Error("expected no metadata recorded on failure")
	}
}

func TestFetchWorkMetadataDownloadsCover(t *testing.T) {
	dir := t.TempDir()

	server := serveCover(t)

	provider := &fakeMetadataProvider{
		name: "fakeprovider",
		payload: sourcetypes.Metadata{
			Author:      "Kentarou",
			CoverURL:    server.URL + "/covers/81e1c82d.jpg",
			CoverSource: "fakeprovider",
		},
	}

	led := testLedger(t, dir)

	coverBase := coverBasePath(LayoutSeries, dir, "One Piece")

	resolved := fetchWorkMetadata(
		metadata.Service{},
		metadata.Request{
			Provider:  provider,
			WorkTitle: "One Piece",
			Log:       testLogger(),
		},
		led,
		coverBase,
		testLogger(),
	)

	// The mirror address is recorded untouched; the file is what
	// downstream consumers read.
	if resolved.CoverURL != server.URL+"/covers/81e1c82d.jpg" {
		t.Errorf(
			"CoverURL = %q, want the recorded address",
			resolved.CoverURL,
		)
	}

	if file := led.CoverFile(); file != "One Piece.cover.jpg" {
		t.Errorf("cover file = %q, want One Piece.cover.jpg", file)
	}

	if !coverFilePresent(coverBase, led.CoverFile()) {
		t.Error("expected the cover file on disk")
	}
}
