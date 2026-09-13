package mangafreakscraper

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/PuerkitoBio/goquery"
)

// Extracts all chapter image URLs from a MangaFreak reader page.
func GetPageImageURLs(
	chapterURL string,
	log *logger.Logger,
) ([]string, error) {
	chapterURL = strings.TrimSpace(chapterURL)

	if chapterURL == "" {
		return nil, fmt.Errorf("chapter URL is empty")
	}

	log.Debug("Fetching chapter page: %s\n", chapterURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		chapterURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create chapter request: %w",
			err,
		)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch chapter page: %w",
			err,
		)
	}
	defer resp.Body.Close()

	log.Debug(
		"Chapter page response status: %d\n",
		resp.StatusCode,
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"chapter page returned status: %d",
			resp.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse chapter page: %w",
			err,
		)
	}

	baseURL, err := url.Parse(chapterURL)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid chapter URL: %w",
			err,
		)
	}

	var imageURLs []string

	doc.Find("div.slideshow-container img").Each(
		func(i int, selection *goquery.Selection) {
			src, exists := selection.Attr("src")
			if !exists {
				return
			}

			src = strings.TrimSpace(src)

			if src == "" {
				return
			}

			imageURL, err := url.Parse(src)
			if err != nil {
				log.Debug(
					"Skipping invalid image URL: %s\n",
					src,
				)
				return
			}

			// Handles both absolute and relative image URLs.
			resolvedURL := baseURL.ResolveReference(imageURL).String()

			imageURLs = append(
				imageURLs,
				resolvedURL,
			)
		},
	)

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf(
			"no chapter images found at %s",
			chapterURL,
		)
	}

	log.Debug(
		"Found %d page image(s)\n",
		len(imageURLs),
	)

	return imageURLs, nil
}
