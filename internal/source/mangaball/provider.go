package mangaball

import (
	"errors"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangaballscraper "github.com/EliasLd/goweeb/internal/source/mangaball/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://mangaball.com"

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
	return "mangaball"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangaballscraper.SearchCatalog(
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

// TODO: Implement language selection.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, errors.New(
		"MangaBall scan path selection is not implemented yet",
	)
}

// TODO: Implement chapter listing.
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, errors.New(
		"MangaBall chapter listing is not implemented yet",
	)
}

// TODO: Implement page image retrieval.
func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, errors.New(
		"MangaBall page image retrieval is not implemented yet",
	)
}
