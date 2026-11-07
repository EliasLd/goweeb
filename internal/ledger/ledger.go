// Package ledger stores per-work download records and reconciles
// them against the filesystem.
//
// The manifest knows which work an output folder belongs to (series,
// provider, scan path, work URL) and caches the fact that a chapter
// was produced in some format; the files on disk remain the arbiter
// of what is actually present. Records are verified at read time, so
// files deleted or truncated outside the ledger — by a user, a web
// UI, or an interrupted run — count as not downloaded and get
// produced again instead of being silently skipped.
package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// Version is the schema version written into every manifest.
const Version = 1

// Format is the output mode an artifact was produced in. One entry
// can exist in several formats at once; each is tracked separately.
type Format string

const (
	FormatCBZ    Format = "cbz"
	FormatPDF    Format = "pdf"
	FormatFolder Format = "folder"
)

// Kind mirrors the kind of entries a provider lists for a work.
type Kind string

const (
	KindChapter Kind = "chapter"
	KindVolume  Kind = "volume"
)

// ErrNotRecorded is reported when the manifest holds no artifact at
// all for the requested chapter number and format. Callers check it
// with errors.Is to tell "never downloaded" apart from "recorded but
// unusable on disk".
var ErrNotRecorded = errors.New("nothing recorded in the ledger")

// Chapter is one produced artifact, tied to the entry it came from.
type Chapter struct {
	Number       common.ChapterNumber `json:"number"`
	Label        string               `json:"label"`
	URL          string               `json:"url"`
	Format       Format               `json:"format"`
	File         string               `json:"file"`
	DownloadedAt string               `json:"downloadedAt"` // RFC3339, UTC
}

// Manifest is the JSON document written on disk, one per work. The
// header carries the work identity the filesystem cannot infer and
// the library will rely on later to re-resolve the chapter list; the
// chapter list is a cache that FindArtifact reconciles on disk.
//
// The optional metadata block is the work's description cache: it is
// additive (works written by older versions load the same), and the
// coverFile field names the cached cover image next to the document.
type Manifest struct {
	Version   int                   `json:"version"`
	Series    string                `json:"series"`
	Provider  string                `json:"provider"`
	ScanPath  string                `json:"scanPath"`
	WorkURL   string                `json:"workUrl"`
	Kind      Kind                  `json:"kind"`
	Metadata  *sourcetypes.Metadata `json:"metadata,omitempty"`
	CoverFile string                `json:"coverFile,omitempty"`
	Chapters  []Chapter             `json:"chapters"`
}

// Identity describes the work a ledger belongs to. It is stamped
// over the header on every open so a renamed series or a moved
// provider stays up to date.
type Identity struct {
	Series   string
	Provider string
	ScanPath string
	WorkURL  string
	Kind     Kind
}

// Ledger is an open manifest: records merge in memory as chapters
// complete and Save replaces the file atomically.
type Ledger struct {
	path     string
	manifest Manifest
}

// New loads the manifest at path for the work described by identity.
//
// A missing file yields an empty ledger. A corrupted manifest, or one
// written by a newer goweeb, also yields an empty ledger together
// with an error the caller may surface: the ledger heals itself as
// chapters are recorded again, and rebuilding it deletes nothing.
func New(path string, identity Identity) (Ledger, error) {
	manifest, loadErr := load(path)

	manifest.Version = Version
	manifest.Series = identity.Series
	manifest.Provider = identity.Provider
	manifest.ScanPath = identity.ScanPath
	manifest.WorkURL = identity.WorkURL
	manifest.Kind = identity.Kind
	manifest.Chapters = nonNil(manifest.Chapters)

	return Ledger{path: path, manifest: manifest}, loadErr
}

// load reads a manifest from disk. A missing file is not an error:
// it only means nothing was downloaded through a ledger yet.
func load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Manifest{}, nil
	}

	if err != nil {
		return Manifest{}, fmt.Errorf(
			"failed to read %s: %w",
			path,
			err,
		)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf(
			"failed to parse %s: %w",
			path,
			err,
		)
	}

	if manifest.Version > Version {
		return Manifest{}, fmt.Errorf(
			"manifest version %d is newer than the supported version %d",
			manifest.Version,
			Version,
		)
	}

	return manifest, nil
}

func nonNil(chapters []Chapter) []Chapter {
	if chapters == nil {
		return []Chapter{}
	}

	return chapters
}

// SetMetadata stores the work's description cache. Only lookups
// that produced something are kept: empty metadata leaves the block
// absent, so a later richer lookup can still win. Reports whether
// the manifest changed.
func (l *Ledger) SetMetadata(metadata sourcetypes.Metadata) bool {
	if !metadata.Known() {
		return false
	}

	if l.manifest.Metadata != nil &&
		metadataEqual(*l.manifest.Metadata, metadata) {
		return false
	}

	l.manifest.Metadata = &metadata

	return true
}

// metadataEqual compares the description caches field by field.
func metadataEqual(
	a sourcetypes.Metadata,
	b sourcetypes.Metadata,
) bool {
	return a.Author == b.Author &&
		a.Artist == b.Artist &&
		a.CoverURL == b.CoverURL &&
		a.Description == b.Description &&
		a.OriginalLanguage == b.OriginalLanguage &&
		a.Year == b.Year &&
		slices.Equal(a.Tags, b.Tags)
}

// SetCoverFile records the cached cover image file name, or clears
// the record when the name is empty. Reports whether the manifest
// changed.
func (l *Ledger) SetCoverFile(name string) bool {
	name = filepath.Base(strings.TrimSpace(name))

	if name == l.manifest.CoverFile {
		return false
	}

	l.manifest.CoverFile = name

	return true
}

// Metadata returns the stored description cache, nil when unknown.
func (l *Ledger) Metadata() *sourcetypes.Metadata {
	return l.manifest.Metadata
}

// CoverFile returns the recorded cover image file name — a plain
// basename next to the manifest — empty when none was recorded.
func (l *Ledger) CoverFile() string {
	return l.manifest.CoverFile
}

// Record merges one completed artifact: an entry for the same number
// and format is replaced, otherwise the chapter is appended. The
// list stays sorted by number so the file reads in order and
// repeated runs produce a stable document.
func (l *Ledger) Record(chapter Chapter) {
	chapters := l.manifest.Chapters
	replaced := false

	for i, existing := range chapters {
		if existing.Number == chapter.Number &&
			existing.Format == chapter.Format {
			chapters[i] = chapter
			replaced = true

			break
		}
	}

	if !replaced {
		chapters = append(chapters, chapter)
	}

	sort.SliceStable(chapters, func(i, j int) bool {
		if order := chapters[i].Number.Compare(
			chapters[j].Number,
		); order != 0 {
			return order < 0
		}

		return chapters[i].Format < chapters[j].Format
	})

	l.manifest.Chapters = chapters
}

// Save replaces the manifest file.
//
// The document is fully written to a temporary file that then
// renames over the destination, so a crash can never leave a
// truncated ledger behind on the common filesystems.
func (l *Ledger) Save() error {
	l.manifest.Version = Version

	data, err := json.MarshalIndent(l.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"failed to encode the manifest: %w",
			err,
		)
	}

	// A trailing newline keeps the ledger diff-friendly.
	data = append(data, '\n')

	// The artifacts and the manifest share one directory in every
	// layout, so Chapter files can stay plain basenames.
	const manifestPerm = 0o644

	tmp := l.path + ".tmp"

	if err := os.WriteFile(tmp, data, manifestPerm); err != nil {
		return fmt.Errorf(
			"failed to write %s: %w",
			tmp,
			err,
		)
	}

	if err := os.Rename(tmp, l.path); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf(
			"failed to replace %s: %w",
			l.path,
			err,
		)
	}

	return nil
}
