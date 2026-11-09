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

	info := buildComicInfo(
		work,
		entry,
		42,
		sourcetypes.Metadata{},
		"",
	)

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

	info := buildComicInfo(
		work,
		entry,
		100,
		sourcetypes.Metadata{},
		"",
	)

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
	if info := buildComicInfo(
		work,
		entry,
		100,
		sourcetypes.Metadata{},
		"",
	); info.Volume != 0 {
		t.Errorf(
			"Volume = %d, want unset for fractional numbers",
			info.Volume,
		)
	}
}

// Work metadata enriches the schema fields when a lookup produced
// something; LanguageISO comes from the scan path of this run.
func TestBuildComicInfoEnrichedWithWorkMetadata(t *testing.T) {
	work := sourcetypes.Work{
		Title: "Berserk",
		Kind:  sourcetypes.ItemChapter,
	}

	entry := sourcetypes.Entry{
		Number: common.ChapterNumber("10"),
		Label:  "Chapter 10",
	}

	info := buildComicInfo(
		work,
		entry,
		371,
		sourcetypes.Metadata{
			Author:      "Miura Kentarou",
			Artist:      "Studio Gaga",
			Description: "Guts, known as the Black Swordsman...",
			Tags:        []string{"Action", "Drama", "Horror"},
			Year:        1989,
		},
		"pt-BR",
	)

	if info.Writer != "Miura Kentarou" {
		t.Errorf("Writer = %q, want the author", info.Writer)
	}

	if info.Penciller != "Studio Gaga" {
		t.Errorf("Penciller = %q, want the artist", info.Penciller)
	}

	if info.Genre != "Action, Drama, Horror" {
		t.Errorf("Genre = %q, want the tags joined", info.Genre)
	}

	if info.Year != 1989 {
		t.Errorf("Year = %d, want 1989", info.Year)
	}

	if info.LanguageISO != "pt-BR" {
		t.Errorf("LanguageISO = %q, want the scan-path code", info.LanguageISO)
	}

	if info.Description != "Guts, known as the Black Swordsman..." {
		t.Errorf("Description = %q, want the work description", info.Description)
	}
}

// Without metadata the document stays the Phase 1 one: no empty
// descriptive elements may appear.
func TestBuildComicInfoWithoutMetadataStaysLean(t *testing.T) {
	work := sourcetypes.Work{Title: "Berserk", Kind: sourcetypes.ItemChapter}

	entry := sourcetypes.Entry{
		Number: common.ChapterNumber("10"),
		Label:  "Chapter 10",
	}

	info := buildComicInfo(
		work,
		entry,
		10,
		sourcetypes.Metadata{},
		"en",
	)

	if info.Writer != "" || info.Penciller != "" || info.Genre != "" ||
		info.Description != "" {
		t.Errorf(
			"descriptive fields = %q/%q/%q/%q, want all empty without metadata",
			info.Writer,
			info.Penciller,
			info.Genre,
			info.Description,
		)
	}
}
