package app

import (
	"testing"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func TestBuildComicInfoForChapter(t *testing.T) {
	work := sourcetypes.Work{
		Title: "One Piece",
		Kind:  sourcetypes.ItemChapter,
	}

	entry := sourcetypes.Entry{
		Number: common.ChapterNumber("6.5"),
		Label:  "Chapter 6.5",
	}

	info := buildComicInfo(work, entry, 42)

	if info.Series != "One Piece" {
		t.Errorf(
			"Series = %q, want %q",
			info.Series,
			"One Piece",
		)
	}

	if info.Number != "6.5" {
		t.Errorf(
			"Number = %q, want %q",
			info.Number,
			"6.5",
		)
	}

	if info.Count != 42 {
		t.Errorf(
			"Count = %d, want 42",
			info.Count,
		)
	}

	if info.Title != "Chapter 6.5" {
		t.Errorf(
			"Title = %q, want %q",
			info.Title,
			"Chapter 6.5",
		)
	}

	if info.Volume != 0 {
		t.Errorf(
			"Volume = %d, want unset for chapters",
			info.Volume,
		)
	}

	if info.Manga != "Yes" {
		t.Errorf(
			"Manga = %q, want %q",
			info.Manga,
			"Yes",
		)
	}
}

func TestBuildComicInfoForVolume(t *testing.T) {
	work := sourcetypes.Work{
		Title: "One Piece",
		Kind:  sourcetypes.ItemVolume,
	}

	entry := sourcetypes.Entry{
		Number: common.ChapterNumber("12"),
		Label:  "Volume 12",
	}

	info := buildComicInfo(work, entry, 100)

	// Volume-kind works carry their number both as the issue number
	// and as the schema's Volume count.
	if info.Volume != 12 {
		t.Errorf(
			"Volume = %d, want 12",
			info.Volume,
		)
	}

	if info.Number != "12" {
		t.Errorf(
			"Number = %q, want %q",
			info.Number,
			"12",
		)
	}
}

func TestBuildComicInfoForFractionalVolume(t *testing.T) {
	work := sourcetypes.Work{
		Title: "One Piece",
		Kind:  sourcetypes.ItemVolume,
	}

	entry := sourcetypes.Entry{
		Number: common.ChapterNumber("6.5"),
		Label:  "Volume 6.5",
	}

	// The schema only accepts integer volumes; the value stays in
	// Number alone.
	if info := buildComicInfo(work, entry, 100); info.Volume != 0 {
		t.Errorf(
			"Volume = %d, want unset for fractional numbers",
			info.Volume,
		)
	}
}
