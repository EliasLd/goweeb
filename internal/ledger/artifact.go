package ledger

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/EliasLd/goweeb/internal/source/common"
)

// FindArtifact reports whether a usable artifact of the given format
// is on disk for a chapter number.
//
// A nil error means the chapter can be skipped. A non-nil error
// explains what is wrong: anything recorded but not usable (missing
// file, empty file, unreadable archive) counts as not downloaded and
// will be produced again.
func (l *Ledger) FindArtifact(
	number common.ChapterNumber,
	format Format,
) error {
	for _, chapter := range l.manifest.Chapters {
		if chapter.Number != number || chapter.Format != format {
			continue
		}

		return checkArtifact(
			filepath.Join(filepath.Dir(l.path), chapter.File),
			format,
		)
	}

	return ErrNotRecorded
}

// checkArtifact validates one recorded artifact. Archives only count
// when they actually open: a download interrupted mid-write leaves
// bytes on disk that no reader would accept.
func checkArtifact(path string, format Format) error {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file is missing: %w", err)
	}

	if stat.Size() == 0 {
		return errors.New("file is empty")
	}

	switch format {
	case FormatFolder:
		return checkFolder(stat, path)

	case FormatCBZ:
		return checkCBZ(stat, path)
	}

	return nil
}

func checkFolder(stat os.FileInfo, path string) error {
	if !stat.IsDir() {
		return errors.New("not a folder")
	}

	children, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("folder unreadable: %w", err)
	}

	if len(children) == 0 {
		return errors.New("folder is empty")
	}

	return nil
}

func checkCBZ(stat os.FileInfo, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("file unreadable: %w", err)
	}
	defer file.Close()

	if _, err := zip.NewReader(file, stat.Size()); err != nil {
		return fmt.Errorf("not a valid CBZ archive: %w", err)
	}

	return nil
}
