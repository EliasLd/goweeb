package mangadexscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

type MangaResult struct {
	Title string
	URL   string
}

type mangaSearchResponse struct {
	Result string       `json:"result"`
	Data   []mangaEntry `json:"data"`
}

type mangaEntry struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes mangaAttributes `json:"attributes"`
}

type mangaAttributes struct {
	Title map[string]string `json:"title"`
}

func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	base := strings.TrimSuffix(domain, "/")

	u, err := url.Parse(base + "/manga")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to build MangaDex search URL: %w",
			err,
		)
	}

	q := u.Query()
	q.Set("title", query)
	q.Set("limit", "50")
	q.Set("order[relevance]", "desc")

	u.RawQuery = q.Encode()

	searchURL := u.String()

	log.Debug(
		"Searching MangaDex catalog: %s\n",
		searchURL,
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		searchURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaDex search request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "goweeb")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to search MangaDex catalog: %w",
			err,
		)
	}

	log.Debug(
		"MangaDex catalog response status: %d\n",
		result.StatusCode,
	)

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"MangaDex catalog returned status: %d",
			result.StatusCode,
		)
	}

	var response mangaSearchResponse

	if err := json.Unmarshal(result.Body, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode MangaDex response: %w",
			err,
		)
	}

	if response.Result != "ok" {
		return nil, fmt.Errorf(
			"MangaDex returned unexpected result: %s",
			response.Result,
		)
	}

	results := make([]MangaResult, 0, len(response.Data))

	for _, manga := range response.Data {
		if manga.ID == "" {
			continue
		}

		title := selectMangaTitle(manga.Attributes.Title)

		if title == "" {
			continue
		}

		results = append(results, MangaResult{
			Title: title,

			// Keep an API URL as the work URL.
			// Later methods can directly extract the UUID from it.
			URL: fmt.Sprintf(
				"%s/manga/%s",
				base,
				manga.ID,
			),
		})
	}

	return results, nil
}

func selectMangaTitle(
	titles map[string]string,
) string {
	if len(titles) == 0 {
		return ""
	}

	// Prefer common readable title variants.
	preferredLanguages := []string{
		"en",
		"fr",
		"es",
		"ja-ro",
		"ja",
		"ko",
		"ko-ro",
		"zh",
		"zh-hk",
	}

	for _, language := range preferredLanguages {
		if title := strings.TrimSpace(titles[language]); title != "" {
			return title
		}
	}

	// Deterministic fallback instead of relying on Go's map order.
	keys := make([]string, 0, len(titles))

	for language := range titles {
		keys = append(keys, language)
	}

	sort.Strings(keys)

	for _, language := range keys {
		if title := strings.TrimSpace(titles[language]); title != "" {
			return title
		}
	}

	return ""
}
