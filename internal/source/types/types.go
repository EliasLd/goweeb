package types

import (
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

type ItemKind int

const (
	ItemChapter ItemKind = iota
	ItemVolume
)

type SearchResult struct {
	Title string
	URL   string

	// CoverURL is set when the provider's search payload happens to
	// carry the work's cover image. It is optional metadata: absent
	// is normal, present is a free win for search screens.
	CoverURL string
}

type Entry struct {
	Number common.ChapterNumber
	Label  string
	URL    string
}

type Work struct {
	Title string
	Kind  ItemKind
}

// Metadata carries what a provider knows about a work beyond its
// title: the people and facts readers recognize, plus the cover URL
// a future interface can render as a card. Every field is optional;
// the cover mirrors what CoverURL carries on search results when the
// detail payload is the one exposing it.
type Metadata struct {
	Author           string   `json:"author,omitempty"`
	Artist           string   `json:"artist,omitempty"`
	CoverURL         string   `json:"coverUrl,omitempty"`
	CoverSource      string   `json:"coverSource,omitempty"`
	Description      string   `json:"description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	OriginalLanguage string   `json:"originalLanguage,omitempty"`
	Year             int      `json:"year,omitempty"`
}

// Known reports whether any metadata survived a lookup.
func (m Metadata) Known() bool {
	return m.Author != "" ||
		m.Artist != "" ||
		m.CoverURL != "" ||
		m.Description != "" ||
		len(m.Tags) > 0 ||
		m.OriginalLanguage != "" ||
		m.Year != 0
}

// Optional capability for providers able to describe a work beyond
// its title: author, cover, description, tags. Providers without a
// metadata payload simply do not implement it, and the download flow
// degrades to empty metadata.
type MetadataConfigurer interface {
	WorkMetadata(workURL string, log *logger.Logger) (Metadata, error)
}

// Optional capability for providers that require specific HTTP headers
// when downloading page images.
type ImageRequestConfigurer interface {
	ImageRequestHeaders() map[string]string
}

type Provider interface {
	Name() string
	Search(query string, log *logger.Logger) ([]SearchResult, error)

	// Returns scan versions when the source exposes them (e.g. mangadex, anime-sama...).
	// For sources without variants, return one implicit default item.
	ListScanPaths(workURL string, log *logger.Logger) ([]common.SelectableItem, error)

	// Returns chapters/volumes using the selected scan path (if any).
	ListEntries(workURL string, scanPath string, log *logger.Logger) (Work, []Entry, error)

	GetPageImageURLs(entryURL string, log *logger.Logger) ([]string, error)
}
