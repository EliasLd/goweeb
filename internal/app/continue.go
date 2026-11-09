package app

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/metadata"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// outputFormat maps the output options to the ledger format of the
// artifact the run produces.
func outputFormat(opts Options) ledger.Format {
	switch {
	case opts.CBZ:
		return ledger.FormatCBZ
	case opts.EbookFriendly:
		return ledger.FormatFolder
	default:
		return ledger.FormatPDF
	}
}

// ledgerKind maps the provider entry kind to the ledger kind.
func ledgerKind(kind sourcetypes.ItemKind) ledger.Kind {
	if kind == sourcetypes.ItemVolume {
		return ledger.KindVolume
	}

	return ledger.KindChapter
}

// openLedger prepares the work's ledger. A corrupted manifest
// surfaces as a warning and the empty ledger downloads everything
// again, healing the file as chapters complete.
//
// The ledger is opened on every download run since the metadata
// phase: chapter records, work description and covers materialize
// for plain runs too, while the skip filtering stays opt-in via the
// continue flag.
func openLedger(
	opts Options,
	work sourcetypes.Work,
	log *logger.Logger,
) *ledger.Ledger {
	led, err := ledger.New(
		ledgerPath(opts.Layout, opts.ScanDir, work.Title),
		ledger.Identity{
			Series:   work.Title,
			Provider: opts.Source,
			ScanPath: opts.ScanPath,
			WorkURL:  opts.MangaURL,
			Kind:     ledgerKind(work.Kind),
		},
	)
	if err != nil {
		log.Error(
			"Ledger unreadable, downloading everything again: %v\n",
			err,
		)
	}

	return &led
}

// resolveWorkMetadata assembles the fallback ladder for a work and
// returns the description the run carries. The ladder result is
// cached in the manifest; failures degrade to empty metadata.
func resolveWorkMetadata(
	opts Options,
	provider sourcetypes.Provider,
	work sourcetypes.Work,
	led *ledger.Ledger,
	log *logger.Logger,
) sourcetypes.Metadata {
	service := newMetadataService(!opts.NoExternalCovers)

	return fetchWorkMetadata(
		service,
		metadata.Request{
			Provider:  provider,
			WorkTitle: work.Title,
			WorkURL:   opts.MangaURL,
			Log:       log,
		},
		led,
		coverBasePath(opts.Layout, opts.ScanDir, work.Title),
		log,
	)
}

// requestHeadersFor exposes the provider's image headers for the
// cover CDNs when the provider gates them; harmless on ungated
// hosts.
func requestHeadersFor(
	provider sourcetypes.Provider,
) map[string]string {
	return imageRequestHeaders(provider)
}

// skipDownloaded removes entries whose artifact of the output format
// is already usable on disk. Entries recorded but unusable (missing
// file, truncated archive) log their reason and download again.
func skipDownloaded(
	led *ledger.Ledger,
	format ledger.Format,
	entries []sourcetypes.Entry,
	log *logger.Logger,
) []sourcetypes.Entry {
	kept := make([]sourcetypes.Entry, 0, len(entries))

	for _, entry := range entries {
		err := led.FindArtifact(entry.Number, format)
		if err == nil {
			log.Info(
				"Skipping %s: already downloaded.\n",
				entry.Label,
			)

			continue
		}

		if !errors.Is(err, ledger.ErrNotRecorded) {
			log.Debug(
				"%s is recorded but %v, downloading again.\n",
				entry.Label,
				err,
			)
		}

		kept = append(kept, entry)
	}

	if skipped := len(entries) - len(kept); skipped > 0 {
		log.Info(
			"Skipped %d chapter(s) already downloaded.\n",
			skipped,
		)
	}

	return kept
}

// recordDownloaded merges one finished artifact into the ledger and
// saves it right away, so a run interrupted later still leaves the
// ledger describing only artifacts that were complete when saved.
func recordDownloaded(
	led *ledger.Ledger,
	entry sourcetypes.Entry,
	format ledger.Format,
	artifact string,
	log *logger.Logger,
) {
	if led == nil {
		return
	}

	led.Record(ledger.Chapter{
		Number: entry.Number,
		Label:  entry.Label,
		URL:    entry.URL,
		Format: format,
		File:   filepath.Base(artifact),
		DownloadedAt: time.Now().UTC().Format(
			time.RFC3339,
		),
	})

	if err := led.Save(); err != nil {
		log.Error(
			"Failed to update the ledger: %v\n",
			err,
		)
	}
}
