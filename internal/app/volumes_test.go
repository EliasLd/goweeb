package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/EliasLd/goweeb/internal/ledger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// testVolume builds a listing-style volume whose entries carry the
// given chapter numbers.
func testVolume(
	t *testing.T,
	number string,
	chapterNumbers ...string,
) sourcetypes.Volume {
	t.Helper()

	entries := make([]sourcetypes.Entry, 0, len(chapterNumbers))

	for _, raw := range chapterNumbers {
		parsed, err := common.ParseChapterNumber(raw)
		if err != nil {
			t.Fatalf("test volume %s: %v", number, err)
		}

		entries = append(entries, sourcetypes.Entry{
			Number: parsed,
			Label:  "Chapter " + raw,
		})
	}

	return sourcetypes.Volume{
		Number:  number,
		Entries: entries,
	}
}

func TestVolumeDigits(t *testing.T) {
	small := []sourcetypes.Volume{
		testVolume(t, "1", "1"),
		testVolume(t, "9", "9"),
	}
	if width := volumeDigits(small); width != 2 {
		t.Errorf("volumeDigits() = %d, want 2", width)
	}

	wide := []sourcetypes.Volume{
		testVolume(t, "1", "1"),
		testVolume(t, "100", "300"),
	}
	if width := volumeDigits(wide); width != 3 {
		t.Errorf("volumeDigits() = %d, want 3", width)
	}
}

func TestVolumeLabel(t *testing.T) {
	cases := []struct {
		number string
		digits int
		want   string
	}{
		{"1", 2, "Vol. 01"},
		{"12", 2, "Vol. 12"},
		{"1", 3, "Vol. 001"},
		{"100", 2, "Vol. 100"},
	}

	for _, tc := range cases {
		if got := volumeLabel(tc.number, tc.digits); got != tc.want {
			t.Errorf(
				"volumeLabel(%q, %d) = %q, want %q",
				tc.number,
				tc.digits,
				got,
				tc.want,
			)
		}
	}
}

func TestLayoutVolumePaths(t *testing.T) {
	cbzSeries := layoutVolumePath(
		LayoutSeries, "/out", "Berserk", "1", 2, ".cbz",
	)
	if want := filepath.Join("/out", "Berserk", "Berserk Vol. 01.cbz"); cbzSeries != want {
		t.Errorf("series bundle = %q, want %q", cbzSeries, want)
	}

	cbzFlat := layoutVolumePath(
		LayoutFlat, "/out", "Berserk", "1", 2, ".cbz",
	)
	if want := filepath.Join("/out", "Berserk_Vol. 01.cbz"); cbzFlat != want {
		t.Errorf("flat bundle = %q, want %q", cbzFlat, want)
	}

	dirSeries := layoutVolumeDir(
		LayoutSeries, "/out", "Berserk", "1", 2,
	)
	if want := filepath.Join("/out", "Berserk", "Vol. 01"); dirSeries != want {
		t.Errorf("series volume dir = %q, want %q", dirSeries, want)
	}

	dirFlat := layoutVolumeDir(
		LayoutFlat, "/out", "Berserk", "1", 2,
	)
	if want := filepath.Join("/out", "Berserk Vol. 01"); dirFlat != want {
		t.Errorf("flat volume dir = %q, want %q", dirFlat, want)
	}

	seriesPages := volumeEntryImageDir(
		dirSeries, LayoutSeries, "Berserk", "10", 3,
	)
	if want := filepath.Join(dirSeries, "Berserk - c010"); seriesPages != want {
		t.Errorf("series chapter folder = %q, want %q", seriesPages, want)
	}

	flatPages := volumeEntryImageDir(
		dirFlat, LayoutFlat, "Berserk", "10", 3,
	)
	if want := filepath.Join(dirFlat, "Chapter 010"); flatPages != want {
		t.Errorf("flat chapter folder = %q, want %q", flatPages, want)
	}
}

func TestFilterVolumesBySelection(t *testing.T) {
	volumes := []sourcetypes.Volume{
		testVolume(t, "10", "20"),
		testVolume(t, "5", "9"),
		testVolume(t, "3", "5"),
		testVolume(t, "not-a-number", "1"),
	}

	selection, _ := ParseRangeExpression("all")

	got := FilterVolumesBySelection(volumes, selection)
	if len(got) != 3 || got[0].Number != "3" || got[2].Number != "10" {
		t.Errorf("all = %v, want 3, 5, 10 in reading order", got)
	}

	selection, _ = ParseRangeExpression("3-5")
	got = FilterVolumesBySelection(volumes, selection)
	if len(got) != 2 || got[0].Number != "3" || got[1].Number != "5" {
		t.Errorf("3-5 = %v, want volumes 3 and 5", got)
	}

	selection, _ = ParseRangeExpression("6-")
	got = FilterVolumesBySelection(volumes, selection)
	if len(got) != 1 || got[0].Number != "10" {
		t.Errorf("6- = %v, want volume 10", got)
	}

	selection, _ = ParseRangeExpression("-2")
	got = FilterVolumesBySelection(volumes, selection)
	if len(got) != 2 || got[0].Number != "5" || got[1].Number != "10" {
		t.Errorf("-2 = %v, want the last two volumes", got)
	}

	selection, _ = ParseRangeExpression("5-7")
	got = FilterVolumesBySelection(volumes, selection)
	if len(got) != 1 || got[0].Number != "5" {
		t.Errorf("5-7 = %v, want volume 5", got)
	}
}

func TestFormatVolumeRanges(t *testing.T) {
	volumes := []sourcetypes.Volume{
		testVolume(t, "5", "9"),
		testVolume(t, "1", "1"),
		testVolume(t, "10", "20"),
		testVolume(t, "2", "2"),
		testVolume(t, "3", "5"),
	}

	if got := FormatVolumeRanges(volumes); got != "1-3, 5, 10" {
		t.Errorf("FormatVolumeRanges() = %q, want %q", got, "1-3, 5, 10")
	}
}

// volumeComplete mirrors what --continue checks on re-run: one
// missing artifact rebuilds the volume, only a fully recorded one
// skips.
func TestVolumeComplete(t *testing.T) {
	dir := t.TempDir()

	led, err := ledger.New(
		filepath.Join(dir, "Test.goweeb.json"),
		ledger.Identity{
			Series:   "Test",
			Provider: "mangadex",
			ScanPath: "en",
			WorkURL:  "https://mangadex.org/manga/x",
			Kind:     ledger.KindVolume,
		},
	)
	if err != nil {
		t.Fatalf("ledger.New() error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "vol.pdf"),
		[]byte("pdf"),
		0o644,
	); err != nil {
		t.Fatalf("failed to write the artifact: %v", err)
	}

	led.Record(ledger.Chapter{
		Number: "1",
		Format: ledger.FormatPDF,
		File:   "vol.pdf",
	})

	volume := testVolume(t, "1", "1")
	if !volumeComplete(&led, ledger.FormatPDF, volume.Entries) {
		t.Errorf("a fully recorded volume should count complete")
	}

	volume = testVolume(t, "1", "1", "2")
	if volumeComplete(&led, ledger.FormatPDF, volume.Entries) {
		t.Errorf("a volume with an unrecorded chapter should stay incomplete")
	}

	if volumeComplete(&led, ledger.FormatPDF, nil) {
		t.Errorf("an empty volume should not count complete")
	}

	if err := os.Remove(filepath.Join(dir, "vol.pdf")); err != nil {
		t.Fatalf("failed to remove the artifact: %v", err)
	}

	volume = testVolume(t, "1", "1")
	if volumeComplete(&led, ledger.FormatPDF, volume.Entries) {
		t.Errorf("a deleted artifact should make the volume incomplete")
	}
}
