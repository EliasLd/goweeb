package mangafreakscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	"github.com/PuerkitoBio/goquery"
)

type ScanInfo struct {
	MangaName string
	Chapters  []ChapterInfo
}

type ChapterInfo struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

func GetScanInfo(
	workURL string,
	log *logger.Logger,
) (*ScanInfo, error) {
	workURL = strings.TrimSuffix(workURL, "/")

	baseURL, err := url.Parse(workURL)
	if err != nil {
		return nil, fmt.Errorf("invalid manga URL: %w", err)
	}

	log.Debug("Fetching manga page: %s\n", workURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, workURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	result, err := httpx.ReadAll(
		client,
		req,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch manga page: %w",
			err,
		)
	}

	log.Debug(
		"Manga page response status: %d\n",
		result.StatusCode,
	)

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"manga page returned status: %d",
			result.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(
		bytes.NewReader(result.Body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse manga page: %w",
			err,
		)
	}

	mangaName := strings.TrimSpace(doc.Find("h1").First().Text())

	if mangaName == "" {
		mangaName = workURL
	}

	var chapters []ChapterInfo

	doc.Find("div.manga_series_list a.chapter-link").Each(
		func(i int, selection *goquery.Selection) {
			href, exists := selection.Attr("href")
			if !exists {
				return
			}

			label := strings.TrimSpace(selection.Text())
			if label == "" {
				return
			}

			// Expected:
			// "Chapter 1"
			// "Chapter 1193 - Imu is actually a parrot.."
			fields := strings.Fields(label)

			if len(fields) < 2 {
				return
			}

			chapterNumber, err := common.ParseChapterNumber(
				strings.TrimRight(fields[1], ":"),
			)
			if err != nil {
				return
			}

			chapterRef, err := url.Parse(href)
			if err != nil {
				return
			}

			href = baseURL.ResolveReference(chapterRef).String()

			chapters = append(chapters, ChapterInfo{
				Number: chapterNumber,
				Label:  label,
				URL:    href,
			})
		},
	)

	return &ScanInfo{
		MangaName: mangaName,
		Chapters:  chapters,
	}, nil
}
