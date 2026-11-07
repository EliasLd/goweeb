package metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// anilistSource is the cover source stamp when a cover comes from
// the external database.
const anilistSource = "anilist"

// anilistDefaultEndpoint is the public GraphQL endpoint. Overridable
// for tests.
var anilistDefaultEndpoint = "https://graphql.anilist.co"

// anilistSearchQuery paginates enough of the result list to catch
// the real work even when the relevance ordering surfaces other
// titles first (probe data shows a foreign work ranked above the
// exact one).
const anilistSearchQuery = `query ($search: String) {
  Page(perPage: 20) {
    media(search: $search, type: MANGA) {
      title { romaji english native }
      synonyms
      coverImage { large }
      description
      countryOfOrigin
      startDate { year }
      genres
      staff(sort: RELEVANCE, perPage: 8) {
        edges { role node { name { full } } }
      }
    }
  }
}`

type anilistMatch struct {
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
	Synonyms   []string `json:"synonyms"`
	CoverImage struct {
		Large string `json:"large"`
	} `json:"coverImage"`
	Description     string `json:"description"`
	CountryOfOrigin string `json:"countryOfOrigin"`
	StartDate       struct {
		Year int `json:"year"`
	} `json:"startDate"`
	Genres []string `json:"genres"`
	Staff  struct {
		Edges []anilistStaffEdge `json:"edges"`
	} `json:"staff"`
}

type anilistStaffEdge struct {
	Role string `json:"role"`
	Node struct {
		Name struct {
			Full string `json:"full"`
		} `json:"name"`
	} `json:"node"`
}

func (m anilistMatch) coverImageURL() string {
	return strings.TrimSpace(m.CoverImage.Large)
}

func (m anilistMatch) startDateYear() int {
	return m.StartDate.Year
}

var titleFields = []func(anilistMatch) string{
	func(m anilistMatch) string { return m.Title.English },
	func(m anilistMatch) string { return m.Title.Romaji },
	func(m anilistMatch) string { return m.Title.Native },
}

// lookupAniList finds the one work whose titles match the query
// exactly. Every field — primary titles and synonyms — counts, but
// strictly: several matching candidates mean the site data cannot
// decide a cover, and no cover is served at all. A lookup failure
// means empty metadata, never a blocker.
func lookupAniList(
	title string,
	log *logger.Logger,
) (*anilistMatch, error) {
	body, err := json.Marshal(map[string]any{
		"query":     anilistSearchQuery,
		"variables": map[string]any{"search": title},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to encode the AniList query: %w",
			err,
		)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		anilistDefaultEndpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create the AniList request: %w",
			err,
		)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "goweeb")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	result, err := httpx.ReadAll(client, request, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to reach the AniList API: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"the AniList API answered status %d",
			result.StatusCode,
		)
	}

	var payload struct {
		Data struct {
			Page struct {
				Media []anilistMatch `json:"media"`
			} `json:"page"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(
		result.Body,
		&payload,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode the AniList response: %w",
			err,
		)
	}

	if len(payload.Errors) > 0 {
		return nil, fmt.Errorf(
			"the AniList API rejected the query: %s",
			payload.Errors[0].Message,
		)
	}

	matches := exactTitleMatches(payload.Data.Page.Media, title)

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no exact title match on AniList")

	case 1:
		return matches[0], nil

	default:
		return nil, fmt.Errorf(
			"%d works match that exact title; refusing to guess the cover",
			len(matches),
		)
	}
}

// exactTitleMatches keeps the candidates whose primary titles or
// synonyms equal the normalized query.
func exactTitleMatches(
	media []anilistMatch,
	title string,
) []*anilistMatch {
	query := normalizeTitle(title)

	var matches []*anilistMatch

	for index := range media {
		if matchesTitle(&media[index], query) {
			matches = append(matches, &media[index])
		}
	}

	return matches
}

// matchesTitle reports whether a candidate's primary titles or
// synonyms equal the normalized query exactly.
func matchesTitle(m *anilistMatch, query string) bool {
	for _, field := range titleFields {
		if normalizeTitle(field(*m)) == query {
			return true
		}
	}

	for _, synonym := range m.Synonyms {
		if normalizeTitle(synonym) == query {
			return true
		}
	}

	return false
}

var stripTags = regexp.MustCompile(`<[^>]*>`)

var lineBreakTags = regexp.MustCompile(`(?i)<br\s*/?\s*>`)

// sanitizeDescription turns the HTML-flavored database description
// into plain text: line breaks survive as paragraph breaks, other
// tags are stripped and the double-encoded entities are resolved
// (a known AniList quirk).
func sanitizeDescription(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}

	description = lineBreakTags.ReplaceAllString(description, "\n")
	description = stripTags.ReplaceAllString(description, "")
	description = html.UnescapeString(description)
	description = html.UnescapeString(description)

	var cleaned []string

	for _, line := range strings.Split(description, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

// staff splits the credited people into the author and artist the
// metadata stores: story roles become the author, art roles the
// artist. A combined credit counts for both.
func (m anilistMatch) staff() (string, string) {
	var (
		authorSeen  = make(map[string]bool)
		artistSeen  = make(map[string]bool)
		authorsSeen []string
		artistsSeen []string
	)

	for _, edge := range m.Staff.Edges {
		name := strings.TrimSpace(edge.Node.Name.Full)
		if name == "" {
			continue
		}

		role := strings.ToLower(edge.Role)

		stories := strings.Contains(role, "story")
		draws := strings.Contains(role, "art")

		if stories {
			if !authorSeen[name] {
				authorSeen[name] = true
				authorsSeen = append(authorsSeen, name)
			}
		}

		if draws {
			if !artistSeen[name] {
				artistSeen[name] = true
				artistsSeen = append(artistsSeen, name)
			}
		}
	}

	return strings.Join(authorsSeen, ", "),
		strings.Join(artistsSeen, ", ")
}
