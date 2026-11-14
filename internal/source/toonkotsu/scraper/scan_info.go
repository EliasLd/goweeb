package toonkotsuscraper

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

// GetScanPaths lists one work's selectable listings: the main series
// first, then one item per color edition, each labeled with its
// chapter count. The value is the series code each listing serves —
// the main listing rides the work's own code — and stays opaque to
// the app.
func GetScanPaths(
	domain string,
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	code, err := WorkCodeFromURL(workURL)
	if err != nil {
		return nil, err
	}

	series, err := GetSeries(domain, code, log)
	if err != nil {
		return nil, err
	}

	items := make([]common.SelectableItem, 0, 1+len(series.Editions))

	items = append(items, common.SelectableItem{
		Label: scanPathLabel(
			"Toonkotsu",
			countChapters(series.Chapters),
		),
		Value: code,
	})

	for _, edition := range series.Editions {
		editionCode := strings.TrimSpace(edition.Code)
		title := strings.TrimSpace(edition.Title)

		if editionCode == "" || title == "" {
			if log != nil {
				log.Debug(
					"Skipping toonkotsu edition without code or title on %s\n",
					code,
				)
			}

			continue
		}

		items = append(items, common.SelectableItem{
			Label: scanPathLabel(title, edition.Chapters),
			Value: editionCode,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d toonkotsu scan paths for %s\n",
			len(items),
			code,
		)
	}

	return items, nil
}

// countChapters counts the distinct chapter numbers of a listing.
// Unnumbered rows (webtoon extras, digitized volumes) carry no
// chapter number and never count.
func countChapters(rows []chapterRow) int {
	numbers := make(map[string]bool)

	for _, row := range rows {
		if row.N == nil {
			continue
		}

		number, err := common.ParseChapterNumber(row.N.String())
		if err != nil {
			continue
		}

		numbers[number.String()] = true
	}

	return len(numbers)
}

// scanPathLabel formats a scan-path label with the chapter count,
// singular-aware like the rest of the app's scan-path labels.
func scanPathLabel(name string, count int) string {
	if count == 1 {
		return fmt.Sprintf("%s (1 chapter)", name)
	}

	return fmt.Sprintf("%s (%d chapters)", name, count)
}
