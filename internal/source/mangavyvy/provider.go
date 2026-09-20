package mangavyvy

import (
	"net/url"
	"path"
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

func mangaTitleFromURL(
	workURL string,
) string {
	parsed, err := url.Parse(workURL)
	if err != nil {
		return "manga"
	}

	slug := path.Base(
		strings.TrimSuffix(parsed.Path, "/"),
	)

	if slug == "" ||
		slug == "." ||
		slug == "/" {
		return "manga"
	}

	return strings.ReplaceAll(
		slug,
		"-",
		" ",
	)
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
	chapters, err := mangavyvyscraper.ListChapters(
		workURL,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	entries := make(
		[]sourcetypes.Entry,
		0,
		len(chapters),
	)

	for _, chapter := range chapters {
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
		Title: mangaTitleFromURL(workURL),
		Kind:  sourcetypes.ItemChapter,
	}, entries, nil
}

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return mangavyvyscraper.GetPageImageURLs(
		entryURL,
		p.domain,
		log,
	)
}
