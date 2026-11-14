package toonkotsu

import (
	"errors"
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	toonkotsuscraper "github.com/EliasLd/goweeb/internal/source/toonkotsu/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://toonkotsu.com"

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
	return "toonkotsu"
}

// Search walks the catalogue's browse route: relevance-ordered rows
// carrying the work URL and cover.
func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := toonkotsuscraper.SearchCatalog(
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

// Toonkotsu splits a work into its main series and its color
// editions; each listing is one selectable path, the chapter count in
// the label. The value is the edition code, empty for the main
// listing.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	items, err := toonkotsuscraper.GetScanPaths(
		p.domain,
		workURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get toonkotsu scan paths: %w",
			err,
		)
	}

	return items, nil
}

// Toonkotsu addresses the reader by chapter id; every upload of a
// number (the rows carrying it and their alternates) travels as
// comma-joined candidate ids. The scan path names an edition's
// listing when set.
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	code, err := toonkotsuscraper.WorkCodeFromURL(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	title, chapters, err := toonkotsuscraper.ListChapters(
		p.domain,
		code,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to list toonkotsu chapters: %w",
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
		Title: title,
		Kind:  sourcetypes.ItemChapter,
	}, entries, nil
}

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, errors.New(
		"toonkotsu page image retrieval is not implemented yet",
	)
}
