package mangadotnetscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

const (
	chaptersListSuffix = "/chapters/list"
	userSource         = "user"
	defaultChapterLbl  = "Chapter"
)

// Chapter is a downloadable chapter of a manga.
type Chapter struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

// flexString accepts both JSON numbers and JSON strings. MangaDot sends
// identifiers and chapter numbers as JSON numbers in most payloads and
// as quoted zero-padded strings ("1.00") in others.
type flexString string

func (s *flexString) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" {
		*s = ""
		return nil
	}

	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}

	*s = flexString(text)

	return nil
}

func (s flexString) String() string {
	return string(s)
}

// chapterRow keeps only the fields needed for the chapter listing; the
// API sends more (volume_number, uploader_*, groups, comment_count...).
type chapterRow struct {
	ID            flexString `json:"id"`
	ChapterNumber flexString `json:"chapter_number"`
	ChapterTitle  string     `json:"chapter_title"`
	DateAdded     string     `json:"date_added"`
	PageCount     int        `json:"page_count"`
	Source        string     `json:"source"`
}

// chapterRecord pairs a parsed chapter with its upload date. Dates
// share one format on this site, so plain string comparison orders
// them chronologically.
type chapterRecord struct {
	chapter   Chapter
	dateAdded string
}

// chaptersListPath builds the chapter list endpoint of a manga:
//
//	{domain}/api/manga/{mangaID}/chapters/list
func chaptersListPath(mangaID string) string {
	return mangaEndpointPath(mangaID) + chaptersListSuffix
}

// ListChapters returns the chapters of a manga, ascending.
//
// scanPath selects an upload group, using a value returned by
// ListScanPaths:
//
//	""                 no filter: every chapter of the manga, deduped
//	                   by number (the newest upload is kept)
//	"18210"            the chapters uploaded by group 18210
//	"scanlator:Stick"  the chapters uploaded by scanlator Stick
//
// The API returns rows newest-first. Every row carries its source
// ("user" or "scraper"); the entry URL is the public reader endpoint
// that GetPageImageURLs parses next:
//
//	{domain}/chapter/{chapterID}             (source "scraper")
//	{domain}/chapter/{chapterID}?source=user (source "user")
func ListChapters(
	domain string,
	mangaID string,
	scanPath string,
	log *logger.Logger,
) ([]Chapter, error) {
	query, err := chapterListQuery(scanPath)
	if err != nil {
		return nil, err
	}

	body, err := apiGet(
		domain,
		"chapter list",
		chaptersListPath(mangaID),
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	var rows []chapterRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf(
			"failed to decode mangadot.net chapter list: %w",
			err,
		)
	}

	// Corrupted rows are skipped, never a hard failure.
	records := make([]chapterRecord, 0, len(rows))

	for _, row := range rows {
		record, ok := parseChapterRow(domain, row, log)
		if !ok {
			continue
		}

		records = append(records, record)
	}

	scanPath = strings.TrimSpace(scanPath)

	if scanPath == "" {
		// Without a scan path, every upload of every group is
		// returned. Downloads are written per chapter number, so keep
		// one row per number.
		records = dedupeChapters(records, log)
	}

	sort.SliceStable(records, func(i, j int) bool {
		cmp := records[i].chapter.Number.Compare(
			records[j].chapter.Number,
		)
		if cmp != 0 {
			return cmp < 0
		}

		return records[i].dateAdded < records[j].dateAdded
	})

	chapters := make([]Chapter, 0, len(records))

	for _, record := range records {
		chapters = append(chapters, record.chapter)
	}

	if log != nil {
		log.Debug(
			"Found %d mangadot.net chapter(s)\n",
			len(chapters),
		)
	}

	return chapters, nil
}

// chapterListQuery translates a scan path value into the matching
// chapter list filter parameters.
func chapterListQuery(scanPath string) (url.Values, error) {
	query := url.Values{}

	scanPath = strings.TrimSpace(scanPath)

	if scanPath == "" {
		return query, nil
	}

	if strings.HasPrefix(scanPath, scanlatorFilterPrefix) {
		name := strings.TrimSpace(
			strings.TrimPrefix(scanPath, scanlatorFilterPrefix),
		)

		if name == "" {
			return nil, fmt.Errorf(
				"unexpected mangadot.net scan path: %q",
				scanPath,
			)
		}

		query.Set("scanlator_name", name)

		return query, nil
	}

	groupID, err := strconv.ParseInt(scanPath, 10, 64)
	if err != nil || groupID <= 0 {
		return nil, fmt.Errorf(
			"unexpected mangadot.net scan path: %q",
			scanPath,
		)
	}

	query.Set("group_id", strconv.FormatInt(groupID, 10))

	return query, nil
}

// parseChapterRow converts an API chapter row into a downloadable
// chapter. The second return value reports whether the row is usable.
func parseChapterRow(
	domain string,
	row chapterRow,
	log *logger.Logger,
) (chapterRecord, bool) {
	drop := func(reason string) {
		if log != nil {
			log.Debug(
				"Skipping mangadot.net chapter row %q: %s\n",
				row.ID.String(),
				reason,
			)
		}
	}

	chapterID := strings.TrimSpace(row.ID.String())

	if _, err := strconv.ParseInt(chapterID, 10, 64); err != nil {
		drop("invalid id")
		return chapterRecord{}, false
	}

	number, err := common.ParseChapterNumber(
		strings.TrimSpace(row.ChapterNumber.String()),
	)
	if err != nil {
		drop("invalid chapter number")
		return chapterRecord{}, false
	}

	if row.PageCount <= 0 {
		drop("no page")
		return chapterRecord{}, false
	}

	entryURL, err := chapterURL(
		domain,
		chapterID,
		strings.TrimSpace(row.Source),
	)
	if err != nil {
		drop("invalid chapter URL")
		return chapterRecord{}, false
	}

	label := strings.TrimSpace(row.ChapterTitle)
	if label == "" {
		label = defaultChapterLbl + " " + number.String()
	}

	return chapterRecord{
		chapter: Chapter{
			Number: number,
			Label:  label,
			URL:    entryURL,
		},
		dateAdded: row.DateAdded,
	}, true
}

// chapterURL builds the public reader endpoint of a chapter, used as
// entry URL by GetPageImageURLs:
//
//	{domain}/chapter/{chapterID}              (scraper-source rows)
//	{domain}/chapter/{chapterID}?source=user  (user-source rows)
//
// Chapter ids belong to two separate id spaces (user uploads vs
// imported chapters), so the source travels in the entry URL. The API
// only sends "user" and "scraper"; any other value is treated as a
// scraper chapter.
func chapterURL(domain, chapterID, source string) (string, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return "", err
	}

	endpoint := *baseURL
	endpoint.Path += "/chapter/" + url.PathEscape(chapterID)

	if source == userSource {
		query := url.Values{}
		query.Set("source", userSource)
		endpoint.RawQuery = query.Encode()
	}

	return endpoint.String(), nil
}

// dedupeChapters keeps one chapter per number, always the newest
// upload. Only used for unfiltered lists.
func dedupeChapters(records []chapterRecord, log *logger.Logger) []chapterRecord {
	byNumber := make(map[common.ChapterNumber]int, len(records))
	kept := make([]chapterRecord, 0, len(records))

	for _, record := range records {
		index, seen := byNumber[record.chapter.Number]

		if seen {
			keeper, dropped := kept[index], record

			// Keep the newest upload; on equal dates the
			// already-kept row wins (first seen).
			if dropped.dateAdded > keeper.dateAdded {
				keeper, dropped = record, kept[index]
			}

			if log != nil {
				log.Debug(
					"Dropping duplicate mangadot.net chapter %s upload from %q, keeping %q\n",
					record.chapter.Number,
					dropped.dateAdded,
					keeper.dateAdded,
				)
			}

			kept[index] = keeper

			continue
		}

		byNumber[record.chapter.Number] = len(kept)
		kept = append(kept, record)
	}

	return kept
}
