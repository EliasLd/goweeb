package mangadotnetscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const (
	uploadsImagesPathSuffix  = "/uploads"
	chaptersImagesPathSuffix = "/chapters"
	imagesPathSuffix         = "/images"
)

type chapterImagesResponse struct {
	Images []chapterImage `json:"images"`
}

type chapterImage struct {
	// URL is relative to the site domain, e.g.
	// /chapters/manga_101/user_0_ch1_es_e2a0987dd63ee762/001.webp
	URL string `json:"url"`
}

// chapterImagesPath builds the chapter page images endpoint:
//
//	{domain}/api/uploads/{chapterID}/images (user-uploaded chapters)
//	{domain}/api/chapters/{chapterID}/images (imported chapters)
func chapterImagesPath(chapterID string, userSource bool) string {
	imagesSegment := chaptersImagesPathSuffix

	if userSource {
		imagesSegment = uploadsImagesPathSuffix
	}

	return "/api" + imagesSegment + "/" + url.PathEscape(chapterID) +
		imagesPathSuffix
}

// GetPageImageURLs returns the ordered page image URLs of a chapter.
//
// The entry URL is the public reader endpoint built while listing
// chapters:
//
//	{domain}/chapter/{chapterID}              (scraper chapters)
//	{domain}/chapter/{chapterID}?source=user  (user uploads)
//
// Chapter ids belong to two separate id spaces (user uploads vs
// imported chapters), hence the two endpoints. Image paths returned by
// the API are relative to the site domain and are kept in the API
// order, from the first page to the last one.
func GetPageImageURLs(
	entryURL string,
	sourceDomain string,
	log *logger.Logger,
) ([]string, error) {
	chapterID, userSource, err := parseChapterEntry(entryURL)
	if err != nil {
		return nil, err
	}

	body, err := apiGet(
		sourceDomain,
		"chapter images",
		chapterImagesPath(chapterID, userSource),
		nil,
		log,
	)
	if err != nil {
		return nil, err
	}

	var payload chapterImagesResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode mangadot.net chapter images: %w",
			err,
		)
	}

	baseURL, err := parseBaseDomain(sourceDomain)
	if err != nil {
		return nil, err
	}

	imageURLs := make([]string, 0, len(payload.Images))

	for _, image := range payload.Images {
		imageURL := strings.TrimSpace(image.URL)

		if imageURL == "" {
			continue
		}

		resolved, err := resolvePageImageURL(baseURL, imageURL)
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping invalid mangadot.net image URL: %s\n",
					imageURL,
				)
			}

			continue
		}

		imageURLs = append(imageURLs, resolved)
	}

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf(
			"no mangadot.net page images found for chapter %s",
			chapterID,
		)
	}

	if log != nil {
		log.Debug(
			"Found %d mangadot.net page image(s)\n",
			len(imageURLs),
		)
	}

	return imageURLs, nil
}

// parseChapterEntry extracts the chapter id and source of an entry URL
// built while listing chapters:
//
//	{domain}/chapter/{chapterID}              (scraper chapters)
//	{domain}/chapter/{chapterID}?source=user  (user uploads)
//
// Unknown query parameters are ignored; only source=user switches the
// images endpoint.
func parseChapterEntry(
	entryURL string,
) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(entryURL))
	if err != nil {
		return "", false, fmt.Errorf(
			"invalid mangadot.net entry URL %q: %w",
			entryURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")
	segments := strings.Split(path, "/")

	// Expected: ["", "chapter", "{chapterID}"]
	if len(segments) != 3 || segments[1] != "chapter" {
		return "", false, fmt.Errorf(
			"unexpected mangadot.net entry URL: %q",
			entryURL,
		)
	}

	chapterID, err := url.PathUnescape(segments[2])
	if err != nil {
		return "", false, fmt.Errorf(
			"unexpected mangadot.net entry URL: %q",
			entryURL,
		)
	}

	chapterID = strings.TrimSpace(chapterID)

	if _, err := strconv.ParseInt(chapterID, 10, 64); err != nil {
		return "", false, fmt.Errorf(
			"unexpected mangadot.net entry URL: %q",
			entryURL,
		)
	}

	userSource := parsed.Query().Get("source") == userSource

	return chapterID, userSource, nil
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

	parsed = baseURL.ResolveReference(parsed)

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
