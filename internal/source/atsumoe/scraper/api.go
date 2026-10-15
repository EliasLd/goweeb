package atsumoescraper

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

const maxErrorBodyBytes = 4096

// MangaIDFromURL extracts the manga ID from an Atsu.moe manga URL.
//
// Expected shape: {domain}/manga/{mangaID}
//
// For example, given https://atsu.moe/manga/jIg3F, this returns jIg3F.
func MangaIDFromURL(workURL string) (string, error) {
	workURL = strings.TrimSpace(workURL)

	parsed, err := url.Parse(workURL)
	if err != nil {
		return "", fmt.Errorf(
			"invalid Atsu.moe manga URL %q: %w",
			workURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	segments := strings.Split(path, "/")

	// Expected: ["", "manga", "{mangaID}"]
	if len(segments) != 3 || segments[1] != "manga" {
		return "", fmt.Errorf(
			"unexpected Atsu.moe manga URL: %q",
			workURL,
		)
	}

	mangaID, err := url.PathUnescape(segments[2])
	if err != nil {
		return "", fmt.Errorf(
			"unexpected Atsu.moe manga URL: %q",
			workURL,
		)
	}

	mangaID = strings.TrimSpace(mangaID)

	if mangaID == "" {
		return "", fmt.Errorf(
			"unexpected Atsu.moe manga URL: %q",
			workURL,
		)
	}

	return mangaID, nil
}

// parseBaseDomain validates a custom or default domain and returns the
// base URL used to build Atsu.moe API endpoints.
func parseBaseDomain(domain string) (*url.URL, error) {
	baseURL, err := url.Parse(strings.TrimSpace(domain))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse Atsu.moe domain: %w",
			err,
		)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf(
			"invalid Atsu.moe domain: %q",
			domain,
		)
	}

	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	return baseURL, nil
}

// apiGet fetches an Atsu.moe JSON API endpoint. Transient failures are
// retried and the response body is returned as-is for decoding.
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
			"Fetching Atsu.moe %s: %s\n",
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
			"failed to create Atsu.moe %s request: %w",
			name,
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch Atsu.moe %s: %w",
			name,
			err,
		)
	}

	if log != nil {
		log.Debug(
			"Atsu.moe %s response status: %d\n",
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
			"Atsu.moe %s returned status %d: %s",
			name,
			result.StatusCode,
			excerpt,
		)
	}

	if len(result.Body) > maxResponseBytes {
		return nil, fmt.Errorf(
			"Atsu.moe %s response too large: %d bytes",
			name,
			len(result.Body),
		)
	}

	return result.Body, nil
}
