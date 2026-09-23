package animesamascraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

type ChapterInfo struct {
	Number common.ChapterNumber
	Raw    string
}

type ScanInfo struct {
	MangaName string
	Chapters  []ChapterInfo
}

// Fetches scan info using the anime-sama API
func GetScanInfo(domain, mangaName string, log *logger.Logger) (*ScanInfo, error) {
	// Call the API endpoint
	apiURL := fmt.Sprintf("%s/s2/scans/get_nb_chap_et_img.php?oeuvre=%s",
		domain,
		url.QueryEscape(mangaName),
	)

	log.Debug("Fetching chapter list from API: %s\n", apiURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	var data map[string]int
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract and sort chapter numbers
	var chapters []ChapterInfo

	for raw := range data {
		number, err := common.ParseChapterNumber(raw)
		if err != nil {
			log.Debug(
				"Skipping invalid Anime-Sama chapter: %q\n",
				raw,
			)
			continue
		}

		chapters = append(chapters, ChapterInfo{
			Number: number,
			Raw:    raw,
		})
	}

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
