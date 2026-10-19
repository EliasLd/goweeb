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
	mangaInfoPath         = "/api/manga"
	scanlatorFilterPrefix = "scanlator:"
)

// ScanPath is one selectable scan path of a manga.
type ScanPath struct {
	// Label is the enriched display text, e.g.
	// "Asura Scans (231 chapters, English)".
	Label string

	// Value is the id forwarded to ListChapters, unchanged.
	Value string

	// Count is the number of valid chapters this scan path lists;
	// unknown (fetch failure) is recorded as -1.
	Count int

	// Languages lists the distinct language codes of those
	// chapters, sorted alphabetically.
	Languages []string
}

type filterOptionsResponse struct {
	Groups []filterOptionsGroup `json:"groups"`
}

type filterOptionsGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mangaInfoResponse struct {
	Manga struct {
		Title string `json:"title"`
	} `json:"manga"`
}

// mangaEndpointPath builds the manga info endpoint of a manga:
//
//	{domain}/api/manga/{mangaID}
func mangaEndpointPath(mangaID string) string {
	return mangaInfoPath + "/" + url.PathEscape(mangaID)
}

// filterOptionsPath builds the scan filter options endpoint of a manga:
//
//	{domain}/api/manga/{mangaID}/chapters/filter-options
func filterOptionsPath(mangaID string) string {
	return mangaEndpointPath(mangaID) + "/chapters/filter-options"
}

// ListScanPaths lists the chapter upload groups exposed for a manga.
//
// The filter option ids are either numeric chapter upload identifiers
// ("18210") or "scanlator:" pseudo-ids for groups lacking their own
// upload table ("scanlator:Stick"). The value is forwarded to
// ListChapters verbatim.
//
// Each item is enriched with the chapter count and the languages of
// the chapters it would list, computed from the unfiltered chapter
// list (a single extra request matching the API's own filter
// semantics, see scanPathMatches). A failure to fetch that list only
// drops the enrichment: the scan paths stay listed, in API order.
func ListScanPaths(
	domain string,
	mangaID string,
	log *logger.Logger,
) ([]ScanPath, error) {
	body, err := apiGet(
		domain,
		"scan filter options",
		filterOptionsPath(mangaID),
		nil,
		log,
	)
	if err != nil {
		return nil, err
	}

	var payload filterOptionsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode mangadot.net scan filter options: %w",
			err,
		)
	}

	// -1 marks counts as unknown until the enrichment succeeds.
	items := make([]ScanPath, 0, len(payload.Groups))

	for _, group := range payload.Groups {
		id := strings.TrimSpace(group.ID)

		if !isSupportedFilterValue(id) {
			if log != nil {
				log.Debug(
					"Skipping mangadot.net filter option with unexpected id %q\n",
					id,
				)
			}

			continue
		}

		name := strings.TrimSpace(group.Name)

		if name == "" {
			// Mirror the other providers: fall back to the id so the
			// group remains selectable.
			name = id
		}

		items = append(items, ScanPath{
			Label: name,
			Value: id,
			Count: -1,
		})
	}

	if len(items) == 0 {
		// Some manga expose no group filter option. Keep a single
		// automatically selectable path covering every chapter.
		items = []ScanPath{
			{
				Label: "MangaDot",
				Value: "",
				Count: -1,
			},
		}
	} else if log != nil {
		log.Debug(
			"Found %d mangadot.net scan path(s)\n",
			len(items),
		)
	}

	rows, err := unfilteredChapterList(domain, mangaID, log)
	if err != nil {
		// The counts are a display nicety; a failure leaves the
		// scan paths unenriched and the download flow unaffected.
		if log != nil {
			log.Debug(
				"Skipping mangadot.net scan path counts: %v\n",
				err,
			)
		}

		return items, nil
	}

	return countScanPaths(items, rows, log), nil
}

// unfilteredChapterList fetches the chapter rows of a manga without
// any scan-path filter: the single request every scan path is counted
// from.
func unfilteredChapterList(
	domain string,
	mangaID string,
	log *logger.Logger,
) ([]chapterRow, error) {
	body, err := apiGet(
		domain,
		"chapter list",
		chaptersListPath(mangaID),
		nil,
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

	return rows, nil
}

// scanPathMatches reports whether a chapter row belongs to a scan
// path value, following the API's own filter semantics (verified
// live): group filters match rows whose groups contain the id,
// scanlator filters match the scanlator_name field, and the implicit
// empty path matches every valid row (deduped per number at count
// time).
func scanPathMatches(value string, row chapterRow) bool {
	value = strings.TrimSpace(value)

	if value == "" {
		return true
	}

	if strings.HasPrefix(value, scanlatorFilterPrefix) {
		name := strings.TrimSpace(
			strings.TrimPrefix(value, scanlatorFilterPrefix),
		)

		return name != "" &&
			strings.TrimSpace(row.ScanlatorName) == name
	}

	for _, group := range row.Groups {
		if strings.TrimSpace(
			group.ID.String(),
		) == value {
			return true
		}
	}

	return false
}

// countScanPaths enriches the scan paths with chapter counts and
// languages computed from the rows, and rebuilds their labels.
func countScanPaths(
	items []ScanPath,
	rows []chapterRow,
	log *logger.Logger,
) []ScanPath {
	type counter struct {
		chapters  map[string]struct{} // distinct chapter numbers
		uploads   int                 // matching valid rows
		languages map[string]struct{}
	}

	byValue := make(map[string]*counter, len(items))

	for _, item := range items {
		byValue[item.Value] = &counter{
			chapters:  map[string]struct{}{},
			languages: map[string]struct{}{},
		}
	}

	for _, row := range rows {
		if !chapterRowValid(row, log) {
			continue
		}

		number, err := common.ParseChapterNumber(
			strings.TrimSpace(row.ChapterNumber.String()),
		)
		if err != nil {
			continue
		}

		language := strings.TrimSpace(row.Language)

		for _, item := range items {
			if !scanPathMatches(item.Value, row) {
				continue
			}

			c := byValue[item.Value]

			if language != "" {
				c.languages[language] = struct{}{}
			}

			if item.Value == "" {
				// The implicit path dedupes chapters by
				// number, like the unfiltered chapter
				// listing does.
				c.chapters[number.String()] = struct{}{}
			}

			c.uploads++
		}
	}

	enriched := make([]ScanPath, 0, len(items))

	for _, item := range items {
		c := byValue[item.Value]

		count := c.uploads
		if item.Value == "" {
			count = len(c.chapters)
		}

		languages := make([]string, 0, len(c.languages))
		for code := range c.languages {
			languages = append(languages, languageName(code))
		}

		sort.Strings(languages)

		enriched = append(enriched, ScanPath{
			Value:     item.Value,
			Count:     count,
			Languages: languages,
			Label:     scanPathLabel(item.Label, count, languages),
		})
	}

	return enriched
}

// scanPathLabel builds the enriched display label of a scan path
// from its raw name, chapter count and display languages. Aggregator
// groups can upload in dozens of languages; the label keeps the
// three first ones at most, the full set staying on the Languages
// field.
func scanPathLabel(
	name string,
	count int,
	languages []string,
) string {
	chapterWord := "chapters"

	if count == 1 {
		chapterWord = "chapter"
	}

	label := fmt.Sprintf(
		"%s (%d %s",
		name,
		count,
		chapterWord,
	)

	shown := languages

	if len(shown) > 3 {
		shown = shown[:3]
	}

	if len(shown) > 0 {
		joined := strings.Join(shown, "/")

		if len(languages) > 3 {
			joined += "/…"
		}

		label += ", " + joined
	}

	return label + ")"
}

// isSupportedFilterValue validates a filter option id: either a numeric
// chapter upload id, or a "scanlator:{name}" pseudo-id.
func isSupportedFilterValue(id string) bool {
	if id == "" {
		return false
	}

	if strings.HasPrefix(id, scanlatorFilterPrefix) {
		return strings.TrimSpace(
			strings.TrimPrefix(id, scanlatorFilterPrefix),
		) != ""
	}

	_, err := strconv.ParseInt(id, 10, 64)
	return err == nil
}

// GetMangaTitle returns the canonical title of a manga.
func GetMangaTitle(
	domain string,
	mangaID string,
	log *logger.Logger,
) (string, error) {
	body, err := apiGet(
		domain,
		"manga info",
		mangaEndpointPath(mangaID),
		nil,
		log,
	)
	if err != nil {
		return "", err
	}

	var payload mangaInfoResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf(
			"failed to decode mangadot.net manga info: %w",
			err,
		)
	}

	title := strings.TrimSpace(payload.Manga.Title)

	if title == "" {
		return "", fmt.Errorf(
			"mangadot.net manga info has no title: %s",
			mangaID,
		)
	}

	return title, nil
}
