package tui

import (
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCatalogResultsOpenMangaSelection(t *testing.T) {
	m := InitialModel()

	next, cmd := Update(
		catalogSearchResultMsg{
			results: []sourcetypes.SearchResult{
				{Title: "Manga A", URL: "/manga/a"},
				{Title: "Manga B", URL: "/manga/b"},
			},
		},
		m,
	)

	if next.State != StateMangaSelection {
		t.Fatalf(
			"state = %v, want StateMangaSelection",
			next.State,
		)
	}

	if cmd != nil {
		t.Error("opening manga selection should not start a command")
	}
}

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

func TestDownloadCompletionReturnsToForm(t *testing.T) {
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
		t.Error("download should no longer be active")
	}

	if next.Cursor != 0 || !next.MangaInput.Focused() {
		t.Error("form should return to the manga input")
	}

	if cmd != nil {
		t.Error("completed download should not schedule another log read")
	}

	if len(next.Logs) == 0 ||
		!strings.Contains(
			next.Logs[len(next.Logs)-1],
			"Download complete!",
		) {
		t.Error("completion message is missing")
	}
}
