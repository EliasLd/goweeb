package animesama

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/EliasLd/goweeb/internal/fetch"
	"github.com/EliasLd/goweeb/internal/logger"
	animesamascraper "github.com/EliasLd/goweeb/internal/source/animesama/scraper"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://anime-sama.to"

type Provider struct {
	domain string
}

func New(customDomain string) *Provider {
	domain := customDomain

	if domain == "" {
		domain = defaultDomain
	}

	return &Provider{domain: strings.TrimSuffix(domain, "/")}
}

func (p *Provider) Name() string { return "animesama" }

func (p *Provider) Search(query string, log *logger.Logger) ([]sourcetypes.SearchResult, error) {
	results, err := animesamascraper.SearchCatalog(p.domain, query, log)
	if err != nil {
		return nil, err
	}

	out := make([]sourcetypes.SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, sourcetypes.SearchResult{Title: r.Title, URL: r.URL})
	}
	return out, nil
}

func (p *Provider) ListScanPaths(workURL string, log *logger.Logger) ([]common.SelectableItem, error) {
	paths, err := animesamascraper.GetAllScanPaths(workURL, log)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan paths: %w", err)
	}

	items := make([]common.SelectableItem, 0, len(paths))
	for _, sp := range paths {
		items = append(items, common.SelectableItem{
			Label: sp.Label,
			Value: sp.Path,
		})
	}

	return items, nil
}

func (p *Provider) ListEntries(workURL string, scanPath string, log *logger.Logger) (sourcetypes.Work, []sourcetypes.Entry, error) {
	if strings.TrimSpace(scanPath) == "" {
		return sourcetypes.Work{}, nil, fmt.Errorf("scan path is required")
	}

	mangaBase, err := url.Parse(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf("invalid work URL: %w", err)
	}
	if !strings.HasSuffix(mangaBase.Path, "/") {
		mangaBase.Path += "/"
	}

	scanRef, err := url.Parse(scanPath)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf("invalid scan path: %w", err)
	}

	scanPageURL := mangaBase.ResolveReference(scanRef).String()

	mangaName, err := animesamascraper.ExtractMangaName(scanPageURL, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf("failed to extract manga name: %w", err)
	}

	scanInfo, err := animesamascraper.GetScanInfo(p.domain, mangaName, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf("failed to get scan info: %w", err)
	}

	entries := make([]sourcetypes.Entry, 0, len(scanInfo.Chapters))
	for _, ch := range scanInfo.Chapters {
		entries = append(entries, sourcetypes.Entry{
			Number: ch,
			Label:  fmt.Sprintf("Chapter %d", ch),
			URL:    fmt.Sprintf("%s/s2/scans/%s/%d", p.domain, scanInfo.MangaName, ch),
		})
	}

	return sourcetypes.Work{
		Title: mangaName,
		Kind:  sourcetypes.ItemChapter,
	}, entries, nil
}

func (p *Provider) GetPageImageURLs(entryURL string, log *logger.Logger) ([]string, error) {
	chapterURL := strings.TrimSuffix(entryURL, "/")
	return fetch.CollectSequentialJPGURLs(chapterURL, log)
}
