package toonkotsuscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

// browsePath is the catalogue search route, same origin as the site.
const browsePath = "/cdn/v1/browse"

const (
	// The route caps the page size at 100 server-side; search
	// screens never need the whole catalogue, so two pages keep
	// the search output in line with the other providers.
	searchLimit    = 100
	maxSearchPages = 2

	// The catalogue hosts adult works; goweeb always filters them
	// out. The route serves them by default otherwise.
	sfwFilter = "1"
)

// browseItem is one catalogue match. code and slug build the
// site-canonical /title/{code}-{slug} URL the later methods consume.
type browseItem struct {
	Code  string `json:"code"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Cover string `json:"cover"`
}

type browseResponse struct {
	Items       []browseItem `json:"items"`
	HasNextPage bool         `json:"hasNextPage"`
}

// MangaResult is one catalog match: the work URL the later methods
// consume, plus the cover the search screen renders for free.
type MangaResult struct {
	Title    string
	URL      string
	CoverURL string
}

// SearchCatalog walks the browse route page by page, up to
// maxSearchPages, stopping when the route says the listing ends or a
// page adds nothing new. The route sorts by relevance when q is set
// and answers an honest empty item list when the query matches
// nothing.
func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	results := make([]MangaResult, 0, searchLimit)

	seen := make(map[string]bool)

	for page := 1; page <= maxSearchPages; page++ {
		body, err := callAPI(
			domain,
			"catalog search",
			browsePath,
			url.Values{
				"q":     {query},
				"sfw":   {sfwFilter},
				"limit": {strconv.Itoa(searchLimit)},
				"page":  {fmt.Sprintf("%d", page)},
			},
			log,
		)
		if err != nil {
			return nil, err
		}

		var response browseResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf(
				"failed to decode toonkotsu catalog search: %w",
				err,
			)
		}

		fresh := 0

		for _, item := range response.Items {
			code := strings.TrimSpace(item.Code)
			slug := strings.TrimSpace(item.Slug)
			title := strings.TrimSpace(item.Title)

			if code == "" || slug == "" || title == "" {
				if log != nil {
					log.Debug(
						"Skipping toonkotsu catalog row without code, slug or title: %q\n",
						item.Title,
					)
				}

				continue
			}

			if seen[code] {
				continue
			}
			seen[code] = true

			results = append(results, MangaResult{
				Title: title,
				URL:   workURL(domain, code, slug),
				CoverURL: absoluteURL(
					domain,
					item.Cover,
				),
			})

			fresh++
		}

		// A page adding nothing new means the route is not
		// advancing; walking on would only repeat rows.
		if fresh == 0 || !response.HasNextPage {
			break
		}
	}

	if log != nil {
		log.Debug(
			"Found %d toonkotsu catalog results for %q\n",
			len(results),
			query,
		)
	}

	return results, nil
}

// workURL builds the site-canonical work URL: /title/{code}-{slug}.
// The code is the short prefix; the slug may itself contain hyphens.
func workURL(domain, code, slug string) string {
	return domain + "/title/" + code + "-" + slug
}

// absoluteURL turns the payloads' same-origin paths into full URLs;
// values that already carry one pass through unchanged.
func absoluteURL(domain, raw string) string {
	value := strings.TrimSpace(raw)

	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") {
		return value
	}

	return domain + value
}
