package mangadex

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangadexscraper "github.com/EliasLd/goweeb/internal/source/mangadex/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://api.mangadex.org"

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
	return "mangadex"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangadexscraper.SearchCatalog(
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

func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	languages, err := mangadexscraper.GetAvailableLanguages(
		workURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaDex available languages: %w",
			err,
		)
	}

	items := make(
		[]common.SelectableItem,
		0,
		len(languages),
	)

	for _, language := range languages {
		items = append(
			items,
			common.SelectableItem{
				Label: languageLabel(language),
				Value: language,
			},
		)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})

	return items, nil
}

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (
	sourcetypes.Work,
	[]sourcetypes.Entry,
	error,
) {
	scanInfo, err := mangadexscraper.GetScanInfo(
		p.domain,
		workURL,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get MangaDex scan info: %w",
			err,
		)
	}

	entries := make(
		[]sourcetypes.Entry,
		0,
		len(scanInfo.Chapters),
	)

	for _, chapter := range scanInfo.Chapters {
		entries = append(
			entries,
			sourcetypes.Entry{
				Number: chapter.Number,
				Label:  chapter.Label,
				URL:    chapter.URL,
			},
		)
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
	imageURLs, err := mangadexscraper.GetPageImageURLs(
		p.domain,
		entryURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaDex page images: %w",
			err,
		)
	}

	return imageURLs, nil
}

func (p *Provider) ImageRequestHeaders() map[string]string {
	return map[string]string{
		"User-Agent": "goweeb (+https://github.com/EliasLd/goweeb)",
		"Accept":     "*/*",
	}
}
