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
