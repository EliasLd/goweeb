package common

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/PuerkitoBio/goquery"
)

type CatalogSearchConfig struct {
	Domain        string
	EndpointPath  string // e.g. "/catalogue/" or "/Find/"
	QueryParam    string // e.g. "search"; empty means query is appended to path
	ExtraParams   map[string]string
	CardSelector  string // e.g. "div.catalog-card"
	LinkSelector  string // e.g. "a"; empty means the card itself is the link
	TitleSelector string // e.g. ".card-title"
}

func SearchHTMLCatalog(
	cfg CatalogSearchConfig,
	query string,
	log *logger.Logger,
) ([]SelectableItem, error) {
	base := strings.TrimSuffix(cfg.Domain, "/")
	path := "/" + strings.TrimPrefix(cfg.EndpointPath, "/")

	var searchURL string

	if cfg.QueryParam == "" {
		// Path-based search:
		path = strings.TrimSuffix(path, "/") + "/" + url.PathEscape(query)

		u, err := url.Parse(base + path)
		if err != nil {
			return nil, fmt.Errorf("failed to build search URL: %w", err)
		}

		searchURL = u.String()
	} else {
		// Query parameter search:
		u, err := url.Parse(base + path)
		if err != nil {
			return nil, fmt.Errorf("failed to build search URL: %w", err)
		}

		q := u.Query()

		for k, v := range cfg.ExtraParams {
			q.Set(k, v)
		}

		q.Set(cfg.QueryParam, query)
		u.RawQuery = q.Encode()

		searchURL = u.String()
	}

	log.Debug("Searching catalog: %s\n", searchURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Catalog response status: %d\n", resp.StatusCode)
	log.Debug("Catalog response headers: %v\n", resp.Header)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse catalog page: %w", err)
	}

	var results []SelectableItem

	doc.Find(cfg.CardSelector).Each(func(i int, card *goquery.Selection) {
		var link *goquery.Selection

		if cfg.LinkSelector == "" {
			link = card
		} else {
			link = card.Find(cfg.LinkSelector).First()
		}

		href, exists := link.Attr("href")
		if !exists {
			return
		}

		title := strings.TrimSpace(card.Find(cfg.TitleSelector).Text())

		if title == "" {
			title = strings.TrimSpace(link.Text())
		}

		if title == "" {
			title = href
		}

		if strings.HasPrefix(href, "/") {
			href = base + href
		}

		results = append(results, SelectableItem{
			Label: title,
			Value: href,
		})
	})

	return results, nil
}
