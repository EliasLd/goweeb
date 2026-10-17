package mangadotnetscraper

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
	maxErrorBodyBytes = 4096
	maxResponseBytes  = 8 << 20
)

// parseBaseDomain validates a custom or default domain and returns the
// base URL used to build mangadot.net API endpoints.
func parseBaseDomain(domain string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSpace(domain))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse mangadot.net domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid mangadot.net domain: %q",
			domain,
		)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL, nil
}

// apiGet fetches a mangadot.net JSON API endpoint. Transient failures are
// retried and the response body is returned as-is for decoding.
//
// MangaDot sits behind a Cloudflare WAF that serves a managed challenge
// for browser navigation requests. The JSON API routes accept plain GET
// requests as long as they look like same-origin fetches, hence the
// Sec-Fetch-Site header. No cookies, sessions or challenge solving are
// involved.
func apiGet(
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
			"Fetching mangadot.net %s: %s\n",
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
			"failed to create mangadot.net %s request: %w",
			name,
			err,
		)
	}

	// Satisfies the Cloudflare WAF rule for /api/* JSON routes
	// (verified: without it the API returns a 403 challenge page).
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch mangadot.net %s: %w",
			name,
			err,
		)
	}

	if log != nil {
		log.Debug(
			"mangadot.net %s response status: %d\n",
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
			"mangadot.net %s returned status %d: %s",
			name,
			result.StatusCode,
			excerpt,
		)
	}

	if len(result.Body) > maxResponseBytes {
		return nil, fmt.Errorf(
			"mangadot.net %s response too large: %d bytes",
			name,
			len(result.Body),
		)
	}

	return result.Body, nil
}
