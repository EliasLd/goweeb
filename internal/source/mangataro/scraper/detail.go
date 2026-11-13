package mangataroscraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// titlePagePathPrefix is the site page path of a work's detail page.
const titlePagePathPrefix = "/manga/"

// MangaDetail is what the title page document yields: the numeric id
// the chapters route wants, the display name the entry listing
// reports, and the site metadata published around the work.
type MangaDetail struct {
	ID       string
	Name     string
	Metadata sourcetypes.Metadata
}

// GetTitleDetail loads a work's server-rendered title page and pulls
// what the flow needs from it.
//
// The chapters route only accepts the numeric id the site stamps into
// the body element of its own URL pages, so the page — not the slug —
// is the data source here. The same document answers the work's
// metadata: author, cover, description and tags.
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
// name is the first heading; the rest is the metadata the page
// publishes around the work (best effort, never a failure).
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
		ID:       mangaID,
		Name:     name,
		Metadata: parseTitleMetadata(doc),
	}, nil
}

// ldJSONComicSeries mirrors the shape of the schema.org document the
// title page embeds: the ComicSeries one carries the author line.
type ldJSONComicSeries struct {
	Type   string `json:"@type"`
	Author *struct {
		Name string `json:"name"`
	} `json:"author"`
}

// parseTitleMetadata collects the work's author, cover, description
// and tags from the title page document. Every part degrades to empty
// when the page stops carrying one; metadata is a free win, never a
// reason to fail.
func parseTitleMetadata(doc *goquery.Document) sourcetypes.Metadata {
	var metadata sourcetypes.Metadata

	var tags []string

	doc.Find("script[type=\"application/ld+json\"]").
		Each(func(_ int, script *goquery.Selection) {
			if metadata.Author == "" {
				metadata.Author = authorFromLDJSON(script.Text())
			}
		})

	// The full description is rendered in paragraphs; the site's
	// schema.org copy is the same text truncated.
	metadata.Description = plainDescription(
		doc.Find("#description-content-tab"),
	)

	if content, ok := doc.Find(`meta[property="og:image"]`).Attr("content"); ok {
		metadata.CoverURL = strings.TrimSpace(content)
	}

	seen := make(map[string]bool)

	doc.Find(`a[href*="/tag/"]`).Each(func(_ int, tag *goquery.Selection) {
		label := strings.TrimSpace(tag.Text())

		// The page renders each tag once per layout; the first
		// occurrence keeps the order.
		if label == "" || seen[label] {
			return
		}

		seen[label] = true
		tags = append(tags, label)
	})

	metadata.Tags = tags

	return metadata
}

// authorFromLDJSON decodes one schema.org document and returns its
// author when it is the ComicSeries one. Anything else — another
// document shape, a truncated script — is no data.
func authorFromLDJSON(raw string) string {
	var document ldJSONComicSeries
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return ""
	}

	if document.Type != "ComicSeries" || document.Author == nil {
		return ""
	}

	return strings.TrimSpace(document.Author.Name)
}

// plainDescription joins the description paragraphs the site renders,
// keeping their shape. goquery's plain Text() concatenates children
// without separators, so paragraphs are joined explicitly.
func plainDescription(container *goquery.Selection) string {
	if container.Length() == 0 {
		return ""
	}

	var lines []string

	paragraphs := container.Find("p")

	if paragraphs.Length() == 0 {
		if line := strings.TrimSpace(container.Text()); line != "" {
			lines = append(lines, line)
		}
	} else {
		paragraphs.Each(func(_ int, paragraph *goquery.Selection) {
			line := strings.TrimSpace(paragraph.Text())

			if line != "" {
				lines = append(lines, line)
			}
		})
	}

	return strings.Join(lines, "\n")
}
