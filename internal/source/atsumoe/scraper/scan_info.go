package atsumoescraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	mangaInfoPath = "/api/manga/info"
	mangaPagePath = "/api/manga/page"
)

type Scanlator struct {
	ID   string
	Name string
}

type mangaPageResponse struct {
	MangaPage struct {
		Scanlators []scanlatorPayload `json:"scanlators"`
	} `json:"mangaPage"`
}

type scanlatorPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type mangaInfoResponse struct {
	Title string `json:"title"`
}

// GetScanlators lists the scanlation groups providing chapters for a
// manga, as displayed on the public manga page.
func GetScanlators(
	domain string,
	mangaID string,
	log *logger.Logger,
) ([]Scanlator, error) {
	query := url.Values{}
	query.Set("id", mangaID)

	body, err := apiGet(domain, "manga page", mangaPagePath, query, log)
	if err != nil {
		return nil, err
	}

	var payload mangaPageResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode Atsu.moe manga page: %w",
			err,
		)
	}

	scanlators := make([]Scanlator, 0, len(payload.MangaPage.Scanlators))

	for _, raw := range payload.MangaPage.Scanlators {
		id := strings.TrimSpace(raw.ID)

		if id == "" {
			continue
		}

		name := strings.TrimSpace(raw.Name)

		if name == "" {
			name = id
		}

		scanlators = append(scanlators, Scanlator{
			ID:   id,
			Name: name,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d Atsu.moe scanlator(s)\n",
			len(scanlators),
		)
	}

	return scanlators, nil
}

// GetMangaTitle returns the canonical title of a manga.
func GetMangaTitle(
	domain string,
	mangaID string,
	log *logger.Logger,
) (string, error) {
	query := url.Values{}
	query.Set("mangaId", mangaID)

	body, err := apiGet(domain, "manga info", mangaInfoPath, query, log)
	if err != nil {
		return "", err
	}

	var payload mangaInfoResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf(
			"failed to decode Atsu.moe manga info: %w",
			err,
		)
	}

	title := strings.TrimSpace(payload.Title)

	if title == "" {
		return "", fmt.Errorf(
			"Atsu.moe manga info has no title: %s",
			mangaID,
		)
	}

	return title, nil
}
