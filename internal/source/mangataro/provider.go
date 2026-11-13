package mangataro

import (
	"errors"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangataroscraper "github.com/EliasLd/goweeb/internal/source/mangataro/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://mangataro.org"

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
	return "mangataro"
}

// SearchCatalog walks the catalog REST route the site's own frontend
// uses; rows carry the work URL and cover.
func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangataroscraper.SearchCatalog(
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
			Title:    result.Title,
			URL:      result.URL,
			CoverURL: result.CoverURL,
		})
	}

	return out, nil
}

func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, errors.New(
		"mangataro scan selection is not implemented yet",
	)
}

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, errors.New(
		"mangataro chapter listing is not implemented yet",
	)
}

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, errors.New(
		"mangataro page image retrieval is not implemented yet",
	)
}
