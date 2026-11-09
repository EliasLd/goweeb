package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	searchPath       = "/api/v1/title/search"
	maxSearchResults = 25
)

type MangaResult struct {
	Title string
	URL   string

	// Cover mirror address carried by the search row, empty when
	// the row lists no usable mirror.
	CoverURL string
}

type searchRequest struct {
	Keyword string `json:"keyword"`
	Limit   int    `json:"limit"`
}

type searchResponse struct {
	Data []searchTitle `json:"data"`
}

type searchTitle struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Slug  string       `json:"slug"`
	Image coverPayload `json:"image"`
}

// SearchCatalog queries the MangaBall search engine and returns the
// matching titles as site-canonical page URLs.
//
// The search endpoint is POST-only (the search screen is fully
// client-side) and answers {code,status,data:[…]} rows carrying the
// Mongo-style title id, display name and URL slug.
func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return []MangaResult{}, nil
	}

	body, err := apiPost(
		domain,
		"title search",
		searchPath,
		searchRequest{Keyword: query, Limit: maxSearchResults},
		log,
	)
	if err != nil {
		return nil, err
	}

	var response searchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode MangaBall search response: %w",
			err,
		)
	}

	results := make([]MangaResult, 0, len(response.Data))

	for _, title := range response.Data {
		// The id-slug pair is what every later step is built from;
		// rows missing it cannot be followed.
		if title.ID == "" || title.Slug == "" {
			if log != nil {
				log.Debug(
					"Skipping MangaBall search result without id or slug: %q\n",
					title.Name,
				)
			}

			continue
		}

		name := strings.TrimSpace(title.Name)
		if name == "" {
			if log != nil {
				log.Debug(
					"Skipping MangaBall search result without a name: %s\n",
					title.ID,
				)
			}

			continue
		}

		url, err := titleURL(domain, title.ID, title.Slug)
		if err != nil {
			return nil, err
		}

		results = append(results, MangaResult{
			Title: name,
			URL:   url,

			// The search rows list crawler CDNs for the cover;
			// the mangadex mirror is directly hotlinkable.
			CoverURL: coverFromImage(title.Image),
		})
	}

	if log != nil {
		log.Debug(
			"Found %d MangaBall search results for %q\n",
			len(results),
			query,
		)
	}

	return results, nil
}

// titleURL builds the site-canonical title-detail page URL from the
// id and slug returned by the search.
func titleURL(
	domain string,
	titleID string,
	slug string,
) (string, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return "", err
	}

	endpoint := *baseURL
	endpoint.Path += fmt.Sprintf("/title-detail/%s-%s", titleID, slug)

	return endpoint.String(), nil
}
