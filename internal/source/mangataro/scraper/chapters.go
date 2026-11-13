package mangataroscraper

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

// Chapter is one downloadable entry of a work's chapter list.
type Chapter struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

// ListChapters returns the complete chapter list of one language,
// oldest chapter first.
//
// The site curates its listing: one row per chapter number per
// language, so one entry is emitted per distinct number and later
// rows repeating an already seen one are only logged, never merged
// (there is no evidence of multi-source copies to merge). Rows
// without usable id, number or url are skipped defensively; a listing
// the walk could not finish is a hard error (silently truncated
// chapters would download wrong ranges). An empty language filter
// keeps every language.
func ListChapters(
	domain string,
	mangaID string,
	language string,
	log *logger.Logger,
) ([]Chapter, error) {
	listing, err := fetchChapterRows(domain, mangaID, log)
	if err != nil {
		return nil, err
	}

	if !listing.complete {
		return nil, fmt.Errorf(
			"MangaTaro chapter listing is incomplete: %d of %d rows after %d pages",
			len(listing.rows),
			listing.declaredTotal,
			maxListingPages,
		)
	}

	type chapterCopy struct {
		number common.ChapterNumber
		row    chapterRow
	}

	seen := make(map[string]*chapterCopy)

	for _, row := range listing.rows {
		if language != "" &&
			!strings.EqualFold(strings.TrimSpace(row.Language), language) {
			continue
		}

		if row.ID == "" || strings.TrimSpace(row.URL) == "" {
			if log != nil {
				log.Debug(
					"Skipping MangaTaro chapter row without id or url: %q\n",
					row.Title,
				)
			}

			continue
		}

		number, err := common.ParseChapterNumber(row.Chapter)
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping MangaTaro chapter row with invalid number %q: %q\n",
					row.Chapter,
					row.Title,
				)
			}

			continue
		}

		key := number.String()

		if _, exists := seen[key]; exists {
			if log != nil {
				log.Debug(
					"Skipping duplicate MangaTaro chapter %s: %q\n",
					key,
					row.Title,
				)
			}

			continue
		}

		seen[key] = &chapterCopy{
			number: number,
			row:    row,
		}
	}

	chapters := make([]Chapter, 0, len(seen))

	for _, copy := range seen {
		chapters = append(chapters, Chapter{
			Number: copy.number,
			Label:  chapterLabel(copy.row),
			URL:    strings.TrimSpace(copy.row.URL),
		})
	}

	// The route's row order is presentation-side (newest first);
	// the app expects the reading order.
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].Number.Compare(chapters[j].Number) < 0
	})

	if log != nil {
		log.Debug(
			"Found %d MangaTaro chapters (language %q)\n",
			len(chapters),
			language,
		)
	}

	return chapters, nil
}

// chapterLabel prefers the site's own chapter title and falls back to
// a "Chapter N" rendering.
func chapterLabel(row chapterRow) string {
	if label := unescapeText(row.Title); label != "" {
		return label
	}

	return "Chapter " + strings.TrimSpace(row.Chapter)
}
