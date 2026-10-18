package animesamascraper

import (
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

type MangaResult struct {
	Title string
	URL   string
}

// Builds the catalogue search configuration for a given domain. The
// site was redesigned in 2026 and now renders its result cards as
// "as-carte" elements; legacy selects the pre-redesign markup.
func catalogSearchConfig(
	domain string,
	legacy bool,
) common.CatalogSearchConfig {
	cardSelector := "div.as-carte"
	titleSelector := ".as-carte-titre"

	if legacy {
		cardSelector = "div.catalog-card"
		titleSelector = ".card-title"
	}

	return common.CatalogSearchConfig{
		Domain:       domain,
		EndpointPath: "/catalogue/",
		QueryParam:   "search",
		ExtraParams: map[string]string{
			"type[]": "Scans",
		},
		CardSelector:  cardSelector,
		LinkSelector:  "a",
		TitleSelector: titleSelector,
	}
}

// Searches anime-sama catalog and returns the manga page URL
func SearchCatalog(domain, query string, log *logger.Logger) ([]MangaResult, error) {
	items, err := common.SearchHTMLCatalog(
		catalogSearchConfig(domain, false),
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		// Mirrors or lagging deployments may still serve the
		// pre-redesign markup; retry the legacy selectors
		// before giving up.
		log.Debug(
			"No redesigned cards, retrying legacy selectors\n",
		)

		items, err = common.SearchHTMLCatalog(
			catalogSearchConfig(domain, true),
			query,
			log,
		)
		if err != nil {
			return nil, err
		}
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
