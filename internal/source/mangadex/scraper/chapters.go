package mangadexscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

const chapterFeedPageSize = 100

type ScanInfo struct {
	MangaName string
	Chapters  []ChapterInfo
}

type ChapterInfo struct {
	Number      common.ChapterNumber
	Label       string
	URL         string
	ID          string
	PublishedAt string
}

type chapterFeedResponse struct {
	Result string         `json:"result"`
	Data   []chapterEntry `json:"data"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
}

type chapterEntry struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes chapterAttributes `json:"attributes"`
}

type chapterAttributes struct {
	Volume             *string `json:"volume"`
	Chapter            *string `json:"chapter"`
	Title              string  `json:"title"`
	TranslatedLanguage string  `json:"translatedLanguage"`
	ExternalURL        *string `json:"externalUrl"`
	Pages              int     `json:"pages"`
	PublishAt          string  `json:"publishAt"`
}

type mangaInfoResponse struct {
	Result string        `json:"result"`
	Data   mangaInfoData `json:"data"`
}

type mangaInfoData struct {
	ID         string              `json:"id"`
	Attributes mangaInfoAttributes `json:"attributes"`
}

type mangaInfoAttributes struct {
	Title                          map[string]string `json:"title"`
	ChapterNumbersResetOnNewVolume bool              `json:"chapterNumbersResetOnNewVolume"`
}

func GetScanInfo(
	domain string,
	workURL string,
	language string,
	log *logger.Logger,
) (*ScanInfo, error) {
	mangaID, err := extractMangaID(workURL)
	if err != nil {
		return nil, err
	}

	language = strings.TrimSpace(language)
	if language == "" {
		return nil, fmt.Errorf("MangaDex language is empty")
	}

	mangaName, resetsOnNewVolume, err := getMangaInfo(
		workURL,
		log,
	)
	if err != nil {
		return nil, err
	}

	// goweeb currently represents chapter numbers as a single int.
	// If a manga restarts chapter numbering for every volume, range
	// selection would become ambiguous.
	if resetsOnNewVolume {
		return nil, fmt.Errorf(
			"MangaDex manga %q resets chapter numbers on new volumes, which is not supported yet",
			mangaName,
		)
	}

	baseURL, err := url.Parse(strings.TrimSuffix(domain, "/"))
	if err != nil {
		return nil, fmt.Errorf(
			"invalid MangaDex domain: %w",
			err,
		)
	}

	feedRef, err := url.Parse(
		fmt.Sprintf("/manga/%s/feed", mangaID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to build MangaDex chapter feed URL: %w",
			err,
		)
	}

	feedURL := baseURL.ResolveReference(feedRef)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// MangaDex can contain several translations of the exact same
	// chapter in the same language, uploaded by different groups.
	//
	// goweeb currently expects one Entry per chapter number, so keep
	// the most recently published version.
	chaptersByNumber := make(map[common.ChapterNumber]ChapterInfo)

	offset := 0
	skippedNonInteger := 0

	for {
		requestURL := *feedURL

		query := requestURL.Query()

		query.Set("limit", strconv.Itoa(chapterFeedPageSize))
		query.Set("offset", strconv.Itoa(offset))

		query.Add(
			"translatedLanguage[]",
			language,
		)

		query.Set(
			"order[chapter]",
			"asc",
		)

		// Only chapters that goweeb can actually download.
		query.Set(
			"includeEmptyPages",
			"0",
		)

		query.Set(
			"includeExternalUrl",
			"0",
		)

		query.Set(
			"includeFuturePublishAt",
			"0",
		)

		requestURL.RawQuery = query.Encode()

		log.Debug(
			"Fetching MangaDex chapter feed: %s\n",
			requestURL.String(),
		)

		req, err := http.NewRequest(
			http.MethodGet,
			requestURL.String(),
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create MangaDex chapter feed request: %w",
				err,
			)
		}

		req.Header.Set(
			"Accept",
			"application/json",
		)

		req.Header.Set(
			"User-Agent",
			"goweeb",
		)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to fetch MangaDex chapter feed: %w",
				err,
			)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()

			return nil, fmt.Errorf(
				"MangaDex chapter feed returned status: %d",
				resp.StatusCode,
			)
		}

		var response chapterFeedResponse

		err = json.NewDecoder(resp.Body).Decode(&response)
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf(
				"failed to decode MangaDex chapter feed: %w",
				err,
			)
		}

		if response.Result != "ok" {
			return nil, fmt.Errorf(
				"MangaDex returned unexpected result: %s",
				response.Result,
			)
		}

		log.Debug(
			"MangaDex chapter feed page: offset=%d results=%d total=%d\n",
			response.Offset,
			len(response.Data),
			response.Total,
		)

		for _, chapter := range response.Data {
			if chapter.ID == "" {
				continue
			}

			// These should already be excluded by the API filters,
			// but keeping the checks here protects us against an
			// unexpected API response.
			if chapter.Attributes.Pages <= 0 {
				continue
			}

			if chapter.Attributes.ExternalURL != nil &&
				strings.TrimSpace(*chapter.Attributes.ExternalURL) != "" {
				continue
			}

			if chapter.Attributes.Chapter == nil {
				log.Debug(
					"Skipping MangaDex chapter %s with no chapter number\n",
					chapter.ID,
				)
				continue
			}

			chapterNumberText := strings.TrimSpace(
				*chapter.Attributes.Chapter,
			)

			if chapterNumberText == "" {
				continue
			}

			chapterNumber, err := common.ParseChapterNumber(
				chapterNumberText,
			)
			if err != nil {
				log.Debug(
					"Skipping invalid MangaDex chapter number: %q\n",
					chapterNumberText,
				)
				continue
			}

			chapterRef, err := url.Parse(
				fmt.Sprintf(
					"/chapter/%s",
					chapter.ID,
				),
			)
			if err != nil {
				log.Debug(
					"Skipping invalid MangaDex chapter ID: %s\n",
					chapter.ID,
				)

				continue
			}

			chapterURL := baseURL.ResolveReference(
				chapterRef,
			).String()

			label := fmt.Sprintf(
				"Chapter %s",
				chapterNumberText,
			)

			info := ChapterInfo{
				Number:      chapterNumber,
				Label:       label,
				URL:         chapterURL,
				ID:          chapter.ID,
				PublishedAt: chapter.Attributes.PublishAt,
			}

			existing, exists := chaptersByNumber[chapterNumber]

			if !exists || isNewerChapter(
				info,
				existing,
			) {
				chaptersByNumber[chapterNumber] = info
			}
		}

		nextOffset := response.Offset + len(response.Data)

		if nextOffset >= response.Total {
			break
		}

		if len(response.Data) == 0 {
			return nil, fmt.Errorf(
				"MangaDex chapter feed pagination stopped unexpectedly at offset %d of %d",
				response.Offset,
				response.Total,
			)
		}

		offset = nextOffset
	}

	chapters := make(
		[]ChapterInfo,
		0,
		len(chaptersByNumber),
	)

	for _, chapter := range chaptersByNumber {
		chapters = append(
			chapters,
			chapter,
		)
	}

	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].Number.Compare(
			chapters[j].Number,
		) < 0
	})

	if len(chapters) == 0 {
		return nil, fmt.Errorf(
			"no downloadable MangaDex chapters found for language %s",
			language,
		)
	}

	log.Debug(
		"Found %d unique MangaDex chapter(s) for language %s\n",
		len(chapters),
		language,
	)

	if skippedNonInteger > 0 {
		log.Debug(
			"Skipped %d non-integer MangaDex chapter(s)\n",
			skippedNonInteger,
		)
	}

	return &ScanInfo{
		MangaName: mangaName,
		Chapters:  chapters,
	}, nil
}

func getMangaInfo(
	workURL string,
	log *logger.Logger,
) (string, bool, error) {
	log.Debug(
		"Fetching MangaDex manga info: %s\n",
		workURL,
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		workURL,
		nil,
	)
	if err != nil {
		return "", false, fmt.Errorf(
			"failed to create MangaDex manga info request: %w",
			err,
		)
	}

	req.Header.Set(
		"Accept",
		"application/json",
	)

	req.Header.Set(
		"User-Agent",
		"goweeb",
	)

	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf(
			"failed to fetch MangaDex manga info: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf(
			"MangaDex manga info returned status: %d",
			resp.StatusCode,
		)
	}

	var response mangaInfoResponse

	if err := json.NewDecoder(resp.Body).Decode(
		&response,
	); err != nil {
		return "", false, fmt.Errorf(
			"failed to decode MangaDex manga info: %w",
			err,
		)
	}

	if response.Result != "ok" {
		return "", false, fmt.Errorf(
			"MangaDex returned unexpected result: %s",
			response.Result,
		)
	}

	title := selectMangaTitle(
		response.Data.Attributes.Title,
	)

	if title == "" {
		title = response.Data.ID
	}

	return title,
		response.Data.Attributes.ChapterNumbersResetOnNewVolume,
		nil
}

func extractMangaID(
	workURL string,
) (string, error) {
	u, err := url.Parse(workURL)
	if err != nil {
		return "", fmt.Errorf(
			"invalid MangaDex manga URL: %w",
			err,
		)
	}

	parts := strings.Split(
		strings.Trim(u.Path, "/"),
		"/",
	)

	if len(parts) < 2 {
		return "", fmt.Errorf(
			"unexpected MangaDex manga URL: %s",
			workURL,
		)
	}

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "manga" {
			mangaID := strings.TrimSpace(
				parts[i+1],
			)

			if mangaID != "" {
				return mangaID, nil
			}
		}
	}

	return "", fmt.Errorf(
		"unable to extract MangaDex manga ID from URL: %s",
		workURL,
	)
}

func isNewerChapter(
	candidate ChapterInfo,
	current ChapterInfo,
) bool {
	candidateTime, candidateErr := time.Parse(
		time.RFC3339,
		candidate.PublishedAt,
	)

	currentTime, currentErr := time.Parse(
		time.RFC3339,
		current.PublishedAt,
	)

	if candidateErr == nil && currentErr == nil {
		if candidateTime.Equal(currentTime) {
			return candidate.ID > current.ID
		}

		return candidateTime.After(currentTime)
	}

	if candidateErr == nil && currentErr != nil {
		return true
	}

	if candidateErr != nil && currentErr == nil {
		return false
	}

	// Deterministic fallback.
	return candidate.ID > current.ID
}
