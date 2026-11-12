// Volume mode bundles a work's chapters into the volumes its source
// actually lists: one CBZ archive per volume, and per-chapter
// artifacts grouped under a volume directory for PDF and
// ebook-friendly output. Membership always comes from the provider's
// listing — volume mode never guesses.
package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/convert"
	"github.com/EliasLd/goweeb/internal/fetch"
	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/metadata"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// volumeDigits returns the zero-padding width of volume numbers: the
// widest number the listing carries, never under two digits.
func volumeDigits(volumes []sourcetypes.Volume) int {
	digits := 2

	for _, volume := range volumes {
		if parsed, err := strconv.Atoi(volume.Number); err == nil {
			digits = max(digits, len(strconv.Itoa(parsed)))
		}
	}

	return digits
}

// FilterVolumesBySelection keeps the volumes whose number matches
// the selection, in reading order. Volume numbers are integers on
// every implementer; one that cannot parse never matches.
func FilterVolumesBySelection(
	volumes []sourcetypes.Volume,
	selection RangeSelection,
) []sourcetypes.Volume {
	if volumes == nil {
		return nil
	}

	ordered := make([]sourcetypes.Volume, 0, len(volumes))
	volumeNumbers := make(map[string]common.ChapterNumber, len(volumes))

	for _, volume := range volumes {
		number, err := common.ParseChapterNumber(volume.Number)
		if err != nil {
			continue
		}

		volumeNumbers[volume.Number] = number
		ordered = append(ordered, volume)
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		return volumeNumbers[ordered[i].Number].
			Compare(volumeNumbers[ordered[j].Number]) < 0
	})

	if selection.All {
		return ordered
	}

	if len(selection.Ranges) == 0 {
		return nil
	}

	if selection.Ranges[0].Mode == RangeLastN {
		count := selection.Ranges[0].Count
		if count > len(ordered) {
			count = len(ordered)
		}

		return ordered[len(ordered)-count:]
	}

	selected := make([]sourcetypes.Volume, 0, len(ordered))

	for _, volume := range ordered {
		number := volumeNumbers[volume.Number]

		for _, r := range selection.Ranges {
			matches := r.Mode == RangeOpenEnded &&
				number.Compare(r.Start) >= 0 ||
				r.Mode == RangeNormal &&
					number.Compare(r.Start) >= 0 &&
					number.Compare(r.End) <= 0

			if matches {
				selected = append(selected, volume)

				break
			}
		}
	}

	return selected
}

// FormatVolumeRanges renders the available volume numbers as compact
// ranges, the way chapter availability is reported.
func FormatVolumeRanges(volumes []sourcetypes.Volume) string {
	numbers := make([]int, 0, len(volumes))

	for _, volume := range volumes {
		if parsed, err := strconv.Atoi(volume.Number); err == nil {
			numbers = append(numbers, parsed)
		}
	}

	sort.Ints(numbers)

	parts := make([]string, 0, len(numbers))

	for i := 0; i < len(numbers); {
		end := i

		for end+1 < len(numbers) && numbers[end+1] == numbers[end]+1 {
			end++
		}

		if end == i {
			parts = append(parts, strconv.Itoa(numbers[i]))
		} else {
			parts = append(
				parts,
				fmt.Sprintf("%d-%d", numbers[i], numbers[end]),
			)
		}

		i = end + 1
	}

	return strings.Join(parts, ", ")
}

// PromptChaptersOrVolumes asks which listing an interactive download
// should use when the source offers both. Only an explicit answer
// picks volumes; everything else keeps the chapter flow.
func PromptChaptersOrVolumes(log *logger.Logger) bool {
	reader := bufio.NewReader(os.Stdin)

	log.Info(
		"This work splits into volumes on this source.\n",
	)

	for {
		log.Info(
			"Download volumes or chapters? (v/c, empty for chapters): ",
		)

		input, err := reader.ReadString('\n')
		if err != nil {
			log.Warn(
				"Failed to read input, using chapters: %v\n",
				err,
			)

			return false
		}

		switch strings.ToLower(strings.TrimSpace(input)) {
		case "v", "volumes":
			return true
		case "", "c", "chapters":
			return false
		}

		log.Warn("Please answer v or c.\n")
	}
}

// PromptVolumeRange asks for the volumes to download, with the same
// range syntax as the chapter prompt applied to volume numbers.
func PromptVolumeRange(
	volumes []sourcetypes.Volume,
	log *logger.Logger,
) (RangeSelection, error) {
	var selection RangeSelection

	reader := bufio.NewReader(os.Stdin)

	log.Info(
		"Found %d volume(s).\n",
		len(volumes),
	)

	log.Info(
		"Available volumes: %s\n",
		FormatVolumeRanges(volumes),
	)

	log.Info("Select volumes to download:\n")
	log.Info("  - all           : all available volumes\n")
	log.Info("  - 10            : only volume 10\n")
	log.Info("  - 1-10          : volumes 1 to 10\n")
	log.Info("  - 10-           : volume 10 onwards\n")
	log.Info("  - -10           : last 10 available volumes\n")
	log.Info("  - 1-5,8         : multiple selections\n")

	for {
		log.Info("Enter range: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return selection, fmt.Errorf(
				"failed to read input: %w",
				err,
			)
		}

		input = strings.TrimSpace(input)

		if input == "" {
			log.Warn(
				"Empty input. Please enter a valid range.\n",
			)

			continue
		}

		parsed, err := ParseRangeExpression(input)
		if err != nil {
			log.Warn("%v\n", err)

			continue
		}

		matched := FilterVolumesBySelection(
			volumes,
			parsed,
		)

		if len(matched) == 0 {
			log.Warn(
				"No available volumes match this selection.\n",
			)

			continue
		}

		return parsed, nil
	}
}

// volumeComplete reports whether every chapter of a volume has a
// usable artifact recorded. One missing chapter rebuilds the whole
// bundle: archives are all-or-nothing.
func volumeComplete(
	led *ledger.Ledger,
	format ledger.Format,
	entries []sourcetypes.Entry,
) bool {
	if len(entries) == 0 {
		return false
	}

	for _, entry := range entries {
		if led.FindArtifact(entry.Number, format) != nil {
			return false
		}
	}

	return true
}

// buildVolumeComicInfo assembles the metadata of one volume bundle.
// The volume is the issue: Volume carries its number, Number stays
// empty, and Count is the number of volumes the source lists. The
// enrichment mirrors the chapter archives.
func buildVolumeComicInfo(
	work sourcetypes.Work,
	volume sourcetypes.Volume,
	totalVolumes int,
	label string,
	workMeta sourcetypes.Metadata,
	languageISO string,
) *convert.ComicInfo {
	info := convert.ComicInfo{
		Series: work.Title,
		Title:  label,
		Count:  totalVolumes,
		Manga:  convert.MangaYes,
	}

	if parsed, err := strconv.Atoi(volume.Number); err == nil {
		info.Volume = parsed
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

// downloadVolumes walks the selected volumes sequentially; the
// downloader's politeness is untouched. Records land at chapter
// grain, so --continue can skip whole volumes and rebuild partial
// ones without ever shipping an incomplete bundle.
func downloadVolumes(
	provider sourcetypes.Provider,
	work sourcetypes.Work,
	totalVolumes int,
	volumes []sourcetypes.Volume,
	ledgerFile *ledger.Ledger,
	opts Options,
	workMeta sourcetypes.Metadata,
	log *logger.Logger,
) {
	log.Info(
		"Downloading %d volume(s)...\n",
		len(volumes),
	)

	requestHeaders := imageRequestHeaders(provider)
	format := outputFormat(opts)
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

	volDigits := volumeDigits(volumes)

	chDigits := 3
	for _, volume := range volumes {
		for _, entry := range volume.Entries {
			chDigits = max(
				chDigits,
				entry.Number.IntegerDigits(),
			)
		}
	}

	if opts.CBZ || opts.EbookFriendly {
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

	for _, volume := range volumes {
		label := volumeLabel(volume.Number, volDigits)

		if ledgerFile != nil && opts.Continue {
			// The bundled archive skips whole; per-chapter
			// artifacts skip chapter by chapter, downloading only
			// the missing ones into the volume directory.
			if opts.CBZ {
				if volumeComplete(ledgerFile, format, volume.Entries) {
					log.Info(
						"Skipping %s: already downloaded.\n",
						label,
					)

					continue
				}
			} else {
				volume.Entries = skipDownloaded(
					ledgerFile,
					format,
					volume.Entries,
					log,
				)

				if len(volume.Entries) == 0 {
					continue
				}
			}
		}

		log.Info(
			"Downloading %s...\n",
			label,
		)

		if opts.CBZ {
			bundleVolumeCBZ(
				provider,
				work,
				volume,
				totalVolumes,
				label,
				volDigits,
				chDigits,
				requestHeaders,
				languageCode,
				workMeta,
				ledgerFile,
				opts,
				homeDir,
				log,
			)

			continue
		}

		downloadVolumeArtifacts(
			provider,
			work,
			volume,
			volDigits,
			chDigits,
			requestHeaders,
			languageCode,
			workMeta,
			ledgerFile,
			opts,
			homeDir,
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

// bundleVolumeCBZ stages every chapter of one volume in order, then
// packs the whole volume into a single archive whose layout mirrors
// the on-disk series layout: one zero-padded folder per chapter.
// Staging into the same tree the chapters will occupy later keeps
// the archive reading order stable. Any failure abandons the bundle:
// no partial archive lands, no record is written.
func bundleVolumeCBZ(
	provider sourcetypes.Provider,
	work sourcetypes.Work,
	volume sourcetypes.Volume,
	totalVolumes int,
	label string,
	volDigits int,
	chDigits int,
	requestHeaders map[string]string,
	languageISO string,
	workMeta sourcetypes.Metadata,
	ledgerFile *ledger.Ledger,
	opts Options,
	homeDir string,
	log *logger.Logger,
) bool {
	staging := filepath.Join(
		homeDir,
		"Images",
		opts.Slug,
		label,
	)

	for _, entry := range volume.Entries {
		log.Info(
			"Downloading %s...\n",
			entry.Label,
		)

		chapterDir := filepath.Join(
			staging,
			entry.Number.Padded(chDigits),
		)

		imageURLs, err := provider.GetPageImageURLs(
			entry.URL,
			log,
		)
		if err != nil {
			log.Error(
				"%s not bundled: failed to collect image URLs for %s: %v\n",
				label,
				entry.Label,
				err,
			)

			_ = os.RemoveAll(staging)

			return false
		}

		if len(imageURLs) == 0 {
			log.Error(
				"%s not bundled: no image URLs found for %s\n",
				label,
				entry.Label,
			)

			_ = os.RemoveAll(staging)

			return false
		}

		if err := fetch.DownloadImagesWithHeaders(
			imageURLs,
			chapterDir,
			requestHeaders,
			log,
		); err != nil {
			log.Error(
				"%s not bundled: failed to download %s: %v\n",
				label,
				entry.Label,
				err,
			)

			_ = os.RemoveAll(staging)

			return false
		}
	}

	cbzPath := layoutVolumePath(
		opts.Layout,
		opts.ScanDir,
		work.Title,
		volume.Number,
		volDigits,
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

		_ = os.RemoveAll(staging)

		return false
	}

	if err := convert.ImageTreeToCBZ(
		staging,
		cbzPath,
		buildVolumeComicInfo(
			work,
			volume,
			totalVolumes,
			label,
			workMeta,
			languageISO,
		),
	); err != nil {
		log.Error(
			"%s not bundled: %v\n",
			label,
			err,
		)

		// A failed packing may leave the file behind; volume
		// bundles are all-or-nothing on disk too.
		_ = os.Remove(cbzPath)
		_ = os.RemoveAll(staging)

		return false
	}

	// The staged images live inside the archive now.
	_ = os.RemoveAll(staging)

	log.Info(
		"%s saved as %s\n",
		label,
		cbzPath,
	)

	for _, entry := range volume.Entries {
		recordDownloaded(
			ledgerFile,
			entry,
			ledger.FormatCBZ,
			cbzPath,
			log,
		)
	}

	return true
}

// downloadVolumeArtifacts writes one volume as per-chapter artifacts
// grouped inside its volume directory: one PDF per chapter, or one
// image folder per chapter in ebook-friendly mode. A failing chapter
// leaves the volume directory with the chapters that did finish;
// --continue rebuilds the missing ones chapter by chapter.
func downloadVolumeArtifacts(
	provider sourcetypes.Provider,
	work sourcetypes.Work,
	volume sourcetypes.Volume,
	volDigits int,
	chDigits int,
	requestHeaders map[string]string,
	languageISO string,
	workMeta sourcetypes.Metadata,
	ledgerFile *ledger.Ledger,
	opts Options,
	homeDir string,
	log *logger.Logger,
) {
	volumeDir := layoutVolumeDir(
		opts.Layout,
		opts.ScanDir,
		work.Title,
		volume.Number,
		volDigits,
	)

	if err := os.MkdirAll(
		volumeDir,
		os.ModePerm,
	); err != nil {
		log.Error(
			"Failed to create output dir: %v\n",
			err,
		)

		return
	}

	for _, entry := range volume.Entries {
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
			entryDir := volumeEntryImageDir(
				volumeDir,
				opts.Layout,
				work.Title,
				entry.Number,
				chDigits,
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

		// PDF output. Files are staged like the chapter flow: pages
		// download into a per-chapter folder, get packed, then the
		// folder is removed.
		imageDir := filepath.Join(
			homeDir,
			"Images",
			opts.Slug,
			entry.Number.String(),
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

		pdfPath := filepath.Join(
			volumeDir,
			volumeChapterBaseName(work.Title, entry.Number, chDigits)+".pdf",
		)

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

		_ = os.RemoveAll(imageDir)

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
}
