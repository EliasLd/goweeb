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
//
// An edge that answers with a bot-protection challenge instead of
// the API (Cloudflare's managed "Just a moment…" page) says nothing
// about the provider: the challenge is unanswerable by design and
// keyed to the requester's IP reputation. Those subtests skip with
// a short reason (see edgeChallenge in edge_test.go) instead of
// failing the nightly red for something no HTTP client can answer.
package e2e

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/convert"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
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
	"mangaball":   "childhood friend complex",
	"mangafreak":  "one piece",
	"mangataro":   "berserk",
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
		liveFailure(t, fmt.Sprintf("search %q", query), err)
	}

	if len(results) == 0 {
		t.Fatalf("search %q returned no results", query)
	}

	workURL := results[0].URL

	scanPaths, err := provider.ListScanPaths(workURL, log)
	if err != nil {
		liveFailure(t, "list scan paths", err)
	}

	if len(scanPaths) == 0 {
		t.Fatal("no scan paths found")
	}

	scanPath := scanPaths[0].Value

	_, entries, err := provider.ListEntries(workURL, scanPath, log)
	if err != nil {
		liveFailure(t, "list entries", err)
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

	pdfs, err := collectOutputs(opts.ScanDir, ".pdf")
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

	// Volume mode rides the same live flow, gated by the provider's
	// own capability. The probe runs in its own subtest: its skip
	// for providers without volume data must stay local, or the
	// whole check of a chapter-only provider would report as
	// skipped after the PDF pipeline above already answered.
	t.Run("volumes", func(t *testing.T) {
		volumeHealthCheck(t, providerID, workURL, scanPaths)
	})
}

// collectOutputs walks the output tree recursively. The series
// layout nests outputs under a per-work folder and the naming
// belongs to the application, so this check stays layout-agnostic
// and only asserts the pipeline contract for one extension.
func collectOutputs(root string, ext string) ([]string, error) {
	var outputs []string

	err := filepath.WalkDir(
		root,
		func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() ||
				!strings.HasSuffix(entry.Name(), ext) {
				return nil
			}

			outputs = append(outputs, path)

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}

	return outputs, nil
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

// volumeHealthCheck bundles one volume the provider lists itself
// through the CBZ pipeline and verifies the produced archive. It
// skips for providers without the capability, works without volume
// data, and — like everything here — when the edge answers instead
// of the site. The smallest listed volume keeps the footprint polite
// on a live site; its number still comes from the listing.
func volumeHealthCheck(
	t *testing.T,
	providerID string,
	workURL string,
	scanPaths []common.SelectableItem,
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

	lister, listsVolumes := provider.(sourcetypes.VolumeLister)
	if !listsVolumes {
		t.Skipf("%s does not list volumes", providerID)
	}

	var (
		volumes  []sourcetypes.Volume
		scanPath string
	)

	for _, path := range scanPaths {
		_, listed, listErr := lister.ListVolumes(
			workURL,
			path.Value,
			log,
		)
		if listErr != nil {
			if edgeChallenge(listErr) {
				t.Skipf(
					"list volumes: %v",
					shortEdgeMessage(listErr),
				)
			}

			continue
		}

		if len(listed) > 0 {
			volumes = listed
			scanPath = path.Value

			break
		}
	}

	if len(volumes) == 0 {
		t.Skipf("no volume data for %q", workURL)
	}

	smallest := volumes[0]
	for _, volume := range volumes[1:] {
		if len(volume.Entries) < len(smallest.Entries) {
			smallest = volume
		}
	}

	number, err := common.ParseChapterNumber(smallest.Number)
	if err != nil {
		t.Fatalf("volume number %q: %v", smallest.Number, err)
	}

	opts := app.Options{
		Slug:     providerID,
		Source:   providerID,
		MangaURL: workURL,
		ScanPath: scanPath,
		VolumeSelection: app.RangeSelection{
			Ranges: []app.RangeSpec{
				{
					Mode:  app.RangeNormal,
					Start: number,
					End:   number,
				},
			},
		},
		ScanDir: app.ResolveOutputDir(t.TempDir()),
		CBZ:     true,
	}

	// RunWithWorkflow logs-and-returns on every failure with no
	// error value, so the outcome is asserted on disk.
	app.RunWithWorkflow(opts, log)

	cbzFiles, err := collectOutputs(opts.ScanDir, ".cbz")
	if err != nil {
		t.Fatalf("list cbz outputs: %v", err)
	}

	if len(cbzFiles) != 1 {
		t.Fatalf(
			"got %d cbz file(s), want 1; RunWithWorkflow did not bundle %s for %s",
			len(cbzFiles),
			volumeLabel(smallest.Number),
			providerID,
		)
	}

	assertRealVolumeCBZ(t, cbzFiles[0], smallest.Number)
}

// volumeLabel names a volume the log line reads, without the app's
// padding rules.
func volumeLabel(number string) string {
	return "vol. " + number
}

// assertRealVolumeCBZ opens the produced archive and verifies it is
// a real volume bundle: it opens, its metadata names the volume, and
// real pages travel inside.
func assertRealVolumeCBZ(
	t *testing.T,
	path string,
	wantNumber string,
) {
	t.Helper()

	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open cbz output %s: %v", path, err)
	}
	defer archive.Close()

	if len(archive.File) < 2 {
		t.Fatalf(
			"cbz output carries %d entries, want at least the metadata and one page",
			len(archive.File),
		)
	}

	var comicInfo convert.ComicInfo

	for _, entry := range archive.File {
		if entry.Name != "ComicInfo.xml" {
			continue
		}

		payload, err := readZipEntry(entry)
		if err != nil {
			t.Fatalf("read ComicInfo.xml: %v", err)
		}

		if err := xml.Unmarshal(payload, &comicInfo); err != nil {
			t.Fatalf("parse ComicInfo.xml: %v", err)
		}

		break
	}

	want, err := strconv.Atoi(wantNumber)
	if err != nil {
		t.Fatalf("volume number %q: %v", wantNumber, err)
	}

	if comicInfo.Volume != want {
		t.Errorf(
			"ComicInfo Volume = %d, want %d",
			comicInfo.Volume,
			want,
		)
	}

	if comicInfo.PageCount <= 0 {
		t.Errorf(
			"ComicInfo PageCount = %d, want the archived pages",
			comicInfo.PageCount,
		)
	}
}

func readZipEntry(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// liveFailure reports a failed live call: when the edge blocked the
// request itself the subtest skips — from that answer nothing can be
// concluded about the provider — and any other failure fails.
func liveFailure(t *testing.T, context string, err error) {
	t.Helper()

	if edgeChallenge(err) {
		t.Skipf("%s: %v", context, shortEdgeMessage(err))
	}

	t.Fatalf("%s: %v", context, err)
}
