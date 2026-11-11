package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/convert"
	"github.com/EliasLd/goweeb/internal/fetch"
	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/metadata"
	"github.com/EliasLd/goweeb/internal/source"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func chapterDigits(maxChapter int) int {
	d := len(fmt.Sprintf("%d", maxChapter))

	if d < 3 {
		return 3
	}

	return d
}

// buildComicInfo assembles the CBZ metadata of one entry.
//
// Manga is always "Yes": goweeb downloads from manga providers and
// pages already carry their final reading order. Count is the number
// of entries available on the source, not just the downloaded ones.
// Work metadata (Phase 3) enriches the schema fields when a lookup
// produced something.
func buildComicInfo(
	work sourcetypes.Work,
	entry sourcetypes.Entry,
	totalEntries int,
	workMeta sourcetypes.Metadata,
	languageISO string,
) *convert.ComicInfo {
	info := convert.ComicInfo{
		Series: work.Title,
		Number: entry.Number.String(),
		Count:  totalEntries,
		Title:  entry.Label,
		Manga:  convert.MangaYes,
	}

	// The schema carries volumes as integers; a provider exposing
	// fractional numbers stays Number-only.
	if work.Kind == sourcetypes.ItemVolume &&
		entry.Number.IsInteger() {
		info.Volume, _ = strconv.Atoi(string(entry.Number))
	}

	if workMeta.Known() {
		info.Writer = workMeta.Author
		info.Penciller = workMeta.Artist
		info.Genre = strings.Join(workMeta.Tags, ", ")
		info.Description = workMeta.Description
		info.LanguageISO = languageISO
		info.Year = workMeta.Year
	}

	return &info
}

func toSelectableManga(
	in []sourcetypes.SearchResult,
) []common.SelectableItem {
	out := make(
		[]common.SelectableItem,
		0,
		len(in),
	)

	for _, r := range in {
		out = append(
			out,
			common.SelectableItem{
				Label: r.Title,
				Value: r.URL,
			},
		)
	}

	return out
}

func hasRangeSelection(
	selection RangeSelection,
) bool {
	return selection.All ||
		len(selection.Ranges) > 0
}

func logAvailableChapters(
	entries []sourcetypes.Entry,
	log *logger.Logger,
) {
	log.Info(
		"Found %d chapter(s).\n",
		len(entries),
	)

	log.Info(
		"Available chapters: %s\n",
		FormatAvailableRanges(entries),
	)
}

func applyRangeSelection(
	entries []sourcetypes.Entry,
	selection RangeSelection,
	log *logger.Logger,
) []sourcetypes.Entry {
	filtered := FilterEntriesBySelection(
		entries,
		selection,
	)

	if len(filtered) == 0 {
		return nil
	}

	if selection.All {
		log.Info(
			"Selected all %d available chapter(s).\n",
			len(filtered),
		)

		return filtered
	}

	log.Info(
		"Selected %d chapter(s): %s\n",
		len(filtered),
		FormatAvailableRanges(filtered),
	)

	return filtered
}

func imageRequestHeaders(
	provider sourcetypes.Provider,
) map[string]string {
	configurer, ok := provider.(sourcetypes.ImageRequestConfigurer)

	if !ok {
		return nil
	}

	return configurer.ImageRequestHeaders()
}

func downloadEntries(
	provider sourcetypes.Provider,
	work sourcetypes.Work,
	totalEntries int,
	entries []sourcetypes.Entry,
	ledgerFile *ledger.Ledger,
	opts Options,
	workMeta sourcetypes.Metadata,
	log *logger.Logger,
) {
	log.Info(
		"Downloading %d chapter(s)...\n",
		len(entries),
	)

	requestHeaders := imageRequestHeaders(
		provider,
	)

	languageCode := metadata.LanguageISO(
		opts.ScanPath,
	)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error(
			"Failed to get home dir: %v\n",
			err,
		)

		return
	}

	digits := 3

	for _, entry := range entries {
		digits = max(
			digits,
			entry.Number.IntegerDigits(),
		)
	}

	if opts.EbookFriendly || opts.CBZ {
		if err := os.MkdirAll(
			opts.ScanDir,
			os.ModePerm,
		); err != nil {
			log.Error(
				"Failed to create output dir: %v\n",
				err,
			)

			return
		}
	}

	for _, entry := range entries {
		chStr := entry.Number.String()

		log.Info(
			"Downloading %s...\n",
			entry.Label,
		)

		imageURLs, err := provider.GetPageImageURLs(
			entry.URL,
			log,
		)
		if err != nil {
			log.Error(
				"Failed to collect image URLs for %s: %v\n",
				entry.Label,
				err,
			)

			continue
		}

		if len(imageURLs) == 0 {
			log.Error(
				"No image URLs found for %s\n",
				entry.Label,
			)

			continue
		}

		if opts.EbookFriendly {
			prefix := "Chapter"

			if work.Kind == sourcetypes.ItemVolume {
				prefix = "Volume"
			}

			entryDir := layoutEntryImageDir(
				opts.Layout,
				opts.ScanDir,
				work.Title,
				prefix,
				entry.Number,
				digits,
			)

			if err := fetch.DownloadImagesWithHeaders(
				imageURLs,
				entryDir,
				requestHeaders,
				log,
			); err != nil {
				log.Error(
					"Failed to download %s: %v\n",
					entry.Label,
					err,
				)

				continue
			}

			log.Info(
				"%s saved to %s\n",
				entry.Label,
				entryDir,
			)

			recordDownloaded(
				ledgerFile,
				entry,
				ledger.FormatFolder,
				entryDir,
				log,
			)

			continue
		}

		// Handle CBZ output. Files are staged like PDFs: pages are
		// downloaded into a per-chapter staging folder, packed into
		// the archive, then the folder is removed.

		if opts.CBZ {
			imageDir := filepath.Join(
				homeDir,
				"Images",
				opts.Slug,
				chStr,
			)

			if err := fetch.DownloadImagesWithHeaders(
				imageURLs,
				imageDir,
				requestHeaders,
				log,
			); err != nil {
				log.Error(
					"Failed to download %s: %v\n",
					entry.Label,
					err,
				)

				continue
			}

			cbzPath := layoutEntryPath(
				opts.Layout,
				opts.ScanDir,
				work.Title,
				entry.Number,
				digits,
				".cbz",
			)

			// The archive may sit in a per-series subdirectory.
			if err := os.MkdirAll(
				filepath.Dir(cbzPath),
				os.ModePerm,
			); err != nil {
				log.Error(
					"Failed to create output dir: %v\n",
					err,
				)

				return
			}

			if err := convert.ImagesToCBZ(
				imageDir,
				cbzPath,
				buildComicInfo(
					work,
					entry,
					totalEntries,
					workMeta,
					languageCode,
				),
			); err != nil {
				log.Error(
					"Failed to create CBZ: %v\n",
					err,
				)

				continue
			}

			// The staged images live inside the archive now.
			_ = os.RemoveAll(imageDir)

			log.Info(
				"%s saved as %s\n",
				entry.Label,
				cbzPath,
			)

			recordDownloaded(
				ledgerFile,
				entry,
				ledger.FormatCBZ,
				cbzPath,
				log,
			)

			continue
		}

		imageDir := filepath.Join(
			homeDir,
			"Images",
			opts.Slug,
			chStr,
		)

		if err := fetch.DownloadImagesWithHeaders(
			imageURLs,
			imageDir,
			requestHeaders,
			log,
		); err != nil {
			log.Error(
				"Failed to download %s: %v\n",
				entry.Label,
				err,
			)

			continue
		}

		pdfPath := layoutEntryPath(
			opts.Layout,
			opts.ScanDir,
			work.Title,
			entry.Number,
			digits,
			".pdf",
		)

		// The archive may sit in a per-series subdirectory.
		if err := os.MkdirAll(
			filepath.Dir(pdfPath),
			os.ModePerm,
		); err != nil {
			log.Error(
				"Failed to create output dir: %v\n",
				err,
			)

			return
		}

		if err := convert.ImagesToPDF(
			imageDir,
			pdfPath,
			log,
		); err != nil {
			log.Error(
				"Failed to create PDF: %v\n",
				err,
			)

			continue
		}

		log.Info(
			"%s saved as %s\n",
			entry.Label,
			pdfPath,
		)

		recordDownloaded(
			ledgerFile,
			entry,
			ledger.FormatPDF,
			pdfPath,
			log,
		)
	}

	if !opts.EbookFriendly && !opts.CBZ {
		rootImagesDir := filepath.Join(
			homeDir,
			"Images",
			opts.Slug,
		)

		log.Debug(
			"Cleaning up: %s\n",
			rootImagesDir,
		)

		_ = os.RemoveAll(rootImagesDir)
	}
}

func Run(
	opts Options,
	log *logger.Logger,
) {
	provider, err := source.New(
		opts.Source,
		opts.CustomDomain,
	)
	if err != nil {
		log.Error(
			"Failed to initialize source provider: %v\n",
			err,
		)

		return
	}

	log.Info(
		"Searching for manga: %s\n",
		opts.Slug,
	)

	searchResults, err := provider.Search(
		opts.Slug,
		log,
	)
	if err != nil {
		log.Error(
			"Failed to search catalog: %v\n",
			err,
		)

		return
	}

	if len(searchResults) == 0 {
		log.Error(
			"Manga '%s' not found in catalog\n",
			opts.Slug,
		)

		return
	}

	if len(searchResults) == 1 {
		log.Info(
			"Found: %s\n",
			searchResults[0].Title,
		)
	} else {
		log.Info(
			"Found %d results.\n",
			len(searchResults),
		)
	}

	mangaURL, err := common.PromptUserToSelect(
		toSelectableManga(searchResults),
		"Multiple results found:",
		"Select a manga (1-%d) or 0 to cancel: ",
		log,
	)
	if err != nil {
		log.Error(
			"Failed to select manga: %v\n",
			err,
		)

		return
	}

	if mangaURL == "" {
		log.Warn(
			"No manga selected. Exiting.",
		)

		return
	}

	scanPaths, err := provider.ListScanPaths(
		mangaURL,
		log,
	)
	if err != nil {
		log.Error(
			"Failed to list scan paths: %v\n",
			err,
		)

		return
	}

	if len(scanPaths) == 0 {
		log.Warn(
			"No scan versions found.",
		)

		return
	}

	scanPath, err := common.PromptUserToSelect(
		scanPaths,
		"Multiple scan versions found:",
		"Select a scan version (1-%d) or 0 to cancel: ",
		log,
	)
	if err != nil {
		log.Error(
			"Failed to select scan path: %v\n",
			err,
		)

		return
	}

	if scanPath == "" {
		log.Warn(
			"No scan version selected. Exiting.",
		)

		return
	}

	// Both flow into the ledger identity and are already carried by
	// opts in the workflow mode.
	opts.MangaURL = mangaURL
	opts.ScanPath = scanPath

	// Resolve the run mode first: volume mode reads its own listing
	// only and never walks the chapter feed, so works numbered per
	// volume stay usable here. Explicit flags decide, and interactive
	// runs probe the volume listing once when the provider has the
	// capability. The chapter flow stays exactly as it is otherwise.
	if hasRangeSelection(opts.Selection) &&
		hasRangeSelection(opts.VolumeSelection) {
		log.Warn(
			"Chapter and volume selections cannot be used together.\n",
		)

		return
	}

	volumeLister, listsVolumes := provider.(sourcetypes.VolumeLister)

	var (
		volumes         []sourcetypes.Volume
		volumeWork      sourcetypes.Work
		volumeMode      bool
		promptedVolumes bool
	)

	if hasRangeSelection(opts.VolumeSelection) {
		if !listsVolumes {
			log.Warn(
				"The %s provider does not support volume downloads.\n",
				provider.Name(),
			)

			return
		}

		volumeWork, volumes, err = volumeLister.ListVolumes(
			mangaURL,
			scanPath,
			log,
		)
		if err != nil {
			log.Error(
				"Failed to list volumes: %v\n",
				err,
			)

			return
		}

		if len(volumes) == 0 {
			log.Warn(
				"This work has no volume data on this provider.\n",
			)

			return
		}

		volumeMode = true
	} else if !hasRangeSelection(opts.Selection) && listsVolumes {
		// Interactive runs may switch to volumes; a failing probe or
		// an untagged work keeps the chapter flow, exactly as today.
		w, v, probeErr := volumeLister.ListVolumes(
			mangaURL,
			scanPath,
			log,
		)

		if probeErr == nil && len(v) > 0 {
			volumeWork, volumes, volumeMode = w, v, true
		}
	}

	if volumeMode && !hasRangeSelection(opts.VolumeSelection) {
		// Volumes are available but not selected yet: ask which
		// listing to use; chapters keeps the historical flow.
		if !PromptChaptersOrVolumes(log) {
			volumeMode = false
		} else {
			promptedVolumes = true

			selection, promptErr := PromptVolumeRange(
				volumes,
				log,
			)
			if promptErr != nil {
				log.Error(
					"Failed to read volume range: %v\n",
					promptErr,
				)

				return
			}

			opts.VolumeSelection = selection
		}
	}

	if volumeMode {
		totalVolumes := len(volumes)

		if !promptedVolumes {
			log.Info(
				"Found %d volume(s).\n",
				totalVolumes,
			)

			log.Info(
				"Available volumes: %s\n",
				FormatVolumeRanges(volumes),
			)
		}

		selected := FilterVolumesBySelection(
			volumes,
			opts.VolumeSelection,
		)

		if len(selected) == 0 {
			log.Warn(
				"No available volumes match the requested selection.\n",
			)

			return
		}

		ledgerFile := openLedger(opts, volumeWork, log)

		// The description ladder runs before any download: it only
		// reads and caches, and a failure never stops the run.
		workMeta := resolveWorkMetadata(
			opts,
			provider,
			volumeWork,
			ledgerFile,
			log,
		)

		downloadVolumes(
			provider,
			volumeWork,
			totalVolumes,
			selected,
			ledgerFile,
			opts,
			workMeta,
			log,
		)

		return
	}

	work, entries, err := provider.ListEntries(
		mangaURL,
		scanPath,
		log,
	)
	if err != nil {
		log.Error(
			"Failed to list entries: %v\n",
			err,
		)

		return
	}

	if len(entries) == 0 {
		log.Warn(
			"No entries found.",
		)

		return
	}

	totalEntries := len(entries)

	ledgerFile := openLedger(opts, work, log)

	// The description ladder runs before any download: it only
	// reads and caches, and a failure never stops the run.
	workMeta := resolveWorkMetadata(
		opts,
		provider,
		work,
		ledgerFile,
		log,
	)

	if !hasRangeSelection(opts.Selection) {
		selection, err := PromptChapterRange(
			entries,
			log,
		)
		if err != nil {
			log.Error(
				"Failed to read chapter range: %v\n",
				err,
			)

			return
		}

		opts.Selection = selection
	} else {
		// When --range or --all was supplied on the CLI,
		// there is no interactive prompt, so still show the
		// user what chapters are actually available.
		logAvailableChapters(
			entries,
			log,
		)
	}

	entries = applyRangeSelection(
		entries,
		opts.Selection,
		log,
	)

	if len(entries) == 0 {
		log.Warn(
			"No available chapters match the requested selection.",
		)

		return
	}

	if ledgerFile != nil && opts.Continue {
		entries = skipDownloaded(
			ledgerFile,
			outputFormat(opts),
			entries,
			log,
		)

		if len(entries) == 0 {
			log.Info(
				"Nothing new to download.\n",
			)

			return
		}
	}

	downloadEntries(
		provider,
		work,
		totalEntries,
		entries,
		ledgerFile,
		opts,
		workMeta,
		log,
	)
}

func RunWithWorkflow(
	opts Options,
	log *logger.Logger,
) {
	provider, err := source.New(
		opts.Source,
		opts.CustomDomain,
	)
	if err != nil {
		log.Error(
			"Failed to initialize source provider: %v\n",
			err,
		)

		return
	}

	if hasRangeSelection(opts.Selection) &&
		hasRangeSelection(opts.VolumeSelection) {
		log.Warn(
			"Chapter and volume selections cannot be used together.\n",
		)

		return
	}

	// Volume mode reads its own listing only, mirroring the CLI flow
	// without its interactive prompts.
	if hasRangeSelection(opts.VolumeSelection) {
		volumeLister, listsVolumes := provider.(sourcetypes.VolumeLister)
		if !listsVolumes {
			log.Warn(
				"The %s provider does not support volume downloads.\n",
				provider.Name(),
			)

			return
		}

		work, volumes, err := volumeLister.ListVolumes(
			opts.MangaURL,
			opts.ScanPath,
			log,
		)
		if err != nil {
			log.Error(
				"Failed to list volumes: %v\n",
				err,
			)

			return
		}

		if len(volumes) == 0 {
			log.Warn(
				"This work has no volume data; nothing to download in volume mode.\n",
			)

			return
		}

		totalVolumes := len(volumes)

		selected := FilterVolumesBySelection(
			volumes,
			opts.VolumeSelection,
		)

		if len(selected) == 0 {
			log.Warn(
				"No available volumes match the requested selection.\n",
			)

			return
		}

		ledgerFile := openLedger(opts, work, log)

		workMeta := resolveWorkMetadata(
			opts,
			provider,
			work,
			ledgerFile,
			log,
		)

		log.Info(
			"Found %d volume(s).\n",
			totalVolumes,
		)

		log.Info(
			"Available volumes: %s\n",
			FormatVolumeRanges(volumes),
		)

		downloadVolumes(
			provider,
			work,
			totalVolumes,
			selected,
			ledgerFile,
			opts,
			workMeta,
			log,
		)

		log.Info(
			"Finished downloading\n",
		)

		return
	}

	if !hasRangeSelection(opts.Selection) {
		log.Warn(
			"Please enter a valid chapter selection.",
		)

		return
	}

	work, entries, err := provider.ListEntries(
		opts.MangaURL,
		opts.ScanPath,
		log,
	)
	if err != nil {
		log.Error(
			"Failed to list entries: %v\n",
			err,
		)

		return
	}

	if len(entries) == 0 {
		log.Warn(
			"No entries found.",
		)

		return
	}

	totalEntries := len(entries)

	ledgerFile := openLedger(opts, work, log)

	// The description ladder runs before any download: it only
	// reads and caches, and a failure never stops the run.
	workMeta := resolveWorkMetadata(
		opts,
		provider,
		work,
		ledgerFile,
		log,
	)

	logAvailableChapters(
		entries,
		log,
	)

	entries = applyRangeSelection(
		entries,
		opts.Selection,
		log,
	)

	if len(entries) == 0 {
		log.Warn(
			"No available chapters match the requested selection.",
		)

		return
	}

	if ledgerFile != nil && opts.Continue {
		entries = skipDownloaded(
			ledgerFile,
			outputFormat(opts),
			entries,
			log,
		)

		if len(entries) == 0 {
			log.Info(
				"Nothing new to download.\n",
			)

			return
		}
	}

	downloadEntries(
		provider,
		work,
		totalEntries,
		entries,
		ledgerFile,
		opts,
		workMeta,
		log,
	)

	log.Info(
		"Finished downloading\n",
	)
}
