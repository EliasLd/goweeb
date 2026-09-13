package mangafreak

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangafreakscraper "github.com/EliasLd/goweeb/internal/source/mangafreak/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
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

// MangaFreak has a single chapter list and no scan/version selection.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return []common.SelectableItem{
		{
			Label: "MangaFreak",
			Value: workURL,
		},
	}, nil
}

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	scanInfo, err := mangafreakscraper.GetScanInfo(workURL, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get scan info: %w",
			err,
		)
	}

	entries := make([]sourcetypes.Entry, 0, len(scanInfo.Chapters))

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

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	imageURLs, err := mangafreakscraper.GetPageImageURLs(
		entryURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaFreak page images: %w",
			err,
		)
	}

	return imageURLs, nil
}
