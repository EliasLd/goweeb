package mangavyvyscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/PuerkitoBio/goquery"
)

type pageImage struct {
	Index int
	URL   string
}

func GetPageImageURLs(
	entryURL string,
	sourceDomain string,
	log *logger.Logger,
) ([]string, error) {
	log.Debug(
		"Fetching Mangavyvy reader page: %s\n",
		entryURL,
	)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		entryURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create Mangavyvy reader request: %w",
			err,
		)
	}

	req.Header.Set(
		"User-Agent",
		"goweeb (+https://github.com/EliasLd/goweeb)",
	)

	req.Header.Set(
		"Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	)

	if strings.TrimSpace(sourceDomain) != "" {
		req.Header.Set(
			"Referer",
			strings.TrimRight(sourceDomain, "/")+"/",
		)
	}

	result, err := httpx.ReadAll(
		client,
		req,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch Mangavyvy reader page: %w",
			err,
		)
	}

	log.Debug(
		"Mangavyvy reader response status: %d\n",
		result.StatusCode,
	)

	// Preserve the final URL after Aovheroes redirects.
	readerURL := entryURL
	if result.FinalURL != "" {
		readerURL = result.FinalURL
	}

	log.Debug(
		"Mangavyvy reader final URL: %s\n",
		readerURL,
	)

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected Mangavyvy reader status: %d",
			result.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(
		bytes.NewReader(result.Body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse Mangavyvy reader page: %w",
			err,
		)
	}

	baseURL, err := url.Parse(readerURL)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid Mangavyvy reader URL: %w",
			err,
		)
	}
	var pages []pageImage

	seen := make(map[string]struct{})

	doc.Find(
		"#carousel .carousel-inner .carousel-item img",
	).Each(
		func(position int, selection *goquery.Selection) {
			index := position

			if parent := selection.Closest(
				".carousel-item",
			); parent.Length() > 0 {
				if rawPage, ok := parent.Attr(
					"data-page",
				); ok {
					rawPage = strings.TrimSpace(
						rawPage,
					)

					if parsedPage, err := strconv.Atoi(
						rawPage,
					); err == nil {
						index = parsedPage
					}
				}
			}

			imageURL := imageSource(
				selection,
			)

			if imageURL == "" {
				return
			}

			resolvedURL, err := resolveImageURL(
				baseURL,
				imageURL,
			)
			if err != nil {
				log.Debug(
					"Skipping invalid Mangavyvy image URL: %s\n",
					imageURL,
				)

				return
			}

			if _, exists := seen[resolvedURL]; exists {
				return
			}

			seen[resolvedURL] = struct{}{}

			pages = append(
				pages,
				pageImage{
					Index: index,
					URL:   resolvedURL,
				},
			)
		},
	)

	if len(pages) == 0 {
		return nil, fmt.Errorf(
			"no Mangavyvy page images found at %s",
			readerURL,
		)
	}

	sort.SliceStable(
		pages,
		func(i, j int) bool {
			return pages[i].Index <
				pages[j].Index
		},
	)

	imageURLs := make(
		[]string,
		0,
		len(pages),
	)

	for _, page := range pages {
		imageURLs = append(
			imageURLs,
			page.URL,
		)
	}

	log.Debug(
		"Found %d Mangavyvy page image(s)\n",
		len(imageURLs),
	)

	return imageURLs, nil
}

func imageSource(
	selection *goquery.Selection,
) string {
	// Mangavyvy lazy-loads later pages.
	//
	// data-src always contains the actual image while
	// src may contain /web/img/loading.gif.
	if value, ok := selection.Attr(
		"data-src",
	); ok {
		value = strings.TrimSpace(value)

		if value != "" {
			return value
		}
	}

	value, ok := selection.Attr("src")
	if !ok {
		return ""
	}

	value = strings.TrimSpace(value)

	if value == "" ||
		isPlaceholderImage(value) {
		return ""
	}

	return value
}

func isPlaceholderImage(
	imageURL string,
) bool {
	lower := strings.ToLower(imageURL)

	return strings.Contains(
		lower,
		"/web/img/loading.gif",
	) ||
		strings.Contains(
			lower,
			"/web/img/blank.gif",
		)
}

func resolveImageURL(
	baseURL *url.URL,
	rawURL string,
) (string, error) {
	parsed, err := url.Parse(
		strings.TrimSpace(rawURL),
	)
	if err != nil {
		return "", err
	}

	if baseURL != nil {
		parsed = baseURL.ResolveReference(
			parsed,
		)
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
