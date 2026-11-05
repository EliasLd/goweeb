package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

const titleDetailPath = "/api/v1/title/detail"

// chapterDetailPathPrefix is the site page path a chapter is readable
// at. Entry URLs carry it followed by one or more comma-separated
// chapter ids (see chapterEntryURL).
const chapterDetailPathPrefix = "/chapter-detail/"

type Chapter struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

type titleDetailResponse struct {
	Data struct {
		Name string `json:"name"`
	} `json:"data"`
}

// GetTitleName fetches a title's display name. The chapter listing
// carries no title, so this is a light dedicated call.
func GetTitleName(
	domain string,
	titleID string,
	log *logger.Logger,
) (string, error) {
	body, err := apiPost(
		domain,
		"title detail",
		titleDetailPath,
		map[string]string{"title_id": titleID},
		log,
	)
	if err != nil {
		return "", err
	}

	var response titleDetailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf(
			"failed to decode MangaBall title detail: %w",
			err,
		)
	}

	name := strings.TrimSpace(response.Data.Name)
	if name == "" {
		return "", fmt.Errorf(
			"MangaBall title %s has no name",
			titleID,
		)
	}

	return name, nil
}

// ListChapters returns the complete chapter list of one language,
// oldest chapter first.
//
// The same chapter number is usually crawled from several sites, and
// the API has no server-side deduplication — rows are all the crawled
// copies. One entry is emitted per distinct number, carrying every
// known source, because the richest copy (the page-count winner) can
// only be resolved at download time, when each copy's pages are
// inspected. Copies keep the order the API listed them in, so ties
// resolve to the site's own preference.
//
// Rows with an unusable id or number are skipped defensively; a
// listing the walk could not finish is a hard error (silently
// truncated chapters would download wrong ranges).
func ListChapters(
	domain string,
	titleID string,
	language string,
	log *logger.Logger,
) ([]Chapter, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return nil, err
	}

	listing, err := fetchChapterRows(domain, titleID, language, log)
	if err != nil {
		return nil, err
	}

	if !listing.complete {
		return nil, fmt.Errorf(
			"MangaBall chapter listing is incomplete: %d of %d rows after %d pages",
			len(listing.rows),
			listing.declaredTotal,
			maxListingPages,
		)
	}

	type chapterSources struct {
		number common.ChapterNumber
		ids    []string
		first  chapterRow
	}

	// Group rows per distinct chapter number, keeping the API's
	// own row order inside each group.
	numbered := make(map[string]*chapterSources)

	for _, row := range listing.rows {
		if row.ID == "" {
			if log != nil {
				log.Debug(
					"Skipping MangaBall chapter row without id: %q\n",
					row.Name,
				)
			}

			continue
		}

		number, err := common.ParseChapterNumber(row.Number.String())
		if err != nil {
			if log != nil {
				log.Debug(
					"Skipping MangaBall chapter row with invalid number %q: %q\n",
					row.Number.String(),
					row.Name,
				)
			}

			continue
		}

		if sources := numbered[number.String()]; sources != nil {
			sources.ids = append(sources.ids, row.ID)
			continue
		}

		numbered[number.String()] = &chapterSources{
			number: number,
			ids:    []string{row.ID},
			first:  row,
		}
	}

	// Numbers are unique by construction; the only work is
	// restoring the ascending order the app expects.
	groups := make([]*chapterSources, 0, len(numbered))

	for _, sources := range numbered {
		groups = append(groups, sources)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].number.Compare(groups[j].number) < 0
	})

	chapters := make([]Chapter, 0, len(groups))

	for _, sources := range groups {
		chapters = append(chapters, Chapter{
			Number: sources.number,
			Label:  chapterLabel(sources.first),
			URL:    chapterEntryURL(baseURL, sources.ids),
		})
	}

	if log != nil {
		log.Debug(
			"Found %d MangaBall chapters (language %q)\n",
			len(chapters),
			language,
		)
	}

	return chapters, nil
}

// chapterLabel prefers the site's own chapter label and falls back to
// a "Chapter N" rendering.
func chapterLabel(row chapterRow) string {
	if label := strings.TrimSpace(row.Name); label != "" {
		return label
	}

	return "Chapter " + row.Number.String()
}

// chapterEntryURL builds the paginated entry URL for one chapter
// number. Every crawled copy of the chapter joins the URL as
// comma-separated ids ({domain}/chapter-detail/{id1},{id2},…), which
// lets the page step compare them and keep the fullest one; copies of
// a chapter are rare, so single-source chapters keep a canonical
// plain URL.
func chapterEntryURL(
	baseURL *url.URL,
	ids []string,
) string {
	endpoint := *baseURL
	endpoint.Path += chapterDetailPathPrefix + strings.Join(ids, ",")

	return endpoint.String()
}
