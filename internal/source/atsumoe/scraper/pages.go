package atsumoescraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const readChapterPath = "/api/read/chapter"

type readChapterResponse struct {
	ReadChapter struct {
		Pages []chapterPage `json:"pages"`
	} `json:"readChapter"`
}

type chapterPage struct {
	Image  string `json:"image"`
	Number int    `json:"number"`
}

// pageImage keeps a parsed page with its number so the returned list
// is always ordered from the first page to the last one.
type pageImage struct {
	Number int
	URL    string
}

// GetPageImageURLs returns the ordered page image URLs of a chapter.
//
// The entry URL is the public read chapter endpoint built when
// listing chapters:
//
//	{domain}/api/read/chapter?mangaId={mangaID}&chapterId={chapterID}
//
// Paths returned by the API are relative to the image CDN, which the
// website exposes at cdn.{domain host}, as mirrored by its og:image
// metadata.
func GetPageImageURLs(
	entryURL string,
	sourceDomain string,
	log *logger.Logger,
) ([]string, error) {
	mangaID, chapterID, err := readChapterIDs(entryURL)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("mangaId", mangaID)
	query.Set("chapterId", chapterID)

	body, err := apiGet(
		sourceDomain,
		"read chapter",
		readChapterPath,
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	var payload readChapterResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode Atsu.moe read chapter: %w",
			err,
		)
	}

	cdnBase, err := cdnBaseURL(sourceDomain)
	if err != nil {
		return nil, err
	}

	pages := make([]pageImage, 0, len(payload.ReadChapter.Pages))

	for _, raw := range payload.ReadChapter.Pages {
		imageURL := strings.TrimSpace(raw.Image)

		if imageURL == "" {
			continue
		}

		resolved, err := resolvePageImageURL(cdnBase, imageURL)
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping invalid Atsu.moe image URL: %s\n",
					imageURL,
				)
			}

			continue
		}

		pages = append(pages, pageImage{
			Number: raw.Number,
			URL:    resolved,
		})
	}

	sort.SliceStable(pages, func(i, j int) bool {
		return pages[i].Number < pages[j].Number
	})

	imageURLs := make([]string, 0, len(pages))

	for _, page := range pages {
		imageURLs = append(imageURLs, page.URL)
	}

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf(
			"no Atsu.moe page images found for chapter %s of manga %s",
			chapterID,
			mangaID,
		)
	}

	if log != nil {
		log.Debug(
			"Found %d Atsu.moe page image(s)\n",
			len(imageURLs),
		)
	}

	return imageURLs, nil
}

// readChapterIDs extracts the manga and chapter IDs from an entry
// URL built while listing chapters.
func readChapterIDs(
	entryURL string,
) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(entryURL))
	if err != nil {
		return "", "", fmt.Errorf(
			"invalid Atsu.moe entry URL %q: %w",
			entryURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")

	if !strings.HasSuffix(path, readChapterPath) {
		return "", "", fmt.Errorf(
			"unexpected Atsu.moe entry URL: %q",
			entryURL,
		)
	}

	query := parsed.Query()

	mangaID := strings.TrimSpace(query.Get("mangaId"))
	chapterID := strings.TrimSpace(query.Get("chapterId"))

	if mangaID == "" {
		return "", "", fmt.Errorf(
			"Atsu.moe entry URL has no mangaId query parameter: %q",
			entryURL,
		)
	}

	if chapterID == "" {
		return "", "", fmt.Errorf(
			"Atsu.moe entry URL has no chapterId query parameter: %q",
			entryURL,
		)
	}

	return mangaID, chapterID, nil
}

// cdnBaseURL builds the image CDN base URL from a provider domain.
func cdnBaseURL(domain string) (*url.URL, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return nil, err
	}

	baseURL.Host = "cdn." + baseURL.Host

	return baseURL, nil
}

// resolvePageImageURL makes an image URL absolute when needed and
// only keeps http(s) schemes.
func resolvePageImageURL(
	baseURL *url.URL,
	rawURL string,
) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}

	if baseURL != nil {
		parsed = baseURL.ResolveReference(parsed)
	}

	switch parsed.Scheme {
	case "http", "https":
	default:
		return "", fmt.Errorf(
			"unsupported image URL scheme: %s",
			parsed.Scheme,
		)
	}

	return parsed.String(), nil
}
