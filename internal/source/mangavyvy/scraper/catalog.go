package mangavyvyscraper

import (
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
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
			EndpointPath:  "/search",
			QueryParam:    "q",
			CardSelector:  ".book-list .comic-item",
			LinkSelector:  `a[href^="/manga/"]`,
			TitleSelector: ".comic-title",
		},
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	results := make(
		[]MangaResult,
		0,
		len(items),
	)

	for _, item := range items {
		results = append(
			results,
			MangaResult{
				Title: item.Label,
				URL:   item.Value,
			},
		)
	}

	return results, nil
}
