package atsumoescraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	searchPath       = "/collections/manga/documents/search"
	maxResponseBytes = 8 << 20
)

type MangaResult struct {
	Title string
	URL   string
}

type searchResponse struct {
	Hits []searchHit `json:"hits"`
}

type searchHit struct {
	Document searchDocument `json:"document"`
}

type searchDocument struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	EnglishTitle string `json:"englishTitle"`
	IsAdult      bool   `json:"isAdult"`
}

func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []MangaResult{}, nil
	}

	baseURL, err := url.Parse(strings.TrimSpace(domain))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse Atsu.moe domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid Atsu.moe domain: %q",
			domain,
		)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	endpoint := *baseURL
	endpoint.Path += searchPath

	params := endpoint.Query()
	params.Set("q", query)
	params.Set(
		"query_by",
		"title,englishTitle,otherNames,authors,acronyms",
	)
	params.Set("query_by_weights", "4,3,2,2,1")
	params.Set("num_typos", "4,3,2,1,0")
	params.Set("prefix", "true,true,true,true,false")
	params.Set(
		"include_fields",
		"id,title,englishTitle,authors,poster,posterSmall,"+
			"posterMedium,type,medium,isAdult,status,year,"+
			"mbRating,popularity,dateAdded",
	)
	params.Set(
		"filter_by",
		"isAdult:=false && views:>0 && hidden:!=true",
	)
	params.Set("page", "1")
	params.Set("per_page", "40")
	params.Set("infix", "off,off,fallback,off,off")

	endpoint.RawQuery = params.Encode()

	if log != nil {
		log.Debug(
			"Fetching Atsu.moe catalog: %s\n",
			endpoint.String(),
		)
	}

	req, err := http.NewRequest(
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create Atsu.moe search request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch Atsu.moe catalog: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if log != nil {
		log.Debug(
			"Atsu.moe catalog response status: %d\n",
			resp.StatusCode,
		)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(
			io.LimitReader(resp.Body, 4096),
		)

		return nil, fmt.Errorf(
			"Atsu.moe catalog returned status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	body, err := io.ReadAll(
		io.LimitReader(resp.Body, maxResponseBytes),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read Atsu.moe catalog response: %w",
			err,
		)
	}

	var payload searchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode Atsu.moe catalog response: %w",
			err,
		)
	}

	results := make([]MangaResult, 0, len(payload.Hits))

	for _, hit := range payload.Hits {
		doc := hit.Document

		// Keep a local safeguard in addition to the API filter.
		if doc.IsAdult {
			continue
		}

		id := strings.TrimSpace(doc.ID)
		title := strings.TrimSpace(doc.Title)

		if title == "" {
			title = strings.TrimSpace(doc.EnglishTitle)
		}

		if id == "" || title == "" {
			continue
		}

		mangaURL := *baseURL
		mangaURL.Path = strings.TrimRight(
			mangaURL.Path,
			"/",
		) + "/manga/" + id

		results = append(results, MangaResult{
			Title: title,
			URL:   mangaURL.String(),
		})
	}

	if log != nil {
		log.Debug(
			"Found %d Atsu.moe manga result(s)\n",
			len(results),
		)
	}

	return results, nil
}
