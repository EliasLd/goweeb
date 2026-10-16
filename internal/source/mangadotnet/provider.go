package mangadotnet

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangadotnetscraper "github.com/EliasLd/goweeb/internal/source/mangadotnet/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://mangadot.net"

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
	return "mangadotnet"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangadotnetscraper.SearchCatalog(
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
			URL:   p.mangaURL(result.ID),
		})
	}

	return out, nil
}

// mangaURL builds the canonical manga page URL used as work URL by the
// following provider steps: {domain}/manga/{mangaID}.
func (p *Provider) mangaURL(mangaID string) string {
	return p.domain + "/manga/" + mangaID
}

// MangaDot exposes one chapter list per upload group. Group filter
// options are either numeric upload ids or "scanlator:" pseudo-ids and
// are forwarded verbatim to the chapter listing step.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	mangaID, err := mangadotnetscraper.MangaIDFromURL(workURL)
	if err != nil {
		return nil, err
	}

	scanPaths, err := mangadotnetscraper.ListScanPaths(
		p.domain,
		mangaID,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get mangadot.net scan paths: %w",
			err,
		)
	}

	items := make([]common.SelectableItem, 0, len(scanPaths))

	for _, scanPath := range scanPaths {
		items = append(items, common.SelectableItem{
			Label: scanPath.Label,
			Value: scanPath.Value,
		})
	}

	return items, nil
}

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	mangaID, err := mangadotnetscraper.MangaIDFromURL(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	title, err := mangadotnetscraper.GetMangaTitle(p.domain, mangaID, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get mangadot.net manga title: %w",
			err,
		)
	}

	chapters, err := mangadotnetscraper.ListChapters(
		p.domain,
		mangaID,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to list mangadot.net chapters: %w",
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
	imageURLs, err := mangadotnetscraper.GetPageImageURLs(
		entryURL,
		p.domain,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get mangadot.net page images: %w",
			err,
		)
	}

	return imageURLs, nil
}
