//go:build live

// Live provider health checks: every registered provider is walked
// through one real search → scan path → chapter → PDF download
// against its actual website. Gated twice, a build tag (kept out of
// the default `go test ./...`) and a GOWEEB_LIVE=1 opt-in, because
// these tests hit real third-party sites: they belong to the
// scheduled workflow (.github/workflows/providers.yml) and manual
// runs, never to the hermetic CI or pre-commit gates.
//
// Assertions stay generic on purpose: search results, scan paths
// and entries are all live data, so the signal checked is the
// pipeline contract ("ends with one real PDF file"), never site
// content shape. Per-provider queries live in liveQueries and are
// maintenance data; when a provider's first hit makes this check
// false-alarm, tune the query.
package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source"
)

func TestMain(m *testing.M) {
	if os.Getenv("GOWEEB_LIVE") != "1" {
		fmt.Fprintln(
			os.Stderr,
			"live provider checks are opt-in; set GOWEEB_LIVE=1",
		)

		os.Exit(0)
	}

	os.Exit(m.Run())
}

// Popular, stable search per provider. When a provider stops
// matching here, fix the query, not the assertions below.
var liveQueries = map[string]string{
	"animesama":   "bleach",
	"atsumoe":     "blue lock",
	"mangafreak":  "one piece",
	"weebcentral": "berserk",
	"mangadex":    "berserk",
	"mangadotnet": "solo leveling",
	"mangavyvy":   "one piece",
}

// Runs every registry provider under a subtest named after its ID,
// so "go test -run 'TestLiveProvider/<id>$'" selects a single
// provider (what the workflow matrix does). The registry drives
// iteration, so newly registered providers appear here without
// touching this file; they skip until added to liveQueries.
func TestLiveProvider(t *testing.T) {
	for _, provider := range source.AvailableProviders() {
		t.Run(provider.ID, func(t *testing.T) {
			query, ok := liveQueries[provider.ID]
			if !ok {
				t.Skipf("no query configured for %s", provider.ID)
			}

			providerHealthCheck(t, provider.ID, query)
		})
	}
}

// providerHealthCheck downloads the first chapter of the first
// search result of the first scan path, through the full PDF
// pipeline, and verifies the produced file.
func providerHealthCheck(
	t *testing.T,
	providerID string,
	query string,
) {
	t.Helper()

	log := logger.New(
		io.Discard,
		logger.LevelInfo,
	)

	provider, err := source.New(providerID, "")
	if err != nil {
		t.Fatalf("init provider: %v", err)
	}

	results, err := provider.Search(query, log)
	if err != nil {
		t.Fatalf("search %q: %v", query, err)
	}

	if len(results) == 0 {
		t.Fatalf("search %q returned no results", query)
	}

	workURL := results[0].URL

	scanPaths, err := provider.ListScanPaths(workURL, log)
	if err != nil {
		t.Fatalf("list scan paths: %v", err)
	}

	if len(scanPaths) == 0 {
		t.Fatal("no scan paths found")
	}

	scanPath := scanPaths[0].Value

	_, entries, err := provider.ListEntries(workURL, scanPath, log)
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no chapters found")
	}

	// One chapter: the first discovered entry, whatever its
	// number is on the live site.
	first := entries[0].Number

	opts := app.Options{
		Slug:     providerID,
		Source:   providerID,
		MangaURL: workURL,
		ScanPath: scanPath,
		Selection: app.RangeSelection{
			Ranges: []app.RangeSpec{
				{
					Mode:  app.RangeNormal,
					Start: first,
					End:   first,
				},
			},
		},
		ScanDir: app.ResolveOutputDir(t.TempDir()),

		// Zero values: PDF output, the default pipeline.
		EbookFriendly: false,
		CBZ:           false,
	}

	// RunWithWorkflow logs-and-returns on every failure with no
	// error value, so the outcome is asserted on disk.
	app.RunWithWorkflow(opts, log)

	pdfs, err := filepath.Glob(
		filepath.Join(opts.ScanDir, "*.pdf"),
	)
	if err != nil {
		t.Fatalf("list pdf outputs: %v", err)
	}

	if len(pdfs) != 1 {
		t.Fatalf(
			"got %d pdf file(s), want 1; RunWithWorkflow did not complete for %s",
			len(pdfs),
			providerID,
		)
	}

	assertRealPDF(t, pdfs[0])
}

// assertRealPDF checks the produced file is a real, non-empty PDF.
func assertRealPDF(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat pdf output: %v", err)
	}

	// A single real chapter of pages; the small floor only
	// rejects empty or truncated-at-zero outputs.
	minPDFSize := 20 * 1024

	if info.Size() < int64(minPDFSize) {
		t.Fatalf(
			"pdf output is %d bytes, want at least %d",
			info.Size(),
			minPDFSize,
		)
	}

	magic := make([]byte, 5)

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open pdf output: %v", err)
	}
	defer file.Close()

	if _, err := file.Read(magic); err != nil {
		t.Fatalf("read pdf output: %v", err)
	}

	if string(magic) != "%PDF-" {
		t.Fatalf(
			"pdf output magic = %q, want %%PDF-",
			magic,
		)
	}
}
