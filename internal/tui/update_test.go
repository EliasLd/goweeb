package tui

import (
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEntriesResultOpensRangeSelection(t *testing.T) {
	m := InitialModel()

	number, err := common.ParseChapterNumber("1.5")
	if err != nil {
		t.Fatal(err)
	}

	next, cmd := Update(
		entriesResultMsg{
			entries: []sourcetypes.Entry{
				{
					Number: number,
					Label:  "Chapter 1.5",
					URL:    "/chapter/1.5",
				},
			},
		},
		m,
	)

	if next.State != StateRangeSelection {
		t.Fatalf(
			"state = %v, want StateRangeSelection",
			next.State,
		)
	}

	if next.DiscoveredEntries != 1 {
		t.Errorf(
			"discovered entries = %d, want 1",
			next.DiscoveredEntries,
		)
	}

	if !next.RangeInput.Focused() {
		t.Error("range input should be focused")
	}

	if cmd != nil {
		t.Error("opening range selection should not start a command")
	}
}

func TestEmptyRangeDoesNotStartDownload(t *testing.T) {
	m := InitialModel()

	m.State = StateRangeSelection
	m.Cursor = 2
	m.AllCheckbox.Checked = false
	m.RangeInput.SetValue("")

	next, cmd := Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	if next.State != StateRangeSelection {
		t.Error("empty range should keep the range screen open")
	}

	if next.IsDownloading || cmd != nil {
		t.Error("empty range must not start downloading")
	}

	if len(next.Logs) == 0 {
		t.Error("empty range should produce an error message")
	}
}

func TestDownloadCompletionReturnsToForm(
	t *testing.T,
) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"",
	)

	m := InitialModel()

	m.State = StateDownloading
	m.IsDownloading = true
	m.Cursor = 2

	next, cmd := Update(
		logMsg("Finished downloading"),
		m,
	)

	if next.State != StateForm {
		t.Errorf(
			"state = %v, want StateForm",
			next.State,
		)
	}

	if next.IsDownloading {
		t.Error(
			"download should no longer be active",
		)
	}

	if next.Cursor != 0 {
		t.Errorf(
			"cursor = %d, want 0",
			next.Cursor,
		)
	}

	if !next.ScanDirInput.Focused() {
		t.Error(
			"form should return focus to the destination input",
		)
	}

	if cmd == nil {
		t.Error(
			"completed download should schedule a completion notification",
		)
	}

	if len(next.Logs) == 0 ||
		!strings.Contains(
			next.Logs[len(next.Logs)-1],
			"Download complete!",
		) {
		t.Error(
			"completion message is missing",
		)
	}
}

func TestFormNavigationStartsOnProviderWhenOutputLocked(
	t *testing.T,
) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"/home/goweeb/Documents",
	)

	m := InitialModel()

	if m.Cursor != 1 {
		t.Fatalf(
			"cursor = %d, want 1 when output directory is locked",
			m.Cursor,
		)
	}

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyDown,
		},
		m,
	)

	if next.Cursor != 2 {
		t.Errorf(
			"cursor = %d, want 2 after navigating down from provider",
			next.Cursor,
		)
	}
}

func TestFormNavigationCannotEnterLockedOutputDir(
	t *testing.T,
) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"/home/goweeb/Documents",
	)

	m := InitialModel()
	m.Cursor = 1

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyUp,
		},
		m,
	)

	if next.Cursor != 1 {
		t.Errorf(
			"cursor = %d, want 1 when destination is locked",
			next.Cursor,
		)
	}
}

func TestOptionalSettingsOpensFromForm(t *testing.T) {
	m := InitialModel()
	m.Cursor = 2

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if next.State != StateOptionalSettings {
		t.Fatalf(
			"state = %v, want StateOptionalSettings",
			next.State,
		)
	}
}

func TestOptionalSettingsEscapeReturnsToForm(t *testing.T) {
	m := InitialModel()
	m.State = StateOptionalSettings

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEsc,
		},
		m,
	)

	if next.State != StateForm {
		t.Fatalf(
			"state = %v, want StateForm",
			next.State,
		)
	}
}

func TestOptionalSettingsToggleEbookFriendly(t *testing.T) {
	m := InitialModel()
	m.State = StateOptionalSettings
	m.OptionalCursor = 0

	initial := m.EbookCheckbox.Checked

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeySpace,
		},
		m,
	)

	if next.EbookCheckbox.Checked == initial {
		t.Fatal(
			"ebook-friendly checkbox was not toggled",
		)
	}
}
