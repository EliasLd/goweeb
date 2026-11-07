package mangadotnetscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const searchPath = "/api/search"

type MangaResult struct {
	ID    string
	Title string
}

// flexBool decodes the JSON flags the search API serves either as
// 0/1 numbers or as true/false booleans. The site drifted between
// the two shapes over time, so the field type is defined by a
// custom decoder that refuses nothing the API sends.
type flexBool bool

func (f *flexBool) UnmarshalJSON(data []byte) error {
	value := strings.Trim(string(data), "\" ")

	switch value {
	case "true", "1":
		*f = true
	case "false", "0", "null", "":
		*f = false
	default:
		return fmt.Errorf(
			"unexpected flag value %s",
			data,
		)
	}

	return nil
}

type searchResponse struct {
	MangaList []searchItem `json:"manga_list"`
}

type searchItem struct {
	ID      json.Number `json:"id"`
	Title   string      `json:"title"`
	IsAdult flexBool    `json:"is_adult"`
}

// SearchCatalog queries a manga by name. The API pagination sends
// 20 results per page and the first page is always enough for a user
// search, so no page parameter is sent.
func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []MangaResult{}, nil
	}

	params := url.Values{}
	params.Set("search", query)

	body, err := apiGet(domain, "search", searchPath, params, log)
	if err != nil {
		return nil, err
	}

	var payload searchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode mangadot.net search: %w",
			err,
		)
	}

	results := make([]MangaResult, 0, len(payload.MangaList))

	for _, item := range payload.MangaList {
		// Adult doujinshi are served by the API by default and are
		// flagged with is_adult: 1. Keep a local safeguard on top of
		// it, same as the other providers.
		if item.IsAdult {
			continue
		}

		id := strings.TrimSpace(item.ID.String())

		// Manga identifiers are positive integers; anything else is
		// treated as a corrupted row.
		numericID, err := strconv.ParseInt(id, 10, 64)
		if err != nil || numericID <= 0 {
			if log != nil {
				log.Debug(
					"Skipping mangadot.net search item with invalid ID %q\n",
					id,
				)
			}

			continue
		}

		title := strings.TrimSpace(item.Title)

		if title == "" {
			continue
		}

		results = append(results, MangaResult{
			ID:    id,
			Title: title,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d mangadot.net manga result(s)\n",
			len(results),
		)
	}

	return results, nil
}

// MangaIDFromURL extracts the manga ID from a mangadot.net manga URL.
//
// Expected shape: {domain}/manga/{mangaID}
//
// For example, given https://mangadot.net/manga/101, this returns 101.
func MangaIDFromURL(workURL string) (string, error) {
	workURL = strings.TrimSpace(workURL)

	parsed, err := url.Parse(workURL)
	if err != nil {
		return "", fmt.Errorf(
			"invalid mangadot.net manga URL %q: %w",
			workURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	segments := strings.Split(path, "/")

	// Expected: ["", "manga", "{mangaID}"]
	if len(segments) != 3 || segments[1] != "manga" {
		return "", fmt.Errorf(
			"unexpected mangadot.net manga URL: %q",
			workURL,
		)
	}

	mangaID, err := url.PathUnescape(segments[2])
	if err != nil {
		return "", fmt.Errorf(
			"unexpected mangadot.net manga URL: %q",
			workURL,
		)
	}

	mangaID = strings.TrimSpace(mangaID)

	if mangaID == "" {
		return "", fmt.Errorf(
			"unexpected mangadot.net manga URL: %q",
			workURL,
		)
	}

	return mangaID, nil
}
