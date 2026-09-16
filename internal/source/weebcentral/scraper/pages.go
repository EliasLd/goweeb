package weebcentralscraper

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/PuerkitoBio/goquery"
)

func GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	entryURL = strings.TrimSpace(entryURL)

	if entryURL == "" {
		return nil, fmt.Errorf("chapter URL is empty")
	}

	chapterURL, err := url.Parse(entryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid chapter URL: %w", err)
	}

	chapterURL.Path = strings.TrimSuffix(chapterURL.Path, "/") + "/images"

	query := chapterURL.Query()
	query.Set("is_prev", "False")
	query.Set("reading_style", "long_strip")
	chapterURL.RawQuery = query.Encode()

	imagesURL := chapterURL.String()

	log.Debug("Fetching chapter images: %s\n", imagesURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		imagesURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create chapter images request: %w",
			err,
		)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch chapter images: %w",
			err,
		)
	}
	defer resp.Body.Close()

	log.Debug(
		"Chapter images response status: %d\n",
		resp.StatusCode,
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"chapter images returned status: %d",
			resp.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse chapter images: %w",
			err,
		)
	}

	imageURLs := make([]string, 0)

	doc.Find("section#chapter-images img[src]").Each(
		func(i int, selection *goquery.Selection) {
			src, exists := selection.Attr("src")
			if !exists {
				return
			}

			src = strings.TrimSpace(src)
			if src == "" {
				return
			}

			imageRef, err := url.Parse(src)
			if err != nil {
				log.Debug(
					"Skipping invalid image URL: %s\n",
					src,
				)
				return
			}

			imageURLs = append(
				imageURLs,
				chapterURL.ResolveReference(imageRef).String(),
			)
		},
	)

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf(
			"no chapter images found at %s",
			imagesURL,
		)
	}

	log.Debug(
		"Found %d page image(s)\n",
		len(imageURLs),
	)

	return imageURLs, nil
}
