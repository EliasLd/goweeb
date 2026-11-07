package app

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func TestOutputFormat(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want ledger.Format
	}{{
		name: "cbz",
		opts: Options{CBZ: true},
		want: ledger.FormatCBZ,
	}, {
		name: "ebook-friendly folders",
		opts: Options{EbookFriendly: true},
		want: ledger.FormatFolder,
	}, {
		name: "default pdf",
		opts: Options{},
		want: ledger.FormatPDF,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := outputFormat(tt.opts); got != tt.want {
				t.Errorf(
					"outputFormat() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestLedgerKind(t *testing.T) {
	if got := ledgerKind(sourcetypes.ItemVolume); got != ledger.KindVolume {
		t.Errorf(
			"ledgerKind(volume) = %q, want %q",
			got,
			ledger.KindVolume,
		)
	}

	if got := ledgerKind(sourcetypes.ItemChapter); got != ledger.KindChapter {
		t.Errorf(
			"ledgerKind(chapter) = %q, want %q",
			got,
			ledger.KindChapter,
		)
	}
}

// writeTestCBZFile creates a small, valid CBZ at path.
func writeTestCBZFile(t *testing.T, path string) {
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

// testEntry builds an entry with a chapter number literal.
func testEntry(number string) sourcetypes.Entry {
	return sourcetypes.Entry{
		Number: common.ChapterNumber(number),
		Label:  "Chapter " + number,
		URL:    "https://example.com/chapter/" + number,
	}
}

func TestSkipDownloaded(t *testing.T) {
	dir := t.TempDir()

	// Two chapters present on disk, one intact and one truncated
	// mid-write, and a third that the ledger never recorded.
	writeTestCBZFile(t, filepath.Join(dir, "One Piece - c010.cbz"))

	full := readTestFileBytes(t, filepath.Join(dir, "One Piece - c010.cbz"))

	truncatedPath := filepath.Join(dir, "One Piece - c012.cbz")

	if err := os.WriteFile(
		truncatedPath,
		full[:len(full)/2],
		0o644,
	); err != nil {
		t.Fatalf("failed to write the truncated CBZ: %v", err)
	}

	led := openTestAppLedger(t, dir)
	led.Record(ledger.Chapter{
		Number: common.ChapterNumber("10"),
		Format: ledger.FormatCBZ,
		File:   "One Piece - c010.cbz",
	})
	led.Record(ledger.Chapter{
		Number: common.ChapterNumber("12"),
		Format: ledger.FormatCBZ,
		File:   "One Piece - c012.cbz",
	})

	entries := []sourcetypes.Entry{
		testEntry("10"),
		testEntry("11"),
		testEntry("12"),
	}

	var logs bytes.Buffer

	log := logger.New(&logs, logger.LevelDebug)

	kept := skipDownloaded(
		&led,
		ledger.FormatCBZ,
		entries,
		log,
	)

	if got := len(kept); got != 2 {
		t.Fatalf(
			"kept %d entries, want 2 (chapter 10 is usable)",
			got,
		)
	}

	if kept[0].Number != "11" || kept[1].Number != "12" {
		t.Errorf(
			"kept = [%s, %s], want [11, 12]",
			kept[0].Number,
			kept[1].Number,
		)
	}

	// The skip explains itself, including the truncated archive.
	if !bytes.Contains(logs.Bytes(), []byte("already downloaded")) {
		t.Error("expected an already-downloaded log line")
	}

	if !bytes.Contains(logs.Bytes(), []byte("recorded but")) {
		t.Error("expected a debug line for the unusable archive")
	}
}

func TestSkipDownloadedDifferentFormatRestarts(t *testing.T) {
	dir := t.TempDir()
	led := openTestAppLedger(t, dir)

	// Chapter 10 exists as a PDF: a CBZ request must redo it.
	if err := os.WriteFile(
		filepath.Join(dir, "One Piece_10.pdf"),
		[]byte("fake pdf"),
		0o644,
	); err != nil {
		t.Fatalf("failed to write the PDF: %v", err)
	}

	led.Record(ledger.Chapter{
		Number: common.ChapterNumber("10"),
		Format: ledger.FormatPDF,
		File:   "One Piece_10.pdf",
	})

	kept := skipDownloaded(
		&led,
		ledger.FormatCBZ,
		[]sourcetypes.Entry{testEntry("10")},
		logger.New(&bytes.Buffer{}, logger.LevelInfo),
	)

	if len(kept) != 1 {
		t.Error(
			"expected the chapter to download again in another format",
		)
	}
}

// openTestAppLedger opens a ledger like the download flow does.
func openTestAppLedger(
	t *testing.T,
	dir string,
) ledger.Ledger {
	t.Helper()

	led, err := ledger.New(
		filepath.Join(dir, "One Piece.goweeb.json"),
		ledger.Identity{
			Series:   "One Piece",
			Provider: "mangadex",
			WorkURL:  "https://mangadex.org/title/abc",
			Kind:     ledger.KindChapter,
		},
	)
	if err != nil {
		t.Fatalf("ledger.New() error: %v", err)
	}

	return led
}

func readTestFileBytes(
	t *testing.T,
	path string,
) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	return data
}
