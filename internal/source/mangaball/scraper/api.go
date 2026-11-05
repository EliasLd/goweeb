package mangaballscraper

import (
	"bytes"
	"encoding/json"
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
)

// parseBaseDomain validates a custom or default domain and returns the
// base URL used to build MangaBall endpoints.
func parseBaseDomain(domain string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSpace(domain))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse MangaBall domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid MangaBall domain: %q",
			domain,
		)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL, nil
}

// callAPI fetches a MangaBall API endpoint. Transient failures are
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
			"Fetching MangaBall %s: %s\n",
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
				"failed to encode MangaBall %s request: %w",
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
			"failed to create MangaBall %s request: %w",
			name,
			err,
		)
	}

	// The API works with a plain browser user agent; the site sits
	// behind Cloudflare without a challenge for these routes.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaBall %s: %w",
			name,
			err,
		)
	}

	if log != nil {
		log.Debug(
			"MangaBall %s response status: %d\n",
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
			"MangaBall %s returned status %d: %s",
			name,
			result.StatusCode,
			excerpt,
		)
	}

	if len(result.Body) > maxResponseBytes {
		return nil, fmt.Errorf(
			"MangaBall %s response too large: %d bytes",
			name,
			len(result.Body),
		)
	}

	return result.Body, nil
}

// apiPost fetches a MangaBall JSON API endpoint with a JSON body.
func apiPost(
	domain string,
	name string,
	apiPath string,
	payload any,
	log *logger.Logger,
) ([]byte, error) {
	return callAPI(
		domain,
		name,
		http.MethodPost,
		apiPath,
		nil,
		payload,
		log,
	)
}

// apiGet fetches a MangaBall JSON API endpoint with query parameters.
func apiGet(
	domain string,
	name string,
	apiPath string,
	query url.Values,
	log *logger.Logger,
) ([]byte, error) {
	return callAPI(
		domain,
		name,
		http.MethodGet,
		apiPath,
		query,
		nil,
		log,
	)
}
