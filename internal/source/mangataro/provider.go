package mangataro

import (
	"fmt"
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

// MangaTaro exposes one chapter list per language; each language's
// chapter count is shown in the label to help pick a path.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	detail, err := mangataroscraper.GetTitleDetail(p.domain, workURL, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaTaro work details: %w",
			err,
		)
	}

	paths, err := mangataroscraper.GetLanguagePaths(p.domain, detail.ID, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaTaro languages: %w",
			err,
		)
	}

	if len(paths) == 0 {
		// A title without any chapter still needs one
		// automatically selectable path; the entry listing
		// will answer "no chapters" downstream.
		return []common.SelectableItem{
			{
				Label: "MangaTaro",
				Value: "",
			},
		}, nil
	}

	items := make([]common.SelectableItem, 0, len(paths))

	for _, path := range paths {
		items = append(items, common.SelectableItem{
			Label: path.Label,
			Value: path.Code,
		})
	}

	return items, nil
}

// MangaTaro's scan path is the publication language; the listing is
// one chapter list per language, oldest first.
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	detail, err := mangataroscraper.GetTitleDetail(p.domain, workURL, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get MangaTaro work details: %w",
			err,
		)
	}

	chapters, err := mangataroscraper.ListChapters(
		p.domain,
		detail.ID,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to list MangaTaro chapters: %w",
			err,
		)
	}

	entries := make([]sourcetypes.Entry, 0, len(chapters))

	for _, chapter := range chapters {
		entries = append(entries, sourcetypes.Entry{
			Number: chapter.Number,
			Label:  chapter.Label,
			URL:    chapter.URL,
		})
	}

	return sourcetypes.Work{
		Title: detail.Name,
		Kind:  sourcetypes.ItemChapter,
	}, entries, nil
}

// MangaTaro embeds each chapter's page list in its reader-content
// route; the storage host serves the images without any header rules.
func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	urls, err := mangataroscraper.GetPageImageURLs(
		p.domain,
		entryURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaTaro page images: %w",
			err,
		)
	}

	return urls, nil
}
