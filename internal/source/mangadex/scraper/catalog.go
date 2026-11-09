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

	// Cover URL assembled when the search rows carry a cover_art
	// relationship; empty otherwise.
	CoverURL string
}

type mangaSearchResponse struct {
	Result string       `json:"result"`
	Data   []mangaEntry `json:"data"`
}

type mangaEntry struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Attributes   mangaAttributes     `json:"attributes"`
	Relationship []relationshipEntry `json:"relationships"`
}

type mangaAttributes struct {
	Title map[string]string `json:"title"`
}

// workCoverURL assembles one work's cover address from the
// cover_art relationship a search or detail payload carries.
func workCoverURL(
	mangaID string,
	relations []relationshipEntry,
) string {
	for _, relation := range relations {
		if relation.Type != "cover_art" {
			continue
		}

		if relation.Attributes == nil {
			continue
		}

		fileName := strings.TrimSpace(
			relation.Attributes.FileName,
		)

		return coverURL(coverCDNBase+"/"+mangaID, fileName)
	}

	return ""
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

	// The cover relationship rides along with each result, so the
	// search screen gets a cover per match without any extra call.
	q.Add("includes[]", "cover_art")

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

			CoverURL: workCoverURL(manga.ID, manga.Relationship),
		})
	}

	return results, nil
}

// domainURL parses the API base URL, rejecting empty or broken
// user-supplied domains.
func domainURL(domain string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSuffix(domain, "/"))
	if err != nil {
		return nil, fmt.Errorf(
			"invalid MangaDex domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("MangaDex domain is empty")
	}

	return baseURL, nil
}

// selectLocalized picks a display string out of a per-locale map:
// common readable variants first, then a deterministic fallback
// instead of relying on Go's map order.
func selectLocalized(
	localized map[string]string,
) string {
	if len(localized) == 0 {
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
		if value := strings.TrimSpace(localized[language]); value != "" {
			return value
		}
	}

	// Deterministic fallback instead of relying on Go's map order.
	keys := make([]string, 0, len(localized))

	for language := range localized {
		keys = append(keys, language)
	}

	sort.Strings(keys)

	for _, language := range keys {
		if value := strings.TrimSpace(localized[language]); value != "" {
			return value
		}
	}

	return ""
}

// selectMangaTitle picks the display title of a manga from its
// per-language map.
func selectMangaTitle(
	titles map[string]string,
) string {
	return selectLocalized(titles)
}
