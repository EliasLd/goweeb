package mangavyvyscraper

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"

	"github.com/PuerkitoBio/goquery"
)

type Chapter struct {
	Number int
	Label  string
	URL    string
}

func ListChapters(
	workURL string,
	log *logger.Logger,
) ([]Chapter, error) {
	log.Debug(
		"Fetching Mangavyvy chapter list: %s\n",
		workURL,
	)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		workURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create Mangavyvy chapter request: %w",
			err,
		)
	}

	req.Header.Set(
		"User-Agent",
		"goweeb (+https://github.com/EliasLd/goweeb)",
	)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch Mangavyvy chapter list: %w",
			err,
		)
	}
	defer resp.Body.Close()

	log.Debug(
		"Mangavyvy chapter list response status: %d\n",
		resp.StatusCode,
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected Mangavyvy chapter list status: %d",
			resp.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(
		resp.Body,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse Mangavyvy chapter list: %w",
			err,
		)
	}

	baseURL, err := url.Parse(workURL)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid Mangavyvy work URL: %w",
			err,
		)
	}

	var chapters []Chapter

	doc.Find("a.list-chapter").Each(
		func(_ int, selection *goquery.Selection) {
			rawID, ok := selection.Attr("id")
			if !ok {
				return
			}

			numberStr := strings.TrimPrefix(
				rawID,
				"chapter-",
			)

			if numberStr == rawID ||
				numberStr == "" {
				return
			}

			number, err := strconv.Atoi(numberStr)
			if err != nil {
				log.Debug(
					"Skipping unsupported non-integer Mangavyvy chapter: %s\n",
					numberStr,
				)

				return
			}

			href, ok := selection.Attr("href")
			if !ok ||
				strings.TrimSpace(href) == "" {
				return
			}

			chapterURL, err := url.Parse(
				strings.TrimSpace(href),
			)
			if err != nil {
				return
			}

			chapterURL = baseURL.ResolveReference(
				chapterURL,
			)

			label := strings.Join(
				strings.Fields(
					selection.Find("span").
						First().
						Text(),
				),
				" ",
			)

			if label == "" {
				label = fmt.Sprintf(
					"Chapter %d",
					number,
				)
			}

			chapters = append(
				chapters,
				Chapter{
					Number: number,
					Label:  label,
					URL:    chapterURL.String(),
				},
			)
		},
	)

	if len(chapters) == 0 {
		return nil, fmt.Errorf(
			"no Mangavyvy chapters found",
		)
	}

	// Mangavyvy renders newest chapters first.
	// goweeb expects entries in ascending chapter order.
	sort.Slice(
		chapters,
		func(i, j int) bool {
			return chapters[i].Number <
				chapters[j].Number
		},
	)

	log.Debug(
		"Found %d Mangavyvy chapter(s)\n",
		len(chapters),
	)

	return chapters, nil
}
