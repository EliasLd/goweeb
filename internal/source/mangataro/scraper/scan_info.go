package mangataroscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

const chaptersPath = "/auth/manga-chapters"

// The route answers every page size, but the site's frontend walks
// 500-row pages; long-running titles span several of them.
const listingPageLimit = 500

// Safety cap against a listing that would claim unreasonably many
// pages: 20 pages carry ~10000 chapter rows.
const maxListingPages = 20

// chapterRow is one entry of the chapters listing. The data carries
// per-row fields the site filters on client-side (language, group),
// because its route offers no usable filter of its own.
type chapterRow struct {
	ID          string `json:"id"`
	Chapter     string `json:"chapter"`
	Title       string `json:"title"`
	ChapterType string `json:"chapter_type"`
	Language    string `json:"language"`
	URL         string `json:"url"`
}

type chapterListingResponse struct {
	Success  bool         `json:"success"`
	Message  string       `json:"message"`
	Chapters []chapterRow `json:"chapters"`
	Total    int          `json:"total"`
	HasMore  bool         `json:"has_more"`
}

// chapterListing carries one walk's result: its rows plus whether the
// walk got every row the route declared.
type chapterListing struct {
	rows          []chapterRow
	declaredTotal int
	complete      bool
}

// fetchChapterRows walks a title's chapter listing, offset page by
// offset page.
//
// Pagination is contractual: long-running titles span several
// 500-row pages (One Piece = 1206 rows over 3 pages) and counting
// from the first page alone undercounts.
func fetchChapterRows(
	domain string,
	mangaID string,
	log *logger.Logger,
) (chapterListing, error) {
	var result chapterListing

	for page := 0; ; page++ {
		token, timestamp := chapterRequestToken()

		query := url.Values{
			"manga_id": {mangaID},
			"offset":   {strconv.Itoa(page * listingPageLimit)},
			"limit":    {strconv.Itoa(listingPageLimit)},
			"order":    {"ASC"},
			"_t":       {token},
			"_ts":      {strconv.FormatInt(timestamp, 10)},
		}

		if log != nil {
			log.Debug(
				"Fetching MangaTaro chapter listing (page %d)\n",
				page+1,
			)
		}

		body, err := callAPI(
			domain,
			"chapter listing",
			http.MethodGet,
			chaptersPath,
			query,
			nil,
			log,
		)
		if err != nil {
			return chapterListing{}, err
		}

		var response chapterListingResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return chapterListing{}, fmt.Errorf(
				"failed to decode MangaTaro chapter listing: %w",
				err,
			)
		}

		if !response.Success {
			return chapterListing{}, fmt.Errorf(
				"MangaTaro chapter listing failed: %s",
				strings.TrimSpace(response.Message),
			)
		}

		result.rows = append(result.rows, response.Chapters...)
		result.declaredTotal = response.Total

		if !response.HasMore || len(response.Chapters) == 0 {
			break
		}

		if page+1 >= maxListingPages {
			break
		}
	}

	result.complete = result.declaredTotal == 0 ||
		len(result.rows) >= result.declaredTotal

	if !result.complete && log != nil {
		log.Debug(
			"MangaTaro chapter listing capped after %d pages (%d of %d rows)\n",
			maxListingPages,
			len(result.rows),
			result.declaredTotal,
		)
	}

	return result, nil
}

// LanguagePath is one selectable language of a work: the code stays
// opaque to the app and the label is what the scan-path screen shows.
type LanguagePath struct {
	Code  string
	Label string
}

// GetLanguagePaths returns one selectable item per language a work
// publishes, labeled with its count of distinct chapter numbers.
//
// The route applies no language filter (the verified `language`
// parameter is ignored), so the counts come from one unfiltered walk
// — exactly mirroring what a download of one language would get. A
// walk capped early must not present partial numbers as real counts,
// so its labels degrade to the bare language name.
func GetLanguagePaths(
	domain string,
	mangaID string,
	log *logger.Logger,
) ([]LanguagePath, error) {
	listing, err := fetchChapterRows(domain, mangaID, log)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]map[string]bool)

	for _, row := range listing.rows {
		language := strings.TrimSpace(row.Language)
		if language == "" {
			continue
		}

		number, err := common.ParseChapterNumber(row.Chapter)
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping MangaTaro chapter row with invalid number %q for language counts: %q\n",
					row.Chapter,
					row.Title,
				)
			}

			continue
		}

		numbers := counts[language]
		if numbers == nil {
			numbers = make(map[string]bool)
			counts[language] = numbers
		}

		numbers[number.String()] = true
	}

	paths := make([]LanguagePath, 0, len(counts))

	for code, numbers := range counts {
		name := languageName(code)

		// A partial walk must not lie about a partial count.
		if listing.complete {
			name = languagePathLabel(name, len(numbers))
		}

		paths = append(paths, LanguagePath{
			Code:  code,
			Label: name,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d MangaTaro languages for manga %s\n",
			len(paths),
			mangaID,
		)
	}

	// The walk's ordering is not contractual; labels sort
	// alphabetically like MangaDex's language picks do.
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Label < paths[j].Label
	})

	return paths, nil
}

// languagePathLabel formats a scan-path label with the chapter count
// (singular-aware, like the rest of the app's scan-path labels).
func languagePathLabel(name string, count int) string {
	if count == 1 {
		return fmt.Sprintf("%s (1 chapter)", name)
	}

	return fmt.Sprintf("%s (%d chapters)", name, count)
}
