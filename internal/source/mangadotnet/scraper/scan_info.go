package mangadotnetscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	mangaInfoPath         = "/api/manga"
	scanlatorFilterPrefix = "scanlator:"
)

type ScanPath struct {
	Label string
	Value string
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
		})
	}

	if len(items) == 0 {
		// Some manga expose no group filter option. Keep a single
		// automatically selectable path covering every chapter.
		return []ScanPath{
			{
				Label: "MangaDot",
				Value: "",
			},
		}, nil
	}

	if log != nil {
		log.Debug(
			"Found %d mangadot.net scan path(s)\n",
			len(items),
		)
	}

	return items, nil
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
