package app

import (
	"os"
	"path/filepath"

	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/metadata"
	"github.com/EliasLd/goweeb/internal/source"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// fetchWorkMetadata resolves the work's description through the
// fallback ladder and caches the result into the manifest, cover
// file included.
//
// The lookup never blocks downloads on failure: every problem
// degrades to empty metadata, logged and left for a later run.
// Results are cached by the ledger — a work with recorded metadata
// (and its cover file present) performs no lookup calls at all.
func fetchWorkMetadata(
	service metadata.Service,
	req metadata.Request,
	led *ledger.Ledger,
	coverBase string,
	log *logger.Logger,
) sourcetypes.Metadata {
	if led != nil {
		cached := led.Metadata()
		if cached != nil {
			return cachedLookup(
				req,
				led,
				coverBase,
				*cached,
				log,
			)
		}
	}

	resolved, err := service.Fetch(req)
	if err != nil {
		// The layers already explained themselves at debug level.
		log.Debug("%v\n", err)

		return sourcetypes.Metadata{}
	}

	log.Info(
		"Work metadata attached (cover from %s).\n",
		resolved.CoverSource,
	)

	if resolved.CoverURL != "" {
		downloadAndRecordCover(
			resolved.CoverURL,
			coverBase,
			req,
			led,
			log,
		)
	}

	if led != nil && led.SetMetadata(resolved) {
		saveLedgerMetadata(led, log)
	}

	return resolved
}

// cachedLookup serves the recorded metadata, refreshing the cover
// file along its recorded URL when something deleted it from disk.
// No metadata endpoint is contacted on this path.
func cachedLookup(
	req metadata.Request,
	led *ledger.Ledger,
	coverBase string,
	cached sourcetypes.Metadata,
	log *logger.Logger,
) sourcetypes.Metadata {
	coverRecorded := led.CoverFile()
	present := coverFilePresent(coverBase, coverRecorded)

	if cached.CoverURL != "" && !present {
		if name, err := metadata.DownloadCover(
			cached.CoverURL,
			coverBase,
			requestHeadersFor(req.Provider),
			log,
		); err == nil {
			if led.SetCoverFile(name) {
				saveLedgerMetadata(led, log)
			}
		} else {
			// The cover is a cosmetic loss; the placeholder path
			// applies at render time.
			log.Debug("Cover re-download failed: %v\n", err)
		}
	}

	log.Debug(
		"Work metadata already recorded for %q; nothing to fetch\n",
		req.WorkTitle,
	)

	return cached
}

// downloadAndRecordCover fetches a cover into the manifest's
// directory and records its name. Failures degrade to no cover.
func downloadAndRecordCover(
	coverURL string,
	coverBase string,
	req metadata.Request,
	led *ledger.Ledger,
	log *logger.Logger,
) {
	// The cover lands next to the manifest, whose directory may not
	// exist yet (the first chapter may not have been written).
	if err := os.MkdirAll(filepath.Dir(coverBase), os.ModePerm); err != nil {
		log.Debug(
			"Failed to create the cover directory: %v\n",
			err,
		)

		return
	}

	name, err := metadata.DownloadCover(
		coverURL,
		coverBase,
		requestHeadersFor(req.Provider),
		log,
	)
	if err != nil {
		log.Debug("Cover download failed: %v\n", err)

		return
	}

	if led != nil && led.SetCoverFile(name) {
		saveLedgerMetadata(led, log)
	}
}

// saveLedgerMetadata persists the metadata block (and cover record).
func saveLedgerMetadata(
	led *ledger.Ledger,
	log *logger.Logger,
) {
	if err := led.Save(); err != nil {
		log.Error(
			"Failed to save the metadata into the ledger: %v\n",
			err,
		)
	}
}

// coverFilePresent reports whether the recorded cover file survived
// on disk.
func coverFilePresent(coverBase string, name string) bool {
	if name == "" {
		return false
	}

	path := filepath.Join(filepath.Dir(coverBase), name)

	info, err := os.Stat(path)

	return err == nil && !info.IsDir() && info.Size() > 0
}

// newMetadataService assembles the ladder: every other registered
// provider as a cross-cover candidate (default domains), and the
// external database layer according to the run's options.
func newMetadataService(
	primary string,
	includeExternal bool,
	log *logger.Logger,
) metadata.Service {
	cross := make(
		[]sourcetypes.Provider,
		0,
		len(source.AvailableProviders()),
	)

	for _, providerInfo := range source.AvailableProviders() {
		if providerInfo.ID == primary {
			continue
		}

		candidate, err := source.New(providerInfo.ID, "")
		if err != nil {
			// A registry entry that cannot be built never blocks
			// the ladder.
			log.Debug(
				"Cross-cover candidate %s unavailable: %v\n",
				providerInfo.ID,
				err,
			)

			continue
		}

		cross = append(cross, candidate)
	}

	return metadata.Service{
		CrossProviders:  cross,
		IncludeExternal: includeExternal,
	}
}
