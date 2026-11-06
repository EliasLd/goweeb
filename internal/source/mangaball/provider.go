package mangaball

import (
	"fmt"
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

// MangaBall exposes one chapter list per language; each language's
// chapter count is shown in the label to help pick a path.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	titleID, err := mangaballscraper.TitleIDFromURL(workURL)
	if err != nil {
		return nil, err
	}

	paths, err := mangaballscraper.GetLanguagePaths(
		p.domain,
		titleID,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaBall languages: %w",
			err,
		)
	}

	if len(paths) == 0 {
		// A title without any chapter still needs one
		// automatically selectable path; the entry listing
		// will answer "no chapters" downstream.
		return []common.SelectableItem{
			{
				Label: "MangaBall",
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

// MangaBall's scan path is the publication language; the listing is
// one chapter list per language, oldest first.
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	titleID, err := mangaballscraper.TitleIDFromURL(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	title, err := mangaballscraper.GetTitleName(p.domain, titleID, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get MangaBall title: %w",
			err,
		)
	}

	chapters, err := mangaballscraper.ListChapters(
		p.domain,
		titleID,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to list MangaBall chapters: %w",
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

// MangaBall embeds each chapter's page list in its server-rendered
// reader page. The storage CDNs only accept downloads carrying the
// site's origin as referer, so it is added here.
func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	urls, err := mangaballscraper.GetPageImageURLs(
		entryURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get MangaBall page images: %w",
			err,
		)
	}

	return urls, nil
}

// ImageRequestHeaders carries what the storage CDNs demand: the
// site's origin as referer, and a user agent they accept — their bot
// rules reject the downloader's bare "Mozilla/5.0" and empty UAs,
// while a browser one and goweeb's own pass.
func (p *Provider) ImageRequestHeaders() map[string]string {
	return map[string]string{
		"User-Agent": "goweeb (+https://github.com/EliasLd/goweeb)",
		"Referer":    p.domain + "/",
	}
}
