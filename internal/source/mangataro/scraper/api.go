package mangataroscraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	maxResponseBytes  = 8 << 20
	maxErrorBodyBytes = 4096

	// The catalog route answers 24 rows per page and accepts no
	// larger page: the site's own frontend walks those small pages
	// one by one.
	catalogPageSize = 24

	// Search screens never need the whole catalog; three pages
	// (72 results) keep the search output in line with the other
	// providers.
	maxCatalogPages = 3

	// The site's frontend serializes its catalog filter arrays as
	// JSON strings; an unused filter is the literal "[]".
	catalogEmptyFilter = "[]"

	// The default ordering of the site's own catalog page.
	catalogSort = "popular"
)

// parseBaseDomain validates a custom or default domain and returns the
// base URL used to build MangaTaro endpoints.
func parseBaseDomain(domain string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSpace(domain))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse MangaTaro domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid MangaTaro domain: %q",
			domain,
		)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL, nil
}

// callAPI fetches a MangaTaro JSON endpoint. Transient failures are
// retried and the response body is returned as-is for decoding.
func callAPI(
	domain string,
	name string,
	method string,
	apiPath string,
	query url.Values,
	payload any,
	log *logger.Logger,
) ([]byte, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return nil, err
	}

	endpoint := *baseURL
	endpoint.Path += apiPath

	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}

	if log != nil {
		log.Debug(
			"Fetching MangaTaro %s: %s\n",
			name,
			endpoint.String(),
		)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	var body []byte

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to encode MangaTaro %s request: %w",
				name,
				err,
			)
		}
	}

	req, err := http.NewRequest(
		method,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaTaro %s request: %w",
			name,
			err,
		)
	}

	// The site serves both the catalog route and its own reads
	// with a plain browser user agent and no challenge on these
	// routes, so nothing further is needed.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaTaro %s: %w",
			name,
			err,
		)
	}

	if log != nil {
		log.Debug(
			"MangaTaro %s response status: %d\n",
			name,
			result.StatusCode,
		)
	}

	if result.StatusCode != http.StatusOK {
		excerpt := strings.TrimSpace(string(result.Body))

		if len(excerpt) > maxErrorBodyBytes {
			excerpt = excerpt[:maxErrorBodyBytes]
		}

		return nil, fmt.Errorf(
			"MangaTaro %s returned status %d: %s",
			name,
			result.StatusCode,
			excerpt,
		)
	}

	if len(result.Body) > maxResponseBytes {
		return nil, fmt.Errorf(
			"MangaTaro %s response too large: %d bytes",
			name,
			len(result.Body),
		)
	}

	return result.Body, nil
}

// unescapeText decodes the HTML entities the payloads embed in their
// text fields (&#039;, &ouml;, …) and trims the surrounding spaces.
func unescapeText(raw string) string {
	return strings.TrimSpace(html.UnescapeString(raw))
}
