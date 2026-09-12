package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/EliasLd/scan-scraper/internal/convert"
	"github.com/EliasLd/scan-scraper/internal/fetch"
	"github.com/EliasLd/scan-scraper/internal/logger"
	"github.com/EliasLd/scan-scraper/internal/source"
	"github.com/EliasLd/scan-scraper/internal/source/common"
	sourcetypes "github.com/EliasLd/scan-scraper/internal/source/types"
)

func chapterDigits(maxChapter int) int {
	d := len(fmt.Sprintf("%d", maxChapter))
	if d < 3 {
		return 3
	}
	return d
}

func toSelectableManga(in []sourcetypes.SearchResult) []common.SelectableItem {
	out := make([]common.SelectableItem, 0, len(in))
	for _, r := range in {
		out = append(out, common.SelectableItem{
			Label: r.Title,
			Value: r.URL,
		})
	}
	return out
}

// Filters entries based on user options.
func filterEntries(entries []sourcetypes.Entry, opts Options, log *logger.Logger) []sourcetypes.Entry {
	if opts.RangeMode == RangeLastN {
		n := min(opts.Range[1], len(entries))
		entries = entries[len(entries)-n:]
		log.Info("Filtered to the last %d chapters\n", n)
	} else if opts.Range != [2]int{} {
		var filtered []sourcetypes.Entry
		for _, e := range entries {
			if opts.RangeMode == RangeOpenEnded && opts.Range[1] == 0 {
				if e.Number >= opts.Range[0] {
					filtered = append(filtered, e)
				}
			} else {
				if e.Number >= opts.Range[0] && e.Number <= opts.Range[1] {
					filtered = append(filtered, e)
				}
			}
		}
		entries = filtered
		if opts.RangeMode == RangeOpenEnded {
			log.Info("Filtered to %d chapters from %d onwards\n", len(entries), opts.Range[0])
		} else {
			log.Info("Filtered to %d chapters (%d-%d)\n", len(entries), opts.Range[0], opts.Range[1])
		}
	}
	return entries
}

func downloadEntries(
	provider sourcetypes.Provider,
	work sourcetypes.Work,
	entries []sourcetypes.Entry,
	opts Options,
	log *logger.Logger,
) {
	log.Info("Downloading %d chapters...\n", len(entries))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error("Failed to get home dir: %v\n", err)
		return
	}

	maxNum := entries[len(entries)-1].Number
	digits := chapterDigits(maxNum)

	if opts.EbookFriendly {
		if err := os.MkdirAll(opts.ScanDir, os.ModePerm); err != nil {
			log.Error("Failed to create output dir: %v\n", err)
			return
		}
	}

	for _, entry := range entries {
		chStr := fmt.Sprintf("%d", entry.Number)
		log.Info("Downloading %s...\n", entry.Label)

		imageURLs, err := provider.GetPageImageURLs(entry.URL, log)
		if err != nil {
			log.Error("Failed to collect image URLs for %s: %v\n", entry.Label, err)
			continue
		}
		if len(imageURLs) == 0 {
			log.Error("No image URLs found for %s\n", entry.Label)
			continue
		}

		if opts.EbookFriendly {
			prefix := "Chapter"
			if work.Kind == sourcetypes.ItemVolume {
				prefix = "Volume"
			}

			entryDir := filepath.Join(opts.ScanDir, fmt.Sprintf("%s %0*d", prefix, digits, entry.Number))
			if err := fetch.DownloadImages(imageURLs, entryDir, log); err != nil {
				log.Error("Failed to download %s: %v\n", entry.Label, err)
				continue
			}

			log.Info("%s saved to %s\n", entry.Label, entryDir)
			continue
		}

		imageDir := filepath.Join(homeDir, "Images", opts.Slug, chStr)
		if err := fetch.DownloadImages(imageURLs, imageDir, log); err != nil {
			log.Error("Failed to download %s: %v\n", entry.Label, err)
			continue
		}

		if err := os.MkdirAll(opts.ScanDir, os.ModePerm); err != nil {
			log.Error("Failed to create output dir: %v\n", err)
			return
		}

		pdfPath := filepath.Join(opts.ScanDir, fmt.Sprintf("%s_%s.pdf", work.Title, chStr))
		if err := convert.ImagesToPDF(imageDir, pdfPath, opts.Cleanup, log); err != nil {
			log.Error("Failed to create PDF: %v\n", err)
			continue
		}

		log.Info("%s saved as %s\n", entry.Label, pdfPath)
	}

	if opts.Cleanup && !opts.EbookFriendly {
		rootImagesDir := filepath.Join(homeDir, "Images", opts.Slug)
		log.Debug("Cleaning up: %s\n", rootImagesDir)
		_ = os.RemoveAll(rootImagesDir)
	}
}

func Run(opts Options, log *logger.Logger) {
	provider, err := source.New(opts.Source, opts.CustomDomain)
	if err != nil {
		log.Error("Failed to initialize source provider: %v\n", err)
		return
	}

	log.Info("Searching for manga: %s\n", opts.Slug)

	searchResults, err := provider.Search(opts.Slug, log)
	if err != nil {
		log.Error("Failed to search catalog: %v\n", err)
		return
	}
	if len(searchResults) == 0 {
		log.Error("Manga '%s' not found in catalog\n", opts.Slug)
		return
	}

	if len(searchResults) == 1 {
		log.Info("Found: %s\n", searchResults[0].Title)
	} else {
		log.Info("Found %d results.\n", len(searchResults))
	}

	mangaURL, err := common.PromptUserToSelect(
		toSelectableManga(searchResults),
		"Multiple results found:",
		"Select a manga (1-%d) or 0 to cancel: ",
		log,
	)
	if err != nil {
		log.Error("Failed to select manga: %v\n", err)
		return
	}
	if mangaURL == "" {
		log.Warn("No manga selected. Exiting.")
		return
	}

	scanPaths, err := provider.ListScanPaths(mangaURL, log)
	if err != nil {
		log.Error("Failed to list scan paths: %v\n", err)
		return
	}
	if len(scanPaths) == 0 {
		log.Warn("No scan versions found.")
		return
	}

	scanPath, err := common.PromptUserToSelect(
		scanPaths,
		"Multiple scan versions found:",
		"Select a scan version (1-%d) or 0 to cancel: ",
		log,
	)
	if err != nil {
		log.Error("Failed to select scan path: %v\n", err)
		return
	}
	if scanPath == "" {
		log.Warn("No scan version selected. Exiting.")
		return
	}

	work, entries, err := provider.ListEntries(mangaURL, scanPath, log)
	if err != nil {
		log.Error("Failed to list entries: %v\n", err)
		return
	}
	if len(entries) == 0 {
		log.Warn("No entries found.")
		return
	}

	if !opts.All && opts.RangeMode == RangeNone {
		chRange, chMode, all, err := PromptChapterRange(len(entries), log)
		if err != nil {
			log.Error("Failed to read chapter range: %v\n", err)
			return
		}
		opts.All = all
		opts.Range = chRange
		opts.RangeMode = chMode
	}

	entries = filterEntries(entries, opts, log)
	if len(entries) == 0 {
		log.Warn("No chapters found in range.")
		return
	}

	downloadEntries(provider, work, entries, opts, log)
}

func RunWithWorkflow(opts Options, log *logger.Logger) {
	if !opts.All && opts.RangeMode == 0 {
		log.Warn("Please enter a valid range to download chapters.")
		return
	}

	provider, err := source.New(opts.Source, opts.CustomDomain)
	if err != nil {
		log.Error("Failed to initialize source provider: %v\n", err)
		return
	}

	work, entries, err := provider.ListEntries(opts.MangaURL, opts.ScanPath, log)
	if err != nil {
		log.Error("Failed to list entries: %v\n", err)
		return
	}
	if len(entries) == 0 {
		log.Warn("No entries found.")
		return
	}

	entries = filterEntries(entries, opts, log)
	if len(entries) == 0 {
		log.Warn("No chapters found in range.")
		return
	}

	downloadEntries(provider, work, entries, opts, log)
	log.Info("Finished downloading")
}
