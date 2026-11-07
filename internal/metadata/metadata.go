// Package metadata resolves what a work is beyond its title:
// author, cover, description, tags. It orchestrates a per-field
// fallback ladder and never blocks or fails what it serves: any
// lookup problem degrades to empty metadata with a logged reason.
//
// Cover ladder, in order:
//  1. the work's own provider (site data only),
//  2. other goweeb providers' search rows, matched by exact title —
//     first-party data, no deep crawl,
//  3. an external database (AniList), attached only on one exact
//     and unambiguous title match, opt-out,
//  4. nothing — consumers (web UI, e-reader feeds) render a
//     generated placeholder themselves when no cover file exists.
//
// A wrong cover is worse than no cover: ambiguous matches are
// refused, nothing is first-hit-selected among external candidates.
package metadata

import (
	"errors"
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// ErrNoMetadata reports that the ladder produced nothing usable. It
// wraps the layers' reasons.
var ErrNoMetadata = errors.New("no work metadata found")

// Service consults the providers and the external database. The
// cross-provider list is optional: nil disables second-layer
// lookups. IncludeExternal gates the external database layer.
type Service struct {
	CrossProviders  []sourcetypes.Provider
	IncludeExternal bool
}

// Request describes one work lookup.
type Request struct {
	Provider  sourcetypes.Provider // the work's own provider
	WorkTitle string               // display title, the matching key
	WorkURL   string               // provider-canonical work URL
	Log       *logger.Logger
}

func (r Request) log() *logger.Logger {
	if r.Log == nil {
		return nil
	}

	return r.Log
}

// Fetch resolves the work metadata through the ladder. The returned
// error is ErrNoMetadata (wrapping the layers' reasons) when nothing
// was found — never a download-stopping failure.
func (s Service) Fetch(
	req Request,
) (sourcetypes.Metadata, error) {
	trace := reasonCollector{log: req.log()}

	resolved := s.fromPrimaryProvider(req, &trace)

	if resolved.CoverURL == "" {
		s.fromCrossProviders(req, &resolved, &trace)
	}

	if s.IncludeExternal {
		s.fromExternal(req, &resolved, &trace)
	}

	if !resolved.Known() {
		return sourcetypes.Metadata{}, fmt.Errorf(
			"%w: %s",
			ErrNoMetadata,
			trace.joined(),
		)
	}

	return resolved, nil
}

// reasonCollector records why each layer produced nothing: every
// reason lands in the debug log, and the joined list travels with
// the ErrNoMetadata error.
type reasonCollector struct {
	log   *logger.Logger
	parts []string
}

func (r *reasonCollector) addf(
	format string,
	args ...any,
) {
	reason := fmt.Sprintf(format, args...)

	if r.log != nil {
		r.log.Debug("%s\n", reason)
	}

	r.parts = append(r.parts, reason)
}

func (r *reasonCollector) joined() string {
	return strings.Join(r.parts, "; ")
}

// fromPrimaryProvider asks the work's own provider, the only layer
// that may contribute every metadata field at once.
func (s Service) fromPrimaryProvider(
	req Request,
	trace *reasonCollector,
) sourcetypes.Metadata {
	configurer, implements := req.Provider.(sourcetypes.MetadataConfigurer)
	if !implements {
		trace.addf(
			"provider %s does not expose work metadata",
			req.Provider.Name(),
		)

		return sourcetypes.Metadata{}
	}

	resolved, err := configurer.WorkMetadata(req.WorkURL, req.log())
	if err != nil {
		trace.addf(
			"provider %s lookup failed: %v",
			req.Provider.Name(),
			err,
		)

		return sourcetypes.Metadata{}
	}

	if !resolved.Known() {
		trace.addf(
			"provider %s knows nothing about this work",
			req.Provider.Name(),
		)

		return sourcetypes.Metadata{}
	}

	// First-party cover: stamp the source.
	resolved.CoverSource = req.Provider.Name()

	return resolved
}

// fromCrossProviders fills a missing cover from other goweeb
// providers: a search row is accepted only when its title matches
// the work exactly, and the first covering provider ends the loop.
func (s Service) fromCrossProviders(
	req Request,
	resolved *sourcetypes.Metadata,
	trace *reasonCollector,
) {
	query := normalizeTitle(req.WorkTitle)

	for _, cross := range s.CrossProviders {
		if cross.Name() == req.Provider.Name() {
			continue
		}

		rows, err := cross.Search(req.WorkTitle, req.log())
		if err != nil {
			trace.addf(
				"cross lookup on %s failed: %v",
				cross.Name(),
				err,
			)

			continue
		}

		for _, row := range rows {
			if normalizeTitle(row.Title) != query {
				continue
			}

			if row.CoverURL == "" {
				trace.addf(
					"%s matches the title but carries no cover",
					cross.Name(),
				)

				break
			}

			resolved.CoverURL = row.CoverURL
			resolved.CoverSource = "cross:" + cross.Name()

			trace.addf(
				"cover matched on %s by exact title",
				cross.Name(),
			)

			return
		}
	}
}

// fromExternal consults the external database only under strict
// conditions: the cover is attached on one exact and unambiguous
// title match, and the text fields are filled wholesale only when
// the work carries no provider text at all. Nothing is ever merged
// partially.
func (s Service) fromExternal(
	req Request,
	resolved *sourcetypes.Metadata,
	trace *reasonCollector,
) {
	match, err := lookupAniList(req.WorkTitle, req.log())
	if err != nil {
		trace.addf("external database lookup: %v", err)

		return
	}

	if resolved.CoverURL == "" {
		resolved.CoverURL = match.coverImageURL()
		resolved.CoverSource = anilistSource
	}

	if !hasOwnerText(*resolved) {
		owner, artist := match.staff()

		resolved.Author = owner
		resolved.Artist = artist
		resolved.Description = sanitizeDescription(match.Description)
		resolved.Tags = match.Genres
		resolved.OriginalLanguage = match.CountryOfOrigin
		resolved.Year = match.startDateYear()
	}
}

// hasOwnerText reports whether any descriptive field is already
// populated by provider data (the cover is not a text field).
func hasOwnerText(m sourcetypes.Metadata) bool {
	return m.Author != "" ||
		m.Artist != "" ||
		m.Description != "" ||
		len(m.Tags) > 0 ||
		m.OriginalLanguage != "" ||
		m.Year != 0
}
