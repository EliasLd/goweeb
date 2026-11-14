package toonkotsuscraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// seriesMetadataRecord carries only the metadata fields of the
// series record. Listings decode their own struct, so a payload shape
// drifting here cannot take the catalogue walk down with it.
type seriesMetadataRecord struct {
	Authors          []string `json:"authors"`
	Artists          []string `json:"artists"`
	Description      string   `json:"description"`
	Cover            string   `json:"cover"`
	Tags             []string `json:"tags"`
	OriginalLanguage string   `json:"originalLanguage"`
	Year             int      `json:"year"`
}

// seriesMetadata wraps the record the same way the payload does:
// the metadata fields sit nested under the series key.
type seriesMetadata struct {
	Series seriesMetadataRecord `json:"series"`
}

// GetWorkMetadata reads one work's free metadata off its series
// payload: the people, facts and description readers recognize, plus
// the cover the metadata flow renders as a card. The cover source is
// stamped by the app's metadata flow itself.
func GetWorkMetadata(
	domain string,
	code string,
	log *logger.Logger,
) (sourcetypes.Metadata, error) {
	body, err := callAPI(
		domain,
		"series metadata",
		seriesPath+code,
		nil,
		log,
	)
	if err != nil {
		return sourcetypes.Metadata{}, err
	}

	var payload seriesMetadata
	if err := json.Unmarshal(body, &payload); err != nil {
		return sourcetypes.Metadata{}, fmt.Errorf(
			"failed to decode toonkotsu series metadata %s: %w",
			code,
			err,
		)
	}

	return sourcetypes.Metadata{
		Author:           joinNames(payload.Series.Authors),
		Artist:           joinNames(payload.Series.Artists),
		CoverURL:         absoluteURL(domain, payload.Series.Cover),
		Description:      strings.TrimSpace(payload.Series.Description),
		Tags:             cleanTagList(payload.Series.Tags),
		OriginalLanguage: strings.TrimSpace(payload.Series.OriginalLanguage),
		Year:             payload.Series.Year,
	}, nil
}

// joinNames renders the payload's people arrays the way the other
// providers present theirs: one comma-joined line.
func joinNames(names []string) string {
	cleaned := make([]string, 0, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			cleaned = append(cleaned, name)
		}
	}

	return strings.Join(cleaned, ", ")
}

// cleanTagList trims each tag and drops empties; an all-empty list
// carries nothing.
func cleanTagList(tags []string) []string {
	cleaned := make([]string, 0, len(tags))

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleaned = append(cleaned, tag)
		}
	}

	if len(cleaned) == 0 {
		return nil
	}

	return cleaned
}
