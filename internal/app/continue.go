package app

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
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

// openLedger prepares the work's ledger when continuing was
// requested. A corrupted manifest surfaces as a warning and the
// empty ledger downloads everything again, healing the file as
// chapters complete.
func openLedger(
	opts Options,
	work sourcetypes.Work,
	log *logger.Logger,
) *ledger.Ledger {
	if !opts.Continue {
		return nil
	}

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
