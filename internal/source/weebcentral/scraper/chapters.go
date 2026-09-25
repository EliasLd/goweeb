package weebcentralscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
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

// Extract the first chapter number from its dedicated label.
// Accepts "Chapter 271.5", "Episode 139.5", "#100",
// "Part 12", or simply "12.5".
var chapterLabelPattern = regexp.MustCompile(
	`^[^0-9]*([0-9]+(?:\.[0-9]+)?)(?:$|[\s:;,()\-–—])`,
)

func parseChapterLabel(
	text string,
) (common.ChapterNumber, string, bool) {
	var zero common.ChapterNumber

	label := strings.Join(
		strings.Fields(text),
		" ",
	)

	match := chapterLabelPattern.FindStringSubmatch(label)
	if len(match) != 2 {
		return zero, "", false
	}

	number, err := common.ParseChapterNumber(match[1])
	if err != nil {
		return zero, "", false
	}

	return number, label, true
}

func GetScanInfo(
	domain string,
	workURL string,
	log *logger.Logger,
) (*ScanInfo, error) {
	seriesID, mangaName, err := extractSeriesInfo(workURL)
	if err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(strings.TrimSuffix(domain, "/"))
	if err != nil {
		return nil, fmt.Errorf(
			"invalid WeebCentral domain: %w",
			err,
		)
	}

	chapterListRef, err := url.Parse(
		fmt.Sprintf(
			"/series/%s/full-chapter-list",
			seriesID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to build chapter list URL: %w",
			err,
		)
	}

	chapterListURL := baseURL.ResolveReference(
		chapterListRef,
	).String()

	log.Debug(
		"Fetching chapter list: %s\n",
		chapterListURL,
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		chapterListURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create chapter list request: %w",
			err,
		)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	result, err := httpx.ReadAll(
		client,
		req,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch chapter list: %w",
			err,
		)
	}

	log.Debug(
		"Chapter list response status: %d\n",
		result.StatusCode,
	)

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"chapter list returned status: %d",
			result.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(
		bytes.NewReader(result.Body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse chapter list: %w",
			err,
		)
	}

	chapterList := doc.Find("#chapter-list")

	// The full-chapter-list endpoint may return an HTML fragment
	// without the outer #chapter-list container.
	if chapterList.Length() == 0 {
		log.Debug(
			"Chapter list container not found; parsing response as an HTML fragment\n",
		)

		chapterList = doc.Selection
	}

	var chapters []ChapterInfo

	chapterList.Find(`a[href^="/chapters/"]`).Each(
		func(_ int, selection *goquery.Selection) {
			href, exists := selection.Attr("href")
			if !exists {
				return
			}

			// Read only the chapter's dedicated label.
			// Ignore nested status elements such as "Last Read".
			rawLabel := selection.
				Find("span.grow > span").
				First().
				Text()

			number, label, ok := parseChapterLabel(rawLabel)
			if !ok {
				log.Debug(
					"Skipping chapter with unrecognized label: %q\n",
					strings.TrimSpace(rawLabel),
				)
				return
			}

			chapterRef, err := url.Parse(href)
			if err != nil {
				log.Debug(
					"Skipping invalid chapter URL: %s\n",
					href,
				)
				return
			}

			chapterURL := baseURL.ResolveReference(
				chapterRef,
			).String()

			chapters = append(
				chapters,
				ChapterInfo{
					Number: number,
					Label:  label,
					URL:    chapterURL,
				},
			)
		},
	)

	if len(chapters) == 0 {
		return nil, fmt.Errorf(
			"no chapters found for series %s",
			seriesID,
		)
	}

	// WeebCentral returns newest chapters first.
	// goweeb expects chapters in ascending order.
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].Number.Compare(
			chapters[j].Number,
		) < 0
	})

	return &ScanInfo{
		MangaName: mangaName,
		Chapters:  chapters,
	}, nil
}

func extractSeriesInfo(
	workURL string,
) (string, string, error) {
	u, err := url.Parse(workURL)
	if err != nil {
		return "", "", fmt.Errorf(
			"invalid series URL: %w",
			err,
		)
	}

	parts := strings.Split(
		strings.Trim(u.Path, "/"),
		"/",
	)

	// Expected:
	// /series/<ID>/<TITLE>
	if len(parts) < 3 || parts[0] != "series" {
		return "", "", fmt.Errorf(
			"unexpected WeebCentral series URL: %s",
			workURL,
		)
	}

	seriesID := parts[1]

	slug, err := url.PathUnescape(parts[2])
	if err != nil {
		return "", "", fmt.Errorf(
			"failed to decode manga slug: %w",
			err,
		)
	}

	mangaName := strings.ReplaceAll(
		slug,
		"-",
		" ",
	)

	return seriesID, mangaName, nil
}
