package mangadexscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// aggregateChapter is one chapter inside a tagged volume of the
// aggregate payload. others[] lists same-number alternatives from
// other groups (and count includes them), so the main entry is the
// only thing a volume bundles.
type aggregateChapter struct {
	Chapter       *string `json:"chapter"`
	ID            string  `json:"id"`
	IsUnavailable bool    `json:"isUnavailable"`
}

// aggregateVolume decodes one entry of the volumes container. Its
// chapters container is keyed by chapter number and also carries the
// volume total as a bare number under the "count" key of the outer
// container, so anything that is not an object decodes to nothing
// and the key-based routing skips it.
type aggregateVolume struct {
	Chapters map[string]aggregateChapter `json:"chapters"`
}

func (v *aggregateVolume) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || data[0] != '{' {
		return nil
	}

	type plain aggregateVolume
	return json.Unmarshal(data, (*plain)(v))
}

type aggregateResponse struct {
	Result  string                     `json:"result"`
	Volumes map[string]aggregateVolume `json:"volumes"`
}

// ListVolumes builds volume mode from real site data: one detail
// call for the canonical title plus one aggregate call filtered to
// the scan path's language. Volume membership comes from the site's
// own tags; the "none" bucket (and any non-number key) holds
// untagged chapters and stays chapter-mode only.
func ListVolumes(
	domain string,
	workURL string,
	language string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Volume, error) {
	mangaID, err := extractMangaID(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	language = strings.TrimSpace(language)
	if language == "" {
		return sourcetypes.Work{}, nil, fmt.Errorf("MangaDex language is empty")
	}

	mangaDetail, err := GetMangaDetail(domain, workURL, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get MangaDex work details: %w",
			err,
		)
	}

	baseURL, err := domainURL(domain)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	aggregateURL := baseURL.JoinPath("manga", mangaID, "aggregate")
	aggregateURL.RawQuery = url.Values{
		"translatedLanguage[]": {language},
	}.Encode()

	log.Debug(
		"Fetching MangaDex aggregate: %s\n",
		aggregateURL.String(),
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		aggregateURL.String(),
		nil,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to create MangaDex aggregate request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "goweeb")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to fetch MangaDex aggregate: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"MangaDex aggregate returned status: %d",
			result.StatusCode,
		)
	}

	var response aggregateResponse
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to decode MangaDex aggregate: %w",
			err,
		)
	}

	if response.Result != "ok" {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"MangaDex returned unexpected result: %s",
			response.Result,
		)
	}

	type taggedVolume struct {
		key    string
		number common.ChapterNumber
		data   aggregateVolume
	}

	// The container is a JSON object, so its ordering is not
	// contractual; restore it by parsing the volume keys as numbers.
	tagged := make([]taggedVolume, 0, len(response.Volumes))

	for key, data := range response.Volumes {
		number, err := common.ParseChapterNumber(key)
		if err != nil {
			log.Debug(
				"Skipping MangaDex volume %q: no volume number\n",
				key,
			)

			continue
		}

		tagged = append(tagged, taggedVolume{
			key:    key,
			number: number,
			data:   data,
		})
	}

	sort.Slice(tagged, func(i, j int) bool {
		return tagged[i].number.Compare(tagged[j].number) < 0
	})

	volumes := make([]sourcetypes.Volume, 0, len(tagged))

	for _, item := range tagged {
		entries, reason := volumeEntries(baseURL, item.data, log)
		if reason != "" {
			log.Warn(
				"Skipping MangaDex volume %s: %s\n",
				item.key,
				reason,
			)

			continue
		}

		volumes = append(volumes, sourcetypes.Volume{
			Number:  item.key,
			Entries: entries,
		})
	}

	log.Debug(
		"Found %d MangaDex volume(s) for language %s\n",
		len(volumes),
		language,
	)

	return sourcetypes.Work{
		Title: mangaDetail.Name,
		Kind:  sourcetypes.ItemVolume,
	}, volumes, nil
}

// volumeEntries expands one tagged volume into its ordered chapter
// entries. A volume carrying anything goweeb cannot bundle — an
// unavailable, unnumbered or unreadable chapter — returns a refusal
// reason instead of a partial bundle.
func volumeEntries(
	baseURL *url.URL,
	data aggregateVolume,
	log *logger.Logger,
) ([]sourcetypes.Entry, string) {
	if len(data.Chapters) == 0 {
		return nil, "volume without chapters"
	}

	// The chapters container is a JSON object, so its ordering is
	// not contractual; a deterministic walk pins which chapter a
	// refusal reports first. The entries themselves re-sort below.
	chapters := make([]aggregateChapter, 0, len(data.Chapters))
	for _, chapter := range data.Chapters {
		chapters = append(chapters, chapter)
	}

	sort.SliceStable(chapters, func(i, j int) bool {
		return volumeChapterSortKey(chapters[i].Chapter) <
			volumeChapterSortKey(chapters[j].Chapter)
	})

	entries := make([]sourcetypes.Entry, 0, len(chapters))
	numbers := make(map[common.ChapterNumber]struct{}, len(chapters))

	for _, chapter := range data.Chapters {
		if chapter.ID == "" {
			return nil, "chapter without an id"
		}

		if chapter.IsUnavailable {
			reason := "a chapter is unavailable"

			if chapter.Chapter != nil {
				reason = fmt.Sprintf(
					"chapter %s is unavailable",
					strings.TrimSpace(*chapter.Chapter),
				)
			}

			return nil, reason
		}

		if chapter.Chapter == nil {
			return nil, "chapter without a number"
		}

		chapterText := strings.TrimSpace(*chapter.Chapter)
		if chapterText == "" {
			return nil, "chapter without a number"
		}

		number, err := common.ParseChapterNumber(chapterText)
		if err != nil {
			return nil, fmt.Sprintf(
				"unreadable chapter number %q",
				chapterText,
			)
		}

		if _, duplicate := numbers[number]; duplicate {
			log.Debug(
				"Skipping duplicate MangaDex chapter %s in one volume\n",
				number.String(),
			)

			continue
		}

		numbers[number] = struct{}{}

		entries = append(entries, sourcetypes.Entry{
			Number: number,
			Label:  fmt.Sprintf("Chapter %s", chapterText),
			URL:    baseURL.JoinPath("chapter", chapter.ID).String(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Number.Compare(entries[j].Number) < 0
	})

	return entries, ""
}

// volumeChapterSortKey renders a chapter number so the volume walk
// stays deterministic; unreadable ones sort last.
func volumeChapterSortKey(chapter *string) string {
	if chapter == nil {
		return volumeSortKeyUnreadable
	}

	number, err := common.ParseChapterNumber(*chapter)
	if err != nil {
		return volumeSortKeyUnreadable
	}

	// Pad the whole-part length so lexical order tracks numeric
	// order: 1.5 sorts before 2, 12 before 100.
	parts := strings.SplitN(string(number), ".", 2)

	return fmt.Sprintf(
		"%02d%s",
		len(parts[0]),
		number.String(),
	)
}

const volumeSortKeyUnreadable = "zz"
