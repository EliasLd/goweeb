package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	chapterListingPath = "/api/v1/title/chapter-listing"

	// The API answers every page size, but caps single pages
	// server-side (verified at 5000). Long-running titles span
	// several pages, so the listing walk loops until exhausted.
	listingPageLimit = 5000

	// Safety cap against a listing API that would claim unreasonably
	// many pages: 20 pages carry ~100k chapter rows.
	maxListingPages = 20
)

type chapterRow struct {
	ID     string      `json:"id"`
	Number json.Number `json:"number"`
	Name   string      `json:"name"`
	Lang   string      `json:"lang"`
	Site   string      `json:"site"`
}

type listingPagination struct {
	Page       int `json:"page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type chapterListingResponse struct {
	Data               []chapterRow      `json:"data"`
	AvailableLanguages []string          `json:"available_languages"`
	Pagination         listingPagination `json:"pagination"`
}

type chapterRows struct {
	rows               []chapterRow
	availableLanguages []string
	complete           bool
}

// LanguagePath is one selectable language of a title: the code is
// opaque to the app and the Label is what the scan-path screen shows.
type LanguagePath struct {
	Code  string
	Label string
}

// fetchChapterRows walks a title's chapter listing, page by page,
// optionally filtered by language (empty = every language).
func fetchChapterRows(
	domain string,
	titleID string,
	language string,
	log *logger.Logger,
) (chapterRows, error) {
	var (
		result    chapterRows
		declared  int
		totalPage int
	)

	for page := 1; ; page++ {
		query := url.Values{
			"title_id": {titleID},
			"limit":    {strconv.Itoa(listingPageLimit)},
		}

		if language != "" {
			query.Set("language", language)
		}

		if page > 1 {
			query.Set("page", strconv.Itoa(page))
		}

		if log != nil {
			log.Debug(
				"Fetching MangaBall chapter listing (page %d, language %q)\n",
				page,
				language,
			)
		}

		body, err := apiGet(
			domain,
			"chapter listing",
			chapterListingPath,
			query,
			log,
		)
		if err != nil {
			return chapterRows{}, err
		}

		var response chapterListingResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return chapterRows{}, fmt.Errorf(
				"failed to decode MangaBall chapter listing: %w",
				err,
			)
		}

		result.rows = append(result.rows, response.Data...)
		declared = response.Pagination.Total
		totalPage = response.Pagination.TotalPages

		if response.AvailableLanguages != nil {
			result.availableLanguages = response.AvailableLanguages
		}

		if page >= totalPage || len(response.Data) == 0 {
			break
		}

		if page >= maxListingPages {
			break
		}
	}

	result.complete = declared == 0 ||
		len(result.rows) >= declared

	if !result.complete {
		if log != nil {
			log.Debug(
				"MangaBall chapter listing capped after %d pages (%d of %d rows)\n",
				maxListingPages,
				len(result.rows),
				declared,
			)
		}
	}

	return result, nil
}

// GetLanguagePaths returns one selectable item per language the title
// publishes, labeled with the count of distinct chapter numbers.
//
// The site does not expose per-language chapter counts, so they are
// counted client-side from an unfiltered listing — exactly mirroring
// the deduplication the download will apply (one entry per chapter
// number). When the walk had to stop early, labels degrade to plain
// language names instead of lying about a partial count.
func GetLanguagePaths(
	domain string,
	titleID string,
	log *logger.Logger,
) ([]LanguagePath, error) {
	listing, err := fetchChapterRows(domain, titleID, "", log)
	if err != nil {
		return nil, err
	}

	// Distinct chapter numbers per language.
	seen := make(map[string]map[string]bool)
	for _, row := range listing.rows {
		language := strings.TrimSpace(row.Lang)
		if language == "" {
			continue
		}

		numbers := seen[language]
		if numbers == nil {
			numbers = make(map[string]bool)
			seen[language] = numbers
		}

		numbers[row.Number.String()] = true
	}

	// Prefer the API's own language ordering, then any extra codes
	// that only appeared in rows.
	codes := listing.availableLanguages
	for language := range seen {
		if !containsCode(codes, language) {
			codes = append(codes, language)
		}
	}

	paths := make([]LanguagePath, 0, len(codes))

	for _, code := range codes {
		count := len(seen[code])
		if count == 0 {
			continue
		}

		label := languageName(code)

		// A partial walk must not present partial numbers as
		// real counts; fall back to the bare language name.
		if listing.complete {
			label = languagePathLabel(label, count)
		}

		paths = append(paths, LanguagePath{
			Code:  code,
			Label: label,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d MangaBall languages for title %s\n",
			len(paths),
			titleID,
		)
	}

	// The API's ordering is not contractual; labels sort
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

func containsCode(codes []string, code string) bool {
	for _, candidate := range codes {
		if candidate == code {
			return true
		}
	}

	return false
}
