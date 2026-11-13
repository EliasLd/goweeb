package mangataroscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

// catalogPath is the REST route the site's own catalog page posts to.
const catalogPath = "/wp-json/manga/v1/load"

// catalogRequest mirrors the JSON body the site's catalog page posts.
// The filter fields stay JSON strings because the site serializes its
// filter arrays that way.
type catalogRequest struct {
	Page           int    `json:"page"`
	Search         string `json:"search"`
	Years          string `json:"years"`
	Genres         string `json:"genres"`
	Types          string `json:"types"`
	Statuses       string `json:"statuses"`
	Sort           string `json:"sort"`
	GenreMatchMode string `json:"genreMatchMode"`
}

type catalogRow struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Cover string `json:"cover"`
}

// MangaResult is one catalog match: the work URL the later methods
// consume, plus the cover the search screen renders for free.
type MangaResult struct {
	Title    string
	URL      string
	CoverURL string
}

// SearchCatalog posts the catalog route page by page, up to
// maxCatalogPages, and stops when a page comes back short. The route
// answers an empty array (never an error status) when the query
// matches nothing.
func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	results := make([]MangaResult, 0, catalogPageSize)

	for page := 1; page <= maxCatalogPages; page++ {
		body, err := callAPI(
			domain,
			"catalog search",
			http.MethodPost,
			catalogPath,
			nil,
			catalogRequest{
				Page:           page,
				Search:         query,
				Years:          catalogEmptyFilter,
				Genres:         catalogEmptyFilter,
				Types:          catalogEmptyFilter,
				Statuses:       catalogEmptyFilter,
				Sort:           catalogSort,
				GenreMatchMode: "",
			},
			log,
		)
		if err != nil {
			return nil, err
		}

		var rows []catalogRow
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf(
				"failed to decode MangaTaro catalog search: %w",
				err,
			)
		}

		for _, row := range rows {
			title := unescapeText(row.Title)

			if row.ID == "" || row.URL == "" || title == "" {
				if log != nil {
					log.Debug(
						"Skipping MangaTaro catalog row without id, url or title: %q\n",
						row.Title,
					)
				}

				continue
			}

			results = append(results, MangaResult{
				Title:    title,
				URL:      strings.TrimSpace(row.URL),
				CoverURL: strings.TrimSpace(row.Cover),
			})
		}

		if len(rows) < catalogPageSize {
			break
		}
	}

	if log != nil {
		log.Debug(
			"Found %d MangaTaro catalog results for %q\n",
			len(results),
			query,
		)
	}

	return results, nil
}
