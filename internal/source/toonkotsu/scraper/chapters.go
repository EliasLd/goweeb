package toonkotsuscraper

import (
	"slices"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

// Chapter is one downloadable entry of a series listing, with every
// upload of the number folded into its URL as comma-joined ids.
type Chapter struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

// ListChapters returns one listing's title and its complete chapter
// list, oldest chapter first.
//
// The raw rows are newest-first and may repeat a number several times
// (legacy numberings carry one row per upload): same-number rows and
// their alternate uploads all fold into one entry URL, and the page
// step resolves them lazily, keeping the fullest copy. Unnumbered
// rows (webtoon extras, digitized "Volume N" entries) and unusable
// rows are skipped defensively. The scan path selects an edition's
// listing when set.
func ListChapters(
	domain string,
	code string,
	scanPath string,
	log *logger.Logger,
) (string, []Chapter, error) {
	seriesCode := strings.TrimSpace(code)

	// An edition selection names that edition's own series
	// record, fetched the same way.
	if edition := strings.TrimSpace(scanPath); edition != "" {
		seriesCode = edition
	}

	series, err := GetSeries(domain, seriesCode, log)
	if err != nil {
		return "", nil, err
	}

	uploads := buildChapterUploads(series, log)

	out := make([]Chapter, 0, len(uploads))

	for _, upload := range uploads {
		out = append(out, Chapter{
			Number: upload.number,
			Label:  upload.label,
			URL:    chapterEntryURL(domain, upload.ids),
		})
	}

	if log != nil {
		log.Debug(
			"Found %d toonkotsu chapters (listing %q)\n",
			len(out),
			seriesCode,
		)
	}

	return seriesTitle(series), out, nil
}

// chapterUpload is one merged chapter of a series listing: every
// upload of the same chapter number folded into one id set, in the
// site's own order. vols carries the canonical volume tags its rows
// are tagged with, empty when the site tags none.
type chapterUpload struct {
	number common.ChapterNumber
	label  string
	vols   []string
	ids    []string
}

// buildChapterUploads folds a series payload's chapter rows into
// merged uploads: unnumbered rows and unusable ids or numbers are
// skipped defensively with a debug line, and rows come out in reading
// order (the payload is newest-first).
func buildChapterUploads(
	series seriesResponse,
	log *logger.Logger,
) []chapterUpload {
	type numberedRow struct {
		row    chapterRow
		number common.ChapterNumber
	}

	numbered := make([]numberedRow, 0, len(series.Chapters))

	for _, row := range series.Chapters {
		if row.N == nil {
			if log != nil {
				log.Debug(
					"Skipping unnumbered toonkotsu chapter row: %q\n",
					row.Label,
				)
			}

			continue
		}

		number, err := common.ParseChapterNumber(row.N.String())
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping toonkotsu chapter row with invalid number %q: %q\n",
					row.N.String(),
					row.Label,
				)
			}

			continue
		}

		numbered = append(numbered, numberedRow{row: row, number: number})
	}

	// The listing's order is presentation-side (newest first);
	// the app expects reading order. Stable: uploads of the same
	// number keep the site's own preference order.
	sort.SliceStable(numbered, func(i, j int) bool {
		return numbered[i].number.Compare(numbered[j].number) < 0
	})

	uploads := make([]chapterUpload, 0, len(numbered))

	for _, entry := range numbered {
		ids := rowIDs(entry.row)

		if len(ids) == 0 {
			if log != nil {
				log.Debug(
					"Skipping toonkotsu chapter %s without ids: %q\n",
					entry.number.String(),
					entry.row.Label,
				)
			}

			continue
		}

		tag := volumeTag(entry.row, entry.number, log)

		// Same-number rows fold into the upload before them.
		if n := len(uploads); n > 0 &&
			uploads[n-1].number.Compare(entry.number) == 0 {
			uploads[n-1].ids = appendIDs(uploads[n-1].ids, ids)
			uploads[n-1].vols = appendVolTag(
				uploads[n-1].vols,
				tag,
			)

			continue
		}

		uploads = append(uploads, chapterUpload{
			number: entry.number,
			label:  chapterLabel(entry.row, entry.number),
			vols:   appendVolTag(nil, tag),
			ids:    ids,
		})
	}

	return uploads
}

// seriesTitle falls back to the site's name when a series record
// carries no title.
func seriesTitle(series seriesResponse) string {
	title := strings.TrimSpace(series.Series.Title)
	if title == "" {
		return "Toonkotsu"
	}

	return title
}

// chapterLabel prefers the site's own label and falls back to a
// "Chapter N" rendering.
func chapterLabel(
	row chapterRow,
	number common.ChapterNumber,
) string {
	if label := strings.TrimSpace(row.Label); label != "" {
		return label
	}

	return "Chapter " + number.String()
}

// volumeTag canonicalizes one row's volume tag; an empty or
// unparseable tag means the chapter is untagged.
func volumeTag(
	row chapterRow,
	number common.ChapterNumber,
	log *logger.Logger,
) string {
	tag := strings.TrimSpace(row.Vol)
	if tag == "" {
		return ""
	}

	volume, err := common.ParseChapterNumber(tag)
	if err != nil {
		if log != nil {
			log.Debug(
				"Ignoring toonkotsu volume tag %q on chapter %s: %q\n",
				tag,
				number.String(),
				row.Label,
			)
		}

		return ""
	}

	return volume.String()
}

// appendVolTag collects the volume tags its rows carry, without
// duplicates.
func appendVolTag(tags []string, addition string) []string {
	if addition == "" || slices.Contains(tags, addition) {
		return tags
	}

	return append(tags, addition)
}

// rowIDs collects the id of the row plus its alternate uploads; the
// row's own id leads so the site's preferred copy is tried first.
func rowIDs(row chapterRow) []string {
	ids := make([]string, 0, 1+len(row.Alts))

	if id := sanitizeID(row.ID); id != "" {
		ids = append(ids, id)
	}

	for _, alt := range row.Alts {
		if id := sanitizeID(alt.ID); id != "" {
			ids = append(ids, id)
		}
	}

	return ids
}

// sanitizeID guards the decoded id field against padding.
func sanitizeID(id rawID) string {
	return strings.TrimSpace(string(id))
}

// appendIDs merges candidate ids without duplicates: uploads of one
// number can share alternates.
func appendIDs(ids []string, addition []string) []string {
	for _, id := range addition {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}

	return ids
}

// chapterEntryURL folds a chapter's candidate uploads into one
// opaque URL the page step resolves:
// {domain}/manga/chapter/{id}[,{id}…].
func chapterEntryURL(domain string, ids []string) string {
	return domain + "/manga/chapter/" + strings.Join(ids, ",")
}
