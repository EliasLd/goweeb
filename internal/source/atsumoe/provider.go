package atsumoe

import (
	"errors"
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	atsumoescraper "github.com/EliasLd/goweeb/internal/source/atsumoe/scraper"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://atsu.moe"

type Provider struct {
	domain string
}

func New(customDomain string) *Provider {
	domain := strings.TrimSpace(customDomain)

	if domain == "" {
		domain = defaultDomain
	}

	return &Provider{
		domain: strings.TrimSuffix(domain, "/"),
	}
}

func (p *Provider) Name() string {
	return "atsumoe"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := atsumoescraper.SearchCatalog(
		p.domain,
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	out := make([]sourcetypes.SearchResult, 0, len(results))

	for _, result := range results {
		out = append(out, sourcetypes.SearchResult{
			Title: result.Title,
			URL:   result.URL,
		})
	}

	return out, nil
}

// Atsu.moe exposes one chapter list per scanlation group, displayed
// on the public manga page.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	mangaID, err := atsumoescraper.MangaIDFromURL(workURL)
	if err != nil {
		return nil, err
	}

	scanlators, err := atsumoescraper.GetScanlators(p.domain, mangaID, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get Atsu.moe scanlators: %w",
			err,
		)
	}

	if len(scanlators) == 0 {
		// Some manga are published without any scanlation group.
		// Keep a single automatically selectable path covering every
		// chapter of the manga.
		return []common.SelectableItem{
			{
				Label: "Atsu.moe",
				Value: "",
			},
		}, nil
	}

	items := make([]common.SelectableItem, 0, len(scanlators))

	for _, scanlator := range scanlators {
		items = append(items, common.SelectableItem{
			Label: scanlator.Name,
			Value: scanlator.ID,
		})
	}

	return items, nil
}

// TODO: Implement chapter listing.
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, errors.New(
		"Atsu.moe chapter listing is not implemented yet",
	)
}

// TODO: Implement page image retrieval.
func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, errors.New(
		"Atsu.moe page retrieval is not implemented yet",
	)
}
