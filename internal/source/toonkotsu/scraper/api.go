package toonkotsuscraper

import (
	"fmt"
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

	// The site answers a bare "Mozilla/5.0" with 403 on its page
	// blobs and even robots.txt; only a full browser user agent
	// passes everywhere the provider reads.
	BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// parseBaseDomain validates a custom or default domain and returns the
// base URL used to build toonkotsu endpoints.
func parseBaseDomain(domain string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSpace(domain))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse toonkotsu domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid toonkotsu domain: %q",
			domain,
		)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL, nil
}

// callAPI fetches a toonkotsu JSON endpoint. Transient failures are
// retried and the response body is returned as-is for decoding.
func callAPI(
	domain string,
	name string,
	apiPath string,
	query url.Values,
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
			"Fetching toonkotsu %s: %s\n",
			name,
			endpoint.String(),
		)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create toonkotsu %s request: %w",
			name,
			err,
		)
	}

	// The site sits behind a hotlink rule that rejects a bare
	// "Mozilla/5.0"; its JSON routes answer a full browser user
	// agent with no further ceremony.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", BrowserUserAgent)

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch toonkotsu %s: %w",
			name,
			err,
		)
	}

	if log != nil {
		log.Debug(
			"toonkotsu %s response status: %d\n",
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
			"toonkotsu %s returned status %d: %s",
			name,
			result.StatusCode,
			excerpt,
		)
	}

	if len(result.Body) > maxResponseBytes {
		return nil, fmt.Errorf(
			"toonkotsu %s response too large: %d bytes",
			name,
			len(result.Body),
		)
	}

	return result.Body, nil
}
