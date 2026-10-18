package tui

import (
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
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
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if next.State != StateRangeSelection {
		t.Error(
			"empty range should keep the range screen open",
		)
	}

	if next.IsDownloading {
		t.Error(
			"empty range must not start downloading",
		)
	}

	if len(next.Logs) == 0 {
		t.Error(
			"empty range should produce an error message",
		)
	}

	if cmd == nil {
		t.Error(
			"empty range should schedule an error notification",
		)
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

func TestFormNavigationStartsOnOptionsWhenOutputLocked(
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
			"cursor = %d, want 2 after navigating down from options",
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
	m.Cursor = 1

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

func TestOptionalSettingsToggleCBZ(t *testing.T) {
	m := InitialModel()
	m.State = StateOptionalSettings
	m.OptionalCursor = 1

	initial := m.CBZCheckbox.Checked

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeySpace,
		},
		m,
	)

	if next.CBZCheckbox.Checked == initial {
		t.Fatal(
			"CBZ checkbox was not toggled",
		)
	}
}

func TestOptionalSettingsModesAreMutuallyExclusive(t *testing.T) {
	m := InitialModel()
	m.State = StateOptionalSettings

	// Checking CBZ unchecks ebook-friendly.
	m.OptionalCursor = 1

	checkedCBZ, _ := Update(
		tea.KeyMsg{
			Type: tea.KeySpace,
		},
		m,
	)

	if !checkedCBZ.CBZCheckbox.Checked {
		t.Fatal("CBZ checkbox should be checked")
	}

	// The user moves to the ebook-friendly setting and checks it.
	m = checkedCBZ
	m.OptionalCursor = 0

	checkedEbook, _ := Update(
		tea.KeyMsg{
			Type: tea.KeySpace,
		},
		m,
	)

	if !checkedEbook.EbookCheckbox.Checked {
		t.Error("ebook-friendly checkbox should be checked")
	}

	if checkedEbook.CBZCheckbox.Checked {
		t.Error(
			"checking ebook-friendly should uncheck CBZ",
		)
	}

	// Unchecking ebook-friendly keeps CBZ untouched.
	m = checkedEbook

	unchecked, _ := Update(
		tea.KeyMsg{
			Type: tea.KeySpace,
		},
		m,
	)

	if unchecked.EbookCheckbox.Checked {
		t.Error("ebook-friendly should be unchecked again")
	}

	if unchecked.CBZCheckbox.Checked {
		t.Error("CBZ should remain unchecked")
	}

	// Checking CBZ unchecks ebook-friendly in the other direction.
	m = unchecked
	m.EbookCheckbox.Checked = true
	m.OptionalCursor = 1

	final, _ := Update(
		tea.KeyMsg{
			Type: tea.KeySpace,
		},
		m,
	)

	if !final.CBZCheckbox.Checked {
		t.Error("CBZ checkbox should be checked")
	}

	if final.EbookCheckbox.Checked {
		t.Error(
			"checking CBZ should uncheck ebook-friendly",
		)
	}
}

func TestDownloadSpinnerUpdatesWhileDownloading(
	t *testing.T,
) {
	m := InitialModel()

	m.IsDownloading = true
	m.State = StateDownloading

	next, cmd := Update(
		spinner.TickMsg{},
		m,
	)

	if cmd == nil {
		t.Error(
			"active download spinner should schedule another tick",
		)
	}

	if next.DownloadSpinner.View() == "" {
		t.Error(
			"download spinner should render while downloading",
		)
	}
}

func TestDownloadSpinnerStopsWhenDownloadIsInactive(
	t *testing.T,
) {
	m := InitialModel()

	m.IsDownloading = false

	_, cmd := Update(
		spinner.TickMsg{},
		m,
	)

	if cmd != nil {
		t.Error(
			"inactive download spinner should not schedule another tick",
		)
	}
}

func TestProviderSelectionOpensFromSearch(t *testing.T) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlP,
		},
		m,
	)

	if next.State != StateProviderSelection {
		t.Fatalf(
			"state = %v, want StateProviderSelection",
			next.State,
		)
	}

	// No provider has been selected so far: the all-providers
	// entry must be the checked one.
	if len(next.ProviderSelectionModel.Options) == 0 {
		t.Fatal("provider options are empty")
	}

	allOption := next.ProviderSelectionModel.Options[0]

	if allOption.ID != "" || !allOption.Checkbox.Checked {
		t.Errorf(
			"first option = %q (checked %v), want the all-providers option checked",
			allOption.ID,
			allOption.Checkbox.Checked,
		)
	}
}

func TestProviderSelectionCancelReturnsToSearch(t *testing.T) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m, _ = Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlP,
		},
		m,
	)

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEsc,
		},
		m,
	)

	if next.State != StateInteractiveSearch {
		t.Fatalf(
			"state = %v, want StateInteractiveSearch",
			next.State,
		)
	}
}

func TestProviderSelectionConfirmRunsSearchWithFilter(t *testing.T) {
	m := InitialModel()
	m.DestinationDir = app.DefaultTUIScanDir()
	m.State = StateInteractiveSearch
	m.SearchInput.SetValue("blue lock")

	m, _ = Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlP,
		},
		m,
	)

	// Pick a specific provider: manga option index 1.
	m.ProviderSelectionModel.Cursor = 1

	m, _ = Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if m.ProviderSelectionModel.SelectedID == "" {
		t.Fatal("selecting a provider should record its id")
	}

	// Confirm.
	next, cmd := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if next.State != StateInteractiveSearch {
		t.Fatalf(
			"state = %v, want StateInteractiveSearch",
			next.State,
		)
	}

	if next.SelectedProvider !=
		next.ProviderSelectionModel.SelectedID {
		t.Errorf(
			"selected provider = %q, want %q",
			next.SelectedProvider,
			next.ProviderSelectionModel.SelectedID,
		)
	}

	// Confirming the filter must refresh the search with the query
	// typed before opening the provider screen.
	if cmd == nil {
		t.Fatal("confirm should start a new search command")
	}

	if !next.SearchInFlight {
		t.Error("the refreshed search should be in flight")
	}

	if next.SearchProvidersPending != 1 {
		t.Errorf(
			"pending providers = %d, want 1",
			next.SearchProvidersPending,
		)
	}

	if len(next.SearchList.Items()) != 0 {
		t.Error("results should be cleared for the new filter")
	}
}

func TestInteractiveSearchFocusReachesProviderRow(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch
	m.SearchInput.SetValue("blue lock")
	m.SearchResultsQuery = "blue lock"
	m.SearchHasSearched = true

	_ = m.SearchList.SetItems(
		[]list.Item{
			SelectionItem{Label: "Blue Lock", Value: "u"},
		},
	)

	m = setInteractiveSearchFocus(m, searchFocusInput)

	// Up from the input focuses the provider row.
	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyUp,
		},
		m,
	)

	if next.SearchFocus != searchFocusProvider {
		t.Fatalf(
			"focus = %v, want searchFocusProvider",
			next.SearchFocus,
		)
	}

	// Down from the provider row returns to the input.
	next, _ = Update(
		tea.KeyMsg{
			Type: tea.KeyDown,
		},
		next,
	)

	if next.SearchFocus != searchFocusInput {
		t.Fatalf(
			"focus = %v, want searchFocusInput",
			next.SearchFocus,
		)
	}

	// Down at the bottom of the results wraps to the provider row.
	resultsFocusModel :=
		setInteractiveSearchFocus(
			next,
			searchFocusResults,
		)

	wrapped, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyDown,
		},
		resultsFocusModel,
	)

	if wrapped.SearchFocus != searchFocusProvider {
		t.Fatalf(
			"focus = %v, want searchFocusProvider",
			wrapped.SearchFocus,
		)
	}

	// Enter on the provider row opens the provider screen.
	opened, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		wrapped,
	)

	if opened.State != StateProviderSelection {
		t.Fatalf(
			"state = %v, want StateProviderSelection",
			opened.State,
		)
	}
}

func TestProviderSelectionCanSelectAllProvidersAgain(
	t *testing.T,
) {
	m := InitialModel()
	m.DestinationDir = app.DefaultTUIScanDir()
	m.State = StateInteractiveSearch

	// Simulate a previous choice of a single provider.
	m.SelectedProvider = "mangadex"
	m.SelectedProviderLabel = "MangaDex (Multilingual)"

	m, _ = Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlP,
		},
		m,
	)

	if m.ProviderSelectionModel.SelectedID != "mangadex" {
		t.Fatalf(
			"selected id = %q, want mangadex",
			m.ProviderSelectionModel.SelectedID,
		)
	}

	// The cursor starts on the all-providers entry: select it.
	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if next.ProviderSelectionModel.SelectedID != "" {
		t.Fatalf(
			"selected id = %q, want the empty all-providers id",
			next.ProviderSelectionModel.SelectedID,
		)
	}

	// …and confirm the all-providers filter.
	confirmed, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		next,
	)

	if confirmed.State != StateInteractiveSearch {
		t.Fatalf(
			"state = %v, want StateInteractiveSearch",
			confirmed.State,
		)
	}

	if confirmed.SelectedProvider != "" {
		t.Errorf(
			"selected provider = %q, want empty (all providers)",
			confirmed.SelectedProvider,
		)
	}

	if confirmed.SelectedProviderLabel != "All providers" {
		t.Errorf(
			"selected provider label = %q, want All providers",
			confirmed.SelectedProviderLabel,
		)
	}
}
