package ledger

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/source/common"
)

const testCBZPerm = 0o644

func testIdentity() Identity {
	return Identity{
		Series:   "One Piece",
		Provider: "mangadex",
		ScanPath: "en",
		WorkURL:  "https://mangadex.org/title/abc",
		Kind:     KindChapter,
	}
}

// writeTestCBZ creates a small, valid CBZ at path.
func writeTestCBZ(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create CBZ: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)

	entry, err := writer.Create("001.jpg")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}

	if _, err := entry.Write([]byte("fake image")); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to finalize zip: %v", err)
	}
}

// openLedgerIn prepares a ledger over a fresh manifest inside dir.
func openLedgerIn(t *testing.T, dir string) Ledger {
	t.Helper()

	led, err := New(
		filepath.Join(dir, "One Piece.goweeb.json"),
		testIdentity(),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return led
}

func openTestLedger(t *testing.T) Ledger {
	t.Helper()

	return openLedgerIn(t, t.TempDir())
}

func TestNewMissingFileIsEmpty(t *testing.T) {
	led := openTestLedger(t)

	if got := len(led.manifest.Chapters); got != 0 {
		t.Fatalf("found %d chapters, want an empty ledger", got)
	}

	if led.manifest.Provider != "mangadex" {
		t.Errorf(
			"Provider = %q, want the stamped identity",
			led.manifest.Provider,
		)
	}
}

func TestSaveWritesExpectedDocument(t *testing.T) {
	led := openTestLedger(t)

	led.Record(Chapter{
		Number:       common.ChapterNumber("6.5"),
		Label:        "Chapter 6.5",
		URL:          "https://mangadex.org/chapter/1",
		Format:       FormatCBZ,
		File:         "One Piece - c006.5.cbz",
		DownloadedAt: "2026-11-07T12:00:00Z",
	})

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(led.path)
	if err != nil {
		t.Fatalf("failed to read the manifest: %v", err)
	}

	// The document shape is the compatibility contract: pin it.
	want := `{
  "version": 1,
  "series": "One Piece",
  "provider": "mangadex",
  "scanPath": "en",
  "workUrl": "https://mangadex.org/title/abc",
  "kind": "chapter",
  "chapters": [
    {
      "number": "6.5",
      "label": "Chapter 6.5",
      "url": "https://mangadex.org/chapter/1",
      "format": "cbz",
      "file": "One Piece - c006.5.cbz",
      "downloadedAt": "2026-11-07T12:00:00Z"
    }
  ]
}
`

	if string(data) != want {
		t.Errorf(
			"manifest =\n%s\nwant\n%s",
			data,
			want,
		)
	}
}

func TestSaveLeavesNoTemporaryFile(t *testing.T) {
	led := openTestLedger(t)

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(led.path + ".tmp"); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Error("expected the temporary file to be renamed away")
	}
}

func TestSaveWithoutRecordsStillValid(t *testing.T) {
	led := openTestLedger(t)

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(led.path)
	if err != nil {
		t.Fatalf("failed to read the manifest: %v", err)
	}

	if !bytes.Contains(data, []byte(`"version": 1`)) {
		t.Error("expected the schema version to be stamped")
	}
}

func TestRecordMergesAndSorts(t *testing.T) {
	led := openTestLedger(t)

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatCBZ,
	})

	led.Record(Chapter{
		Number: common.ChapterNumber("6.5"),
		Format: FormatCBZ,
	})

	// Recording the same number and format again replaces the
	// entry instead of duplicating it.
	led.Record(Chapter{
		Number:       common.ChapterNumber("10"),
		Format:       FormatCBZ,
		DownloadedAt: "2026-11-07T12:00:00Z",
	})

	want := []string{"6.5", "10"}

	if got := len(led.manifest.Chapters); got != 2 {
		t.Fatalf("found %d chapters, want 2", got)
	}

	for i, number := range want {
		if got := string(
			led.manifest.Chapters[i].Number,
		); got != number {
			t.Errorf(
				"chapter %d = %q, want %q",
				i,
				got,
				number,
			)
		}
	}

	if led.manifest.Chapters[1].DownloadedAt != "2026-11-07T12:00:00Z" {
		t.Error("expected the recording to replace the entry")
	}
}

func TestRecordKeepsFormatsApart(t *testing.T) {
	led := openTestLedger(t)

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatPDF,
	})

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatCBZ,
	})

	if got := len(led.manifest.Chapters); got != 2 {
		t.Fatalf(
			"found %d chapters, want one entry per format",
			got,
		)
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	led := openLedgerIn(t, dir)

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		URL:    "https://mangadex.org/chapter/2",
		Format: FormatCBZ,
		File:   "One Piece - c010.cbz",
	})

	writeTestCBZ(t, filepath.Join(dir, "One Piece - c010.cbz"))

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reloaded, err := New(led.path, testIdentity())
	if err != nil {
		t.Fatalf("reload New() error: %v", err)
	}

	if err := reloaded.FindArtifact(
		common.ChapterNumber("10"),
		FormatCBZ,
	); err != nil {
		t.Errorf(
			"FindArtifact() after reload: %v",
			err,
		)
	}
}

func TestFindArtifactNotRecorded(t *testing.T) {
	led := openTestLedger(t)

	err := led.FindArtifact(
		common.ChapterNumber("10"),
		FormatCBZ,
	)
	if !errors.Is(err, ErrNotRecorded) {
		t.Errorf(
			"FindArtifact() = %v, want ErrNotRecorded",
			err,
		)
	}
}

func TestFindArtifactValidCBZ(t *testing.T) {
	dir := t.TempDir()
	led := openLedgerIn(t, dir)

	writeTestCBZ(t, filepath.Join(dir, "One Piece - c010.cbz"))

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatCBZ,
		File:   "One Piece - c010.cbz",
	})

	if err := led.FindArtifact(
		common.ChapterNumber("10"),
		FormatCBZ,
	); err != nil {
		t.Errorf(
			"FindArtifact() = %v, want a usable archive",
			err,
		)
	}
}

func TestFindArtifactTruncatedCBZ(t *testing.T) {
	dir := t.TempDir()
	led := openLedgerIn(t, dir)

	cbzPath := filepath.Join(dir, "One Piece - c010.cbz")

	writeTestCBZ(t, cbzPath)

	// Simulate a download interrupted mid-write.
	full, err := os.ReadFile(cbzPath)
	if err != nil {
		t.Fatalf("failed to read the CBZ: %v", err)
	}

	if err := os.WriteFile(
		cbzPath,
		full[:len(full)/2],
		testCBZPerm,
	); err != nil {
		t.Fatalf("failed to truncate the CBZ: %v", err)
	}

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatCBZ,
		File:   "One Piece - c010.cbz",
	})

	err = led.FindArtifact(
		common.ChapterNumber("10"),
		FormatCBZ,
	)
	if err == nil {
		t.Fatal("expected a truncated archive to be unusable")
	}

	if errors.Is(err, ErrNotRecorded) {
		t.Error(
			"expected the reason to describe the artifact, not its absence",
		)
	}

	if !strings.Contains(err.Error(), "archive") {
		t.Errorf(
			"unexpected reason: %v",
			err,
		)
	}
}

func TestFindArtifactMissingFile(t *testing.T) {
	led := openTestLedger(t)

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatPDF,
		File:   "One Piece_10.pdf",
	})

	err := led.FindArtifact(
		common.ChapterNumber("10"),
		FormatPDF,
	)
	if err == nil || errors.Is(err, ErrNotRecorded) {
		t.Fatalf(
			"FindArtifact() = %v, want a file-missing reason",
			err,
		)
	}
}

func TestFindArtifactEmptyFile(t *testing.T) {
	dir := t.TempDir()
	led := openLedgerIn(t, dir)

	pdfPath := filepath.Join(dir, "One Piece_10.pdf")

	if err := os.WriteFile(pdfPath, nil, 0o644); err != nil {
		t.Fatalf("failed to write an empty PDF: %v", err)
	}

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatPDF,
		File:   "One Piece_10.pdf",
	})

	if err := led.FindArtifact(
		common.ChapterNumber("10"),
		FormatPDF,
	); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf(
			"FindArtifact() = %v, want an empty-file reason",
			err,
		)
	}
}

func TestFindArtifactFolder(t *testing.T) {
	dir := t.TempDir()
	led := openLedgerIn(t, dir)

	folder := filepath.Join(dir, "One Piece - c010")

	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatFolder,
		File:   "One Piece - c010",
	})

	// An image folder counts as usable only once it holds pages.
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatalf("failed to create the folder: %v", err)
	}

	if err := led.FindArtifact(
		common.ChapterNumber("10"),
		FormatFolder,
	); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf(
			"FindArtifact() = %v, want an empty-folder reason",
			err,
		)
	}

	if err := os.WriteFile(
		filepath.Join(folder, "001.jpg"),
		[]byte("x"),
		0o644,
	); err != nil {
		t.Fatalf("failed to write a page: %v", err)
	}

	if err := led.FindArtifact(
		common.ChapterNumber("10"),
		FormatFolder,
	); err != nil {
		t.Errorf(
			"FindArtifact() = %v, want a usable folder",
			err,
		)
	}
}

func TestNewCorruptedManifestHeals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "One Piece.goweeb.json")

	if err := os.WriteFile(
		path,
		[]byte("{not json"),
		0o644,
	); err != nil {
		t.Fatalf("failed to write a corrupt manifest: %v", err)
	}

	led, loadErr := New(path, testIdentity())
	if loadErr == nil {
		t.Fatal("expected an error for a corrupted manifest")
	}

	if got := len(led.manifest.Chapters); got != 0 {
		t.Error("expected the corrupted manifest to be ignored")
	}

	// Recording the first chapter again heals the file, and the
	// reloaded ledger passes verification.
	led.Record(Chapter{
		Number: common.ChapterNumber("10"),
		Format: FormatPDF,
		File:   "One Piece_10.pdf",
	})

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reloaded, err := New(path, testIdentity())
	if err != nil {
		t.Fatalf("reload New() error: %v", err)
	}

	if got := len(reloaded.manifest.Chapters); got != 1 {
		t.Errorf(
			"reloaded ledger holds %d chapters, want 1",
			got,
		)
	}
}

func TestNewRejectsNewerVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "One Piece.goweeb.json")

	if err := os.WriteFile(
		path,
		[]byte(`{"version":42}`),
		0o644,
	); err != nil {
		t.Fatalf("failed to write the manifest: %v", err)
	}

	if _, err := New(path, testIdentity()); err == nil {
		t.Fatal("expected an error for a newer manifest version")
	}
}
