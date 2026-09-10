package mangafreak

import (
	"fmt"
	"strings"

	"github.com/EliasLd/scan-scraper/internal/logger"
	"github.com/EliasLd/scan-scraper/internal/source/common"
	mangafreakscraper "github.com/EliasLd/scan-scraper/internal/source/mangafreak/scraper"
	sourcetypes "github.com/EliasLd/scan-scraper/internal/source/types"
)

const defaultDomain = "https://ww3.mangafreak.me"

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
	return "mangafreak"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangafreakscraper.SearchCatalog(
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

// TODO: Implement this
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, fmt.Errorf(
		"mangafreak: ListScanPaths not implemented yet",
	)
}

// TODO: Implement this
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, fmt.Errorf(
		"mangafreak: ListEntries not implemented yet",
	)
}

// TODO: Implement this
func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, fmt.Errorf(
		"mangafreak: GetPageImageURLs not implemented yet",
	)
}
