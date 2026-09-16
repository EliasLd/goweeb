package weebcentral

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	weebcentralscraper "github.com/EliasLd/goweeb/internal/source/weebcentral/scraper"
)

const defaultDomain = "https://weebcentral.com"

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
	return "weebcentral"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := weebcentralscraper.SearchCatalog(
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

// WeebCentral doesn't implement different scan versions.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return []common.SelectableItem{
		{
			Label: "WeebCentral",
			Value: workURL,
		},
	}, nil
}

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	scanInfo, err := weebcentralscraper.GetScanInfo(
		p.domain,
		workURL,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get WeebCentral scan info: %w",
			err,
		)
	}

	entries := make(
		[]sourcetypes.Entry,
		0,
		len(scanInfo.Chapters),
	)

	for _, chapter := range scanInfo.Chapters {
		entries = append(entries, sourcetypes.Entry{
			Number: chapter.Number,
			Label:  chapter.Label,
			URL:    chapter.URL,
		})
	}

	return sourcetypes.Work{
		Title: scanInfo.MangaName,
		Kind:  sourcetypes.ItemChapter,
	}, entries, nil
}

//TODO: Next step is to implement and test each
// of the following functions one by one.

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, fmt.Errorf(
		"weebcentral: GetPageImageURLs not implemented yet",
	)
}
