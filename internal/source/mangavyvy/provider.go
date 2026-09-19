package mangavyvy

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangavyvyscraper "github.com/EliasLd/goweeb/internal/source/mangavyvy/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://mangavyvy.com"

type Provider struct {
	domain string
}

func New(customDomain string) *Provider {
	domain := customDomain

	if domain == "" {
		domain = defaultDomain
	}

	return &Provider{
		domain: strings.TrimSuffix(domain, "/"),
	}
}

func (p *Provider) Name() string {
	return "mangavyvy"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangavyvyscraper.SearchCatalog(
		p.domain,
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	out := make(
		[]sourcetypes.SearchResult,
		0,
		len(results),
	)

	for _, result := range results {
		out = append(
			out,
			sourcetypes.SearchResult{
				Title: result.Title,
				URL:   result.URL,
			},
		)
	}

	return out, nil
}

// Mangavyvy currently exposes a single chapter list per manga.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return []common.SelectableItem{
		{
			Label: "Mangavyvy",
			Value: workURL,
		},
	}, nil
}

//TODO:
// - List Chapters
// - Download Images

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, fmt.Errorf(
		"Mangavyvy chapter listing is not implemented yet",
	)
}

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, fmt.Errorf(
		"Mangavyvy page extraction is not implemented yet",
	)
}
