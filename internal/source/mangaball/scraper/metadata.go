package mangaballscraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// MangaDetail is what the title detail endpoint yields: the name the
// entry listing needs plus the site metadata published alongside it.
type MangaDetail struct {
	Name     string
	Metadata sourcetypes.Metadata
}

// coverPayload mirrors the image object every title row carries.
// The site's own cover path has no reachable CDN host, so the usable
// mirrors are the crawler CDNs it lists: the mangadex upload one
// first (hotlinkable, probed), then the mangaupdates one.
type coverPayload struct {
	CDNMangadex    string `json:"cdn_mangadex"`
	CDNMangaupdate string `json:"cdn_mangaupdate"`
}

// coverFromImage picks the hotlinkable cover address of a title
// image object, preferring the mangadex mirror.
func coverFromImage(image coverPayload) string {
	if url := strings.TrimSpace(image.CDNMangadex); url != "" {
		return url
	}

	return strings.TrimSpace(image.CDNMangaupdate)
}

// lenientStrings decodes an optional description: a single string or
// an array of strings. Anything else decodes to empty instead of
// failing — this endpoint also feeds the chapter listing, and
// unexpected metadata shapes must never break it.
type lenientStrings []string

func (l *lenientStrings) UnmarshalJSON(data []byte) error {
	switch {
	case len(data) == 0 || string(data) == "null":
		*l = nil

	case data[0] == '"':
		var single string
		if err := json.Unmarshal(data, &single); err != nil {
			*l = nil

			return nil
		}

		*l = []string{single}

	case data[0] == '[':
		var items []string
		if err := json.Unmarshal(data, &items); err != nil {
			*l = nil

			return nil
		}

		*l = items
	}

	return nil
}

// lenientNameNames decodes an array of rows into their trimmed,
// deduplicated "name" values, dropping rows that carry none. The
// metadata arrays ride the same endpoint as the chapter listing, so
// a malformed row means "no data", never a failure.
func lenientNameNames(raw json.RawMessage) []string {
	var rows []nameRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}

	seen := make(map[string]bool)
	out := make([]string, 0, len(rows))

	for _, row := range rows {
		name := strings.TrimSpace(row.Name)

		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		out = append(out, name)
	}

	return out
}

type nameRow struct {
	Name string `json:"name"`
}

// GetTitleDetail fetches one title's detail document and derives the
// work metadata from it: author names (the API repeats rows),
// description paragraphs, tags, publication year, original language
// and the hotlinkable cover mirror. No separate artist collection
// exists on the site, so the artist field stays empty.
func GetTitleDetail(
	domain string,
	titleID string,
	log *logger.Logger,
) (MangaDetail, error) {
	body, err := apiPost(
		domain,
		"title detail",
		titleDetailPath,
		map[string]string{"title_id": titleID},
		log,
	)
	if err != nil {
		return MangaDetail{}, err
	}

	var response struct {
		Data struct {
			Name          string          `json:"name"`
			Image         coverPayload    `json:"image"`
			Description   lenientStrings  `json:"description"`
			OriginalLang  string          `json:"originalLanguage"`
			DatePublished json.RawMessage `json:"date_published"`
			Tags          json.RawMessage `json:"tags"`
			Author        json.RawMessage `json:"author"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return MangaDetail{}, fmt.Errorf(
			"failed to decode MangaBall title detail: %w",
			err,
		)
	}

	name := strings.TrimSpace(response.Data.Name)
	if name == "" {
		return MangaDetail{}, fmt.Errorf(
			"MangaBall title %s has no name",
			titleID,
		)
	}

	metadata := sourcetypes.Metadata{
		Author:           strings.Join(lenientNameNames(response.Data.Author), ", "),
		CoverURL:         coverFromImage(response.Data.Image),
		Description:      strings.Join(response.Data.Description, "\n"),
		Tags:             lenientNameNames(response.Data.Tags),
		OriginalLanguage: strings.TrimSpace(response.Data.OriginalLang),
		Year:             publishedYear(response.Data.DatePublished),
	}

	return MangaDetail{Name: name, Metadata: metadata}, nil
}

// GetTitleName fetches a title's display name. The chapter listing
// carries no title, so this is a light dedicated call.
func GetTitleName(
	domain string,
	titleID string,
	log *logger.Logger,
) (string, error) {
	detail, err := GetTitleDetail(domain, titleID, log)
	if err != nil {
		return "", err
	}

	return detail.Name, nil
}

// publishedYear parses the publication year best-effort: the field
// is a number today, and any other shape simply means no year is
// known rather than a decode failure.
func publishedYear(raw json.RawMessage) int {
	var value json.Number
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0
	}

	year, err := value.Int64()
	if err != nil {
		return 0
	}

	return int(year)
}
