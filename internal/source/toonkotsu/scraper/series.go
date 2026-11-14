package toonkotsuscraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

// seriesPath serves one work by its code: its record, the full inline
// chapter list (every chapter, one response) and its color editions.
const seriesPath = "/cdn/v1/series/"

// seriesRecord is the work itself.
type seriesRecord struct {
	Title string `json:"title"`
}

// rawID is a chapter id the payloads deliver as either a JSON number
// or a string; both travel through the pipeline as text.
type rawID string

func (id *rawID) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)

	if string(trimmed) == "null" {
		*id = ""

		return nil
	}

	if len(trimmed) > 0 && trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}

		*id = rawID(text)

		return nil
	}

	var number json.Number
	if err := json.Unmarshal(trimmed, &number); err != nil {
		return err
	}

	*id = rawID(number.String())

	return nil
}

// altRow is one alternate upload of the same chapter number.
type altRow struct {
	ID rawID `json:"id"`
}

// chapterRow is one row of the inline chapter list. n is nullable and
// may be fractional; json.Number keeps its exact text, null decodes
// as nil. vol is the site's tag for the volume the chapter belongs
// to, also nullable.
type chapterRow struct {
	ID    rawID        `json:"id"`
	N     *json.Number `json:"n"`
	Vol   string       `json:"vol"`
	Label string       `json:"label"`
	Alts  []altRow     `json:"alts"`
}

// editionRecord is one variant publication (e.g. an official colored
// relaunch): it has its own code and its own chapter set. The count
// is the site's own figure for the label.
type editionRecord struct {
	Code     string `json:"code"`
	Title    string `json:"title"`
	Chapters int    `json:"chapters"`
}

type seriesResponse struct {
	Series   seriesRecord    `json:"series"`
	Chapters []chapterRow    `json:"chapters"`
	Editions []editionRecord `json:"editions"`
}

// GetSeries fetches one series payload by code.
func GetSeries(
	domain string,
	code string,
	log *logger.Logger,
) (seriesResponse, error) {
	body, err := callAPI(
		domain,
		"series",
		seriesPath+code,
		nil,
		log,
	)
	if err != nil {
		return seriesResponse{}, err
	}

	var response seriesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return seriesResponse{}, fmt.Errorf(
			"failed to decode toonkotsu series %s: %w",
			code,
			err,
		)
	}

	return response, nil
}

// WorkCodeFromURL extracts the work code from a toonkotsu title URL.
//
// Expected shape: {domain}/title/{code}-{slug}; the code is the
// prefix before the first hyphen, and the slug may itself contain
// hyphens.
func WorkCodeFromURL(workURL string) (string, error) {
	workURL = strings.TrimSpace(workURL)

	parsed, err := url.Parse(workURL)
	if err != nil {
		return "", fmt.Errorf(
			"invalid toonkotsu title URL %q: %w",
			workURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")

	// Expected: ["", "title", "{code}-{slug}"]
	segments := strings.Split(path, "/")

	if len(segments) != 3 || segments[1] != "title" {
		return "", fmt.Errorf(
			"unexpected toonkotsu title URL: %q",
			workURL,
		)
	}

	code := strings.TrimSpace(
		strings.SplitN(segments[2], "-", 2)[0],
	)

	if code == "" {
		return "", fmt.Errorf(
			"unexpected toonkotsu title URL: %q",
			workURL,
		)
	}

	return code, nil
}
