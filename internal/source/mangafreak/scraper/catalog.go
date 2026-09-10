package mangafreakscraper

import (
	"github.com/EliasLd/scan-scraper/internal/logger"
	"github.com/EliasLd/scan-scraper/internal/source/common"
)

type MangaResult struct {
	Title string
	URL   string
}

func SearchCatalog(
	domain string,
	query string,
	log *logger.Logger,
) ([]MangaResult, error) {
	items, err := common.SearchHTMLCatalog(
		common.CatalogSearchConfig{
			Domain:        domain,
			EndpointPath:  "/Find/",
			QueryParam:    "",
			CardSelector:  "div.manga_search_item",
			LinkSelector:  "h3 a",
			TitleSelector: "h3 a",
		},
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		log.Warn("No manga found in catalog\n")
		return nil, nil
	}

	results := make([]MangaResult, 0, len(items))

	for _, item := range items {
		results = append(results, MangaResult{
			Title: item.Label,
			URL:   item.Value,
		})
	}

	log.Info("Found %d result(s)\n", len(results))

	return results, nil
}
