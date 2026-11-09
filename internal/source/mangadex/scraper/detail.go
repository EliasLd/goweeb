package mangadexscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// coverCDNBase serves work cover images. The upload CDN is its own
// host, separate from any API mirror the user configured, so the
// name stays fixed.
const coverCDNBase = "https://uploads.mangadex.org/covers"

// MangaDetail is everything the detail endpoint says about a work:
// what the entry listing needs (name, numbering quirk) plus the
// metadata the site publishes (authors, cover, description, tags).
type MangaDetail struct {
	Name              string
	ResetsOnNewVolume bool
	Metadata          sourcetypes.Metadata
}

type tagElement struct {
	Attributes struct {
		Name map[string]string `json:"name"`
	} `json:"attributes"`
}

type mangaDetailAttributes struct {
	Title             map[string]string `json:"title"`
	Description       map[string]string `json:"description"`
	Tags              []tagElement      `json:"tags"`
	Year              int               `json:"year"`
	Status            string            `json:"status"`
	OriginalLanguage  string            `json:"originalLanguage"`
	ResetsOnNewVolume bool              `json:"chapterNumbersResetOnNewVolume"`
}

type relationshipEntry struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes *struct {
		Name     string `json:"name"`
		FileName string `json:"fileName"`
	} `json:"attributes"`
}

type mangaDetailResponse struct {
	Result string          `json:"result"`
	Data   mangaDetailData `json:"data"`
}

type mangaDetailData struct {
	ID           string                `json:"id"`
	Attributes   mangaDetailAttributes `json:"attributes"`
	Relationship []relationshipEntry   `json:"relationships"`
}

// GetMangaDetail fetches everything one endpoint says about a work.
// The entry listing consumes the name and the numbering quirk, and
// the metadata service consumes the rest. The includes parameters
// make the API inline the author and artist names into the
// relationships, and the cover image into a cover_art relationship.
func GetMangaDetail(
	domain string,
	workURL string,
	log *logger.Logger,
) (*MangaDetail, error) {
	mangaID, err := extractMangaID(workURL)
	if err != nil {
		return nil, err
	}

	baseURL, err := domainURL(domain)
	if err != nil {
		return nil, err
	}

	detailURL := baseURL.JoinPath("manga", mangaID)

	detailURL.RawQuery = url.Values{
		"includes[]": {"cover_art", "author", "artist"},
	}.Encode()

	log.Debug(
		"Fetching MangaDex manga detail: %s\n",
		detailURL.String(),
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		detailURL.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaDex manga detail request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "goweeb")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaDex manga detail: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"MangaDex manga detail returned status: %d",
			result.StatusCode,
		)
	}

	var response mangaDetailResponse
	if err := json.Unmarshal(
		result.Body,
		&response,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode MangaDex manga detail: %w",
			err,
		)
	}

	if response.Result != "ok" {
		return nil, fmt.Errorf(
			"MangaDex returned unexpected result: %s",
			response.Result,
		)
	}

	return detailFromResponse(response.Data), nil
}

// plainDescription cleans the formatting mangadex mirrors inside its
// descriptions: separator lines (`___`) and bold markers ride along
// when official sources and staff notes are copied in. The lines that
// hold nothing are paragraph structure and stay.
func plainDescription(description string) string {
	cleaned := make([]string, 0, strings.Count(description, "\n")+1)

	for _, line := range strings.Split(description, "\n") {
		// An empty line is a paragraph break; only non-empty lines
		// can be decoration.
		if line != "" && isSeparatorLine(line) {
			continue
		}

		cleaned = append(cleaned, strings.ReplaceAll(line, "**", ""))
	}

	// Only the edges of the whole text are trimmed; interior blank
	// lines are the paragraphs the reader should keep seeing.
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// isSeparatorLine reports whether a line is pure decoration, such as
// a horizontal rule the source mirrors verbatim.
func isSeparatorLine(line string) bool {
	return strings.Trim(line, " _-*~") == ""
}

// detailFromResponse maps the decoded payload. Relationship arrays
// arrive in API order; author and artist names are deduplicated in
// that order, and the artist list drops every name already listed as
// an author so the two stay distinct in the metadata.
func detailFromResponse(
	data mangaDetailData,
) *MangaDetail {
	authors, artists, coverFile := scanRelationships(
		data.Relationship,
	)

	metadata := sourcetypes.Metadata{
		Author:           joinNames(authors),
		Artist:           joinNames(removeSet(artists, authors)),
		CoverURL:         coverURL(coverCDNBase+"/"+data.ID, coverFile),
		Description:      plainDescription(selectLocalized(data.Attributes.Description)),
		Tags:             tagNames(data.Attributes.Tags),
		OriginalLanguage: data.Attributes.OriginalLanguage,
		Year:             data.Attributes.Year,
	}

	name := selectLocalized(data.Attributes.Title)
	if name == "" {
		name = data.ID
	}

	return &MangaDetail{
		Name:              name,
		ResetsOnNewVolume: data.Attributes.ResetsOnNewVolume,
		Metadata:          metadata,
	}
}

func scanRelationships(
	relations []relationshipEntry,
) (authors []string, artists []string, coverFile string) {
	for _, relation := range relations {
		// The name only travels on person relationships; the cover
		// relationship carries a file name instead.
		name := ""
		if relation.Attributes != nil {
			name = strings.TrimSpace(relation.Attributes.Name)
		}

		switch relation.Type {
		case "author":
			if name != "" {
				authors = append(authors, name)
			}

		case "artist":
			if name != "" {
				artists = append(artists, name)
			}

		case "cover_art":
			if coverFile == "" && relation.Attributes != nil {
				coverFile = strings.TrimSpace(
					relation.Attributes.FileName,
				)
			}
		}
	}

	return firstDistinct(authors),
		firstDistinct(artists),
		coverFile
}

func tagNames(tagElems []tagElement) []string {
	names := make([]string, 0, len(tagElems))

	for _, tagElem := range tagElems {
		if name := selectLocalized(tagElem.Attributes.Name); name != "" {
			names = append(names, name)
		}
	}

	return names
}

func firstDistinct(names []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(names))

	for _, name := range names {
		if seen[name] {
			continue
		}

		seen[name] = true
		out = append(out, name)
	}

	return out
}

func removeSet(names []string, exclude []string) []string {
	excluded := make(map[string]bool)
	for _, name := range exclude {
		excluded[name] = true
	}

	kept := make([]string, 0, len(names))

	for _, name := range names {
		if !excluded[name] {
			kept = append(kept, name)
		}
	}

	return kept
}

// joinNames renders a list of people as the comma-joined string the
// metadata stores.
func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

// coverURL concatenates a cover base and a file name into the
// hotlinkable cover address, empty when one half is missing.
func coverURL(base, fileName string) string {
	if base == "" || fileName == "" {
		return ""
	}

	return fmt.Sprintf("%s/%s", base, fileName)
}
