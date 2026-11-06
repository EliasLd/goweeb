package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// pagesKeyPattern matches the escaped "pages":[…] JSON blob the
// server-rendered chapter page embeds (there is no public API for
// the page list; the reader page itself is the data source).
var pagesKeyPattern = regexp.MustCompile(
	`(?s)\\"pages\\":(\[.*?\])`,
)

// ChapterIDsFromURL extracts the chapter ids a chapter entry URL
// carries.
//
// Expected shape: {domain}/chapter-detail/{id}[,{id}…] — a single id
// for single-source chapters, comma-joined ids when one chapter
// number was crawled from several sites (see chapterEntryURL).
func ChapterIDsFromURL(entryURL string) ([]string, error) {
	workURL := strings.TrimSpace(entryURL)

	parsed, err := url.Parse(workURL)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid MangaBall chapter URL %q: %w",
			entryURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")

	// Expected: ["", "chapter-detail", "{ids}"]
	segments := strings.Split(path, "/")

	if len(segments) != 3 || segments[1] != "chapter-detail" {
		return nil, fmt.Errorf(
			"unexpected MangaBall chapter URL: %q",
			entryURL,
		)
	}

	raw := segments[2]

	ids := make([]string, 0)

	for _, token := range strings.Split(raw, ",") {
		id := strings.TrimSpace(token)
		if id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf(
			"unexpected MangaBall chapter URL: %q",
			entryURL,
		)
	}

	return ids, nil
}

// GetPageImageURLs resolves the ordered page image URLs of a chapter
// entry.
//
// The chapter page embeds its page list as an escaped JSON payload.
// When the entry URL carries several comma-joined copies of the
// chapter, every copy is inspected and the one with the most pages
// wins — the site re-hosts crawls of the same chapter that differ in
// completeness, so the fullest copy is kept. Copies tie on count keep
// the first, which mirrors the site's own ordering. A copy that fails
// to load or to parse simply loses; the error only surfaces when no
// copy is usable at all.
func GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	ids, err := ChapterIDsFromURL(entryURL)
	if err != nil {
		return nil, err
	}

	parsed, err := url.Parse(strings.TrimSpace(entryURL))
	if err != nil {
		return nil, fmt.Errorf(
			"invalid MangaBall chapter URL %q: %w",
			entryURL,
			err,
		)
	}

	candidate := *parsed

	var (
		bestURLs  []string
		lastError error
	)

	for _, id := range ids {
		candidate.Path = chapterDetailPathPrefix + id

		urls, err := fetchChapterPages(
			candidate.String(),
			id,
			log,
		)
		if err != nil {
			lastError = err
			continue
		}

		if len(urls) > len(bestURLs) {
			bestURLs = urls
		}
	}

	if len(bestURLs) == 0 {
		if lastError != nil {
			return nil, lastError
		}

		return nil, fmt.Errorf(
			"MangaBall chapter %s has no page images",
			strings.Join(ids, ","),
		)
	}

	if log != nil {
		log.Debug(
			"Found %d MangaBall page images\n",
			len(bestURLs),
		)
	}

	return bestURLs, nil
}

// fetchChapterPages loads one chapter copy's page and extracts its
// ordered page image URLs. Absolute http(s) URLs only are kept; the
// site's page lists are absolute on its own storage CDNs.
func fetchChapterPages(
	pageURL string,
	id string,
	log *logger.Logger,
) ([]string, error) {
	if log != nil {
		log.Debug(
			"Fetching MangaBall chapter page: %s\n",
			pageURL,
		)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaBall chapter page request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaBall chapter %s: %w",
			id,
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"MangaBall chapter %s returned status %d",
			id,
			result.StatusCode,
		)
	}

	pages, err := parsePagesBlob(result.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"MangaBall chapter %s: %w",
			id,
			err,
		)
	}

	return pages, nil
}

// parsePagesBlob digs the escaped "pages":[] payload out of the
// chapter HTML and returns the page image URLs in order.
func parsePagesBlob(
	html []byte,
) ([]string, error) {
	match := pagesKeyPattern.FindSubmatch(html)
	if match == nil {
		return nil, fmt.Errorf(
			"chapter page carries no page list",
		)
	}

	// The payload is embedded inside a JavaScript string, so the
	// quotes are escaped; unescape them before decoding.
	unescaped := strings.ReplaceAll(string(match[1]), `\"`, `"`)

	var urls []string
	if err := json.Unmarshal([]byte(unescaped), &urls); err != nil {
		return nil, fmt.Errorf(
			"failed to decode the chapter page list: %w",
			err,
		)
	}

	pageURLs := make([]string, 0, len(urls))

	for _, raw := range urls {
		pageURL := strings.TrimSpace(raw)

		keep := strings.HasPrefix(pageURL, "http://") ||
			strings.HasPrefix(pageURL, "https://")

		if !keep {
			continue
		}

		pageURLs = append(pageURLs, pageURL)
	}

	return pageURLs, nil
}
