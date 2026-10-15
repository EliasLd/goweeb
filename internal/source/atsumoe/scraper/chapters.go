package atsumoescraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

const allChaptersPath = "/api/manga/allChapters"

type Chapter struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

type allChaptersResponse struct {
	Chapters []chapterPayload `json:"chapters"`
}

type chapterPayload struct {
	ID                string      `json:"id"`
	ScanlationMangaID string      `json:"scanlationMangaId"`
	Title             string      `json:"title"`
	Number            json.Number `json:"number"`
	Index             int         `json:"index"`
	PageCount         int         `json:"pageCount"`
}

// orderedChapter keeps a parsed chapter along with its position in the
// raw API response so sorting is deterministic when chapter numbers
// are equal.
type orderedChapter struct {
	chapter Chapter
	index   int
}

// ListChapters returns the downloadable chapters of a manga, keeping
// only the selected scanlation group when scanID is not empty.
//
// The complete chapter list is returned; chapter range filtering is
// handled by the generic application code. Chapters are sorted from
// the oldest to the newest.
func ListChapters(
	domain string,
	mangaID string,
	scanID string,
	log *logger.Logger,
) ([]Chapter, error) {
	scanID = strings.TrimSpace(scanID)

	query := url.Values{}
	query.Set("mangaId", mangaID)

	body, err := apiGet(domain, "chapter list", allChaptersPath, query, log)
	if err != nil {
		return nil, err
	}

	var payload allChaptersResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf(
			"failed to decode Atsu.moe chapter list: %w",
			err,
		)
	}

	chapters := make([]orderedChapter, 0, len(payload.Chapters))

	for _, raw := range payload.Chapters {
		if raw.ID == "" {
			continue
		}

		// An empty scanID keeps chapters from every scanlation group.
		if scanID != "" &&
			strings.TrimSpace(raw.ScanlationMangaID) != scanID {
			continue
		}

		// Chapters without pages cannot be downloaded. This should
		// never happen, but protects us against an unexpected API
		// response.
		if raw.PageCount <= 0 {
			if log != nil {
				log.Debug(
					"Skipping Atsu.moe chapter with no pages: %s\n",
					raw.ID,
				)
			}

			continue
		}

		// json.Number keeps the exact decimal sent by the API, for
		// example 6.5 or 350.5.
		number, err := common.ParseChapterNumber(raw.Number.String())
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping invalid Atsu.moe chapter number: %s\n",
					raw.ID,
				)
			}

			continue
		}

		entryURL, err := readChapterURL(domain, mangaID, raw.ID)
		if err != nil {
			return nil, err
		}

		label := strings.TrimSpace(raw.Title)

		if label == "" {
			label = "Chapter " + number.String()
		}

		chapters = append(chapters, orderedChapter{
			chapter: Chapter{
				Number: number,
				Label:  label,
				URL:    entryURL,
			},
			index: raw.Index,
		})
	}

	sort.Slice(chapters, func(i, j int) bool {
		if c := chapters[i].chapter.Number.Compare(
			chapters[j].chapter.Number,
		); c != 0 {
			return c < 0
		}

		return chapters[i].index < chapters[j].index
	})

	out := make([]Chapter, 0, len(chapters))

	for _, entry := range chapters {
		out = append(out, entry.chapter)
	}

	if log != nil {
		log.Debug(
			"Found %d Atsu.moe chapter(s)\n",
			len(out),
		)
	}

	return out, nil
}

// readChapterURL points to the public read chapter endpoint, which is
// used by goweeb to retrieve page images. It keeps both IDs required
// for the next step of the workflow.
func readChapterURL(
	domain string,
	mangaID string,
	chapterID string,
) (string, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return "", err
	}

	endpoint := *baseURL
	endpoint.Path += "/api/read/chapter"

	query := url.Values{}
	query.Set("chapterId", chapterID)
	query.Set("mangaId", mangaID)

	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}
