package mangataroscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// titlePagePathPrefix is the site page path of a work's detail page.
const titlePagePathPrefix = "/manga/"

// MangaDetail is what the title page document yields: the numeric id
// the chapters route wants and the display name the entry listing
// reports.
type MangaDetail struct {
	ID   string
	Name string
}

// GetTitleDetail loads a work's server-rendered title page and pulls
// what the flow needs from it.
//
// The chapters route only accepts the numeric id the site stamps into
// the body element of its own URL pages, so the page — not the slug —
// is the data source here.
func GetTitleDetail(
	domain string,
	workURL string,
	log *logger.Logger,
) (MangaDetail, error) {
	slug, err := MangaSlugFromURL(workURL)
	if err != nil {
		return MangaDetail{}, err
	}

	body, err := fetchTitlePage(domain, slug, log)
	if err != nil {
		return MangaDetail{}, err
	}

	return parseTitlePage(body, slug)
}

func fetchTitlePage(
	domain string,
	slug string,
	log *logger.Logger,
) ([]byte, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return nil, err
	}

	endpoint := *baseURL
	endpoint.Path += titlePagePathPrefix + slug

	if log != nil {
		log.Debug(
			"Fetching MangaTaro title page: %s\n",
			endpoint.String(),
		)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaTaro title page request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaTaro title page: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		excerpt := strings.TrimSpace(string(result.Body))

		if len(excerpt) > maxErrorBodyBytes {
			excerpt = excerpt[:maxErrorBodyBytes]
		}

		return nil, fmt.Errorf(
			"MangaTaro title page returned status %d: %s",
			result.StatusCode,
			excerpt,
		)
	}

	if len(result.Body) > maxResponseBytes {
		return nil, fmt.Errorf(
			"MangaTaro title page response too large: %d bytes",
			len(result.Body),
		)
	}

	return result.Body, nil
}

// parseTitlePage reads the SSR markers from the title page document.
// The numeric id is an attribute of the body element; the display
// name is the first heading.
func parseTitlePage(
	pageHTML []byte,
	slug string,
) (MangaDetail, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(pageHTML))
	if err != nil {
		return MangaDetail{}, fmt.Errorf(
			"failed to parse MangaTaro title page: %w",
			err,
		)
	}

	mangaID := strings.TrimSpace(doc.Find("body").AttrOr("data-manga-id", ""))
	if mangaID == "" {
		return MangaDetail{}, fmt.Errorf(
			"MangaTaro title page %s carries no manga id",
			slug,
		)
	}

	name := strings.TrimSpace(doc.Find("h1").First().Text())
	if name == "" {
		return MangaDetail{}, fmt.Errorf(
			"MangaTaro title page %s carries no name",
			slug,
		)
	}

	return MangaDetail{
		ID:   mangaID,
		Name: name,
	}, nil
}
