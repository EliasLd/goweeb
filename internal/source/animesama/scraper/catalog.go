package animesamascraper

import (
	"github.com/EliasLd/scan-scraper/internal/logger"
	"github.com/EliasLd/scan-scraper/internal/source/common"
)

type MangaResult struct {
	Title string
	URL   string
}

// Searches anime-sama catalog and returns the manga page URL
func SearchCatalog(domain, query string, log *logger.Logger) ([]MangaResult, error) {
	items, err := common.SearchHTMLCatalog(common.CatalogSearchConfig{
		Domain:       domain,
		EndpointPath: "/catalogue/",
		QueryParam:   "search",
		ExtraParams: map[string]string{
			"type[]": "Scans",
		},
		CardSelector:  "div.catalog-card",
		LinkSelector:  "a",
		TitleSelector: ".card-title",
	}, query, log)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		log.Warn("No manga found in catalog\n")
		return nil, nil
	}

	results := make([]MangaResult, 0, len(items))
	for _, it := range items {
		results = append(results, MangaResult{
			Title: it.Label,
			URL:   it.Value,
		})
	}

	return results, nil
}

func PromptUserToSelectManga(results []MangaResult, log *logger.Logger) (string, error) {
	items := make([]common.SelectableItem, 0, len(results))
	for _, r := range results {
		items = append(items, common.SelectableItem{
			Label: r.Title,
			Value: r.URL,
		})
	}

	return common.PromptUserToSelect(
		items,
		"Multiple results found:",
		"Select a manga (1-%d) or 0 to cancel: ",
		log,
	)
}
