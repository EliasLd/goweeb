package mangafreakscraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/PuerkitoBio/goquery"
)

type ScanInfo struct {
	MangaName string
	Chapters  []ChapterInfo
}

type ChapterInfo struct {
	Number int
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga page: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("Manga page response status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"manga page returned status: %d",
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manga page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse manga page: %w", err)
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

			chapterNumber, err := strconv.Atoi(fields[1])
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
