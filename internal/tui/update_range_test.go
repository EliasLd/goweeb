package tui

import (
	"testing"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func rangeTestModel() Model {
	volumes := []sourcetypes.Volume{
		{Number: "1", Entries: []sourcetypes.Entry{
			{Number: common.ChapterNumber("1"), Label: "Chapter 1"},
			{Number: common.ChapterNumber("2"), Label: "Chapter 2"},
		}},
		{Number: "2", Entries: []sourcetypes.Entry{
			{Number: common.ChapterNumber("3"), Label: "Chapter 3"},
		}},
	}

	m := Model{
		State:               StateRangeSelection,
		Cursor:              3,
		Logs:                []string{},
		RangeInput:          textinput.New(),
		AllCheckbox:         Checkbox{Label: "Download all chapters."},
		VolumesCheckbox:     Checkbox{Label: "Download as volumes."},
		DiscoveredEntryList: []sourcetypes.Entry{},
		DiscoveredVolumes:   volumes,
	}

	m.RangeInput.SetValue("1-2")

	return m
}

func enterKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func downKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyDown}
}

// Submitting with the volumes checkbox on selects volumes and leaves
// the chapter range untouched.
func TestRangeSubmitSelectsVolumes(t *testing.T) {
	m := rangeTestModel()
	m.VolumesCheckbox.Checked = true

	next, _ := handleRangeUpdate(enterKey(), m)

	got := next.SelectedVolumeRange

	if len(got.Ranges) != 1 ||
		got.Ranges[0].Start.String() != "1" ||
		got.Ranges[0].End.String() != "2" {
		t.Fatalf("volume selection = %+v, want range 1-2", got)
	}

	if got.All {
		t.Errorf("volume selection must not set All")
	}

	if next.SelectedRange.All || len(next.SelectedRange.Ranges) > 0 {
		t.Errorf("chapter range must stay empty in volume mode")
	}

	if !next.IsDownloading || next.State != StateDownloading {
		t.Errorf("submit must start the download, state = %v", next.State)
	}

	if len(next.Logs) == 0 ||
		next.Logs[len(next.Logs)-2] != "Selected 2 volume(s): 1-2" {
		t.Errorf("missing volume selection log: %v", next.Logs)
	}
}

// Submitting without the volume checkbox selects chapters and clears
// any stale volume range, even while the listing is present.
func TestRangeSubmitSelectsChaptersWithListingPresent(t *testing.T) {
	m := rangeTestModel()
	m.VolumesCheckbox.Checked = false
	m.SelectedVolumeRange = app.RangeSelection{All: true}
	m.DiscoveredEntryList = []sourcetypes.Entry{
		{Number: common.ChapterNumber("1"), Label: "Chapter 1"},
		{Number: common.ChapterNumber("2"), Label: "Chapter 2"},
	}

	next, _ := handleRangeUpdate(enterKey(), m)

	if len(next.SelectedRange.Ranges) != 1 ||
		next.SelectedRange.Ranges[0].Start.String() != "1" ||
		next.SelectedRange.Ranges[0].End.String() != "2" {
		t.Fatalf("chapter selection = %+v, want range 1-2", next.SelectedRange)
	}

	if next.SelectedVolumeRange.All || len(next.SelectedVolumeRange.Ranges) > 0 {
		t.Errorf("stale volume selection must be cleared")
	}
}

// Cursor navigation stops at the Download button; the volumes row
// only exists while the work lists volumes.
func TestRangeCursorStopsAtDownload(t *testing.T) {
	m := rangeTestModel()
	m.Cursor = 3

	next, _ := handleRangeUpdate(downKey(), m)
	if next.Cursor != 3 {
		t.Errorf("cursor = %d, want 3 (listing present)", next.Cursor)
	}

	next.VolumesCheckbox.Checked = true
	next, _ = handleRangeUpdate(downKey(), next)
	if next.Cursor != 3 {
		t.Errorf("cursor = %d, want 3 (Download is the last row)", next.Cursor)
	}

	// Without a listing there is no volumes row at all.
	withoutVolumes := rangeTestModel()
	withoutVolumes.DiscoveredVolumes = nil
	withoutVolumes.Cursor = 2
	withoutVolumes.DiscoveredEntryList = []sourcetypes.Entry{
		{Number: common.ChapterNumber("1"), Label: "Chapter 1"},
		{Number: common.ChapterNumber("2"), Label: "Chapter 2"},
	}

	next, _ = handleRangeUpdate(downKey(), withoutVolumes)
	if next.Cursor != 2 {
		t.Errorf("cursor = %d, want 2 without a volume listing", next.Cursor)
	}

	next, _ = handleRangeUpdate(enterKey(), next)
	if len(next.SelectedRange.Ranges) != 1 ||
		next.SelectedRange.Ranges[0].Start.String() != "1" {
		t.Errorf("submit without a listing = %+v, want chapters 1-2", next.SelectedRange)
	}

	if next.SelectedVolumeRange.All || len(next.SelectedVolumeRange.Ranges) > 0 {
		t.Errorf("volume selection must stay empty without a listing")
	}
}
