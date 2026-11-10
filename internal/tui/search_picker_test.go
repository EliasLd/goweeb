package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// pickerTestModel returns a search screen showing one grouped
// result, ready for Enter, plus the group row itself.
func pickerTestModel(t *testing.T) (Model, SelectionItem) {
	t.Helper()

	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("berserk")
	m.SearchResultsQuery = "berserk"
	m.SearchHasSearched = true

	group := SelectionItem{
		Label:         "Berserk",
		Value:         "mangadex/Berserk",
		Provider:      "mangadex",
		ProviderLabel: "MangaDex (Multilingual)",
		Members: []SelectionItem{
			{
				Label:         "Berserk",
				Value:         "mangadex/Berserk",
				Provider:      "mangadex",
				ProviderLabel: "MangaDex (Multilingual)",
			},
			{
				Label:         "BERSERK",
				Value:         "atsumoe/BERSERK",
				Provider:      "atsumoe",
				ProviderLabel: "Atsumoe (English)",
			},
		},
	}

	if err := m.SearchList.SetItems(
		[]list.Item{group},
	); err != nil {
		t.Fatalf("set items: %v", err)
	}

	m = setInteractiveSearchFocus(
		m,
		searchFocusResults,
	)

	return m, group
}

func TestSearchPickerOpensOnGroupRow(t *testing.T) {
	m, _ := pickerTestModel(t)

	next, _ := Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	if !next.SearchPickerVisible {
		t.Fatal("enter on a group row should open the picker")
	}

	if next.SearchPickerLabel != "Berserk" {
		t.Errorf(
			"picker label = %q, want %q",
			next.SearchPickerLabel,
			"Berserk",
		)
	}

	if len(next.SearchPickerItems) != 2 {
		t.Fatalf(
			"picker items = %d, want 2",
			len(next.SearchPickerItems),
		)
	}

	if next.SearchPickerCursor != 0 {
		t.Errorf(
			"picker cursor = %d, want 0",
			next.SearchPickerCursor,
		)
	}

	if next.State != StateInteractiveSearch {
		t.Errorf(
			"state = %v, want StateInteractiveSearch",
			next.State,
		)
	}

	if got := next.SearchList.Index(); got != 0 {
		t.Errorf(
			"list cursor = %d, want 0 (list untouched by the opening)",
			got,
		)
	}
}

func TestSearchPickerStaysClosedOnPlainRow(t *testing.T) {
	m, _ := pickerTestModel(t)

	m.SearchList.Select(0)

	// Replace the grouped row with a plain one: Enter selects it
	// directly and the picker never opens.
	_ = m.SearchList.SetItems(
		[]list.Item{
			SelectionItem{
				Label:         "Berserk Memorial",
				Value:         "atsumoe/memorial",
				Provider:      "atsumoe",
				ProviderLabel: "Atsumoe (English)",
			},
		},
	)

	m = setInteractiveSearchFocus(
		m,
		searchFocusResults,
	)

	next, _ := Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	if next.SearchPickerVisible {
		t.Error("enter on a plain row must not open the picker")
	}

	if next.State != StateForm {
		t.Fatalf(
			"state = %v, want StateForm",
			next.State,
		)
	}
}

func TestSearchPickerEscRestoresList(t *testing.T) {
	m, _ := pickerTestModel(t)

	opened, _ := Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	// Move the cursor inside the picker, then close it.
	opened, _ = handleSearchPickerKey(
		tea.KeyMsg{Type: tea.KeyDown},
		opened,
	)

	if opened.SearchPickerCursor != 1 {
		t.Fatalf(
			"picker cursor = %d, want 1 after moving",
			opened.SearchPickerCursor,
		)
	}

	closed, _ := Update(
		tea.KeyMsg{Type: tea.KeyEsc},
		opened,
	)

	if closed.SearchPickerVisible {
		t.Fatal("esc should close the picker")
	}

	items := closed.SearchList.Items()

	if len(items) != 1 {
		t.Fatalf(
			"list items = %d, want 1 (unchanged)",
			len(items),
		)
	}

	item, _ := items[0].(SelectionItem)

	if item.Label != "Berserk" || len(item.Members) != 2 {
		t.Errorf(
			"list row = %q with %d members, want the group untouched",
			item.Label,
			len(item.Members),
		)
	}

	if got := closed.SearchList.Index(); got != 0 {
		t.Errorf(
			"list cursor = %d, want 0 (restored exactly)",
			got,
		)
	}

	if closed.State != StateInteractiveSearch {
		t.Errorf(
			"state = %v, want StateInteractiveSearch",
			closed.State,
		)
	}

	if closed.SearchPickerItems != nil ||
		closed.SearchPickerCursor != 0 {
		t.Error("picker state should reset on close")
	}
}

func TestSearchPickerPickFollowsPlainRowPath(t *testing.T) {
	// The plain-row path, for comparison: one member of the same
	// group, listed flat.
	plain, _ := pickerTestModel(t)

	if err := plain.SearchList.SetItems(
		[]list.Item{
			SelectionItem{
				Label:         "BERSERK",
				Value:         "atsumoe/BERSERK",
				Provider:      "atsumoe",
				ProviderLabel: "Atsumoe (English)",
			},
		},
	); err != nil {
		t.Fatalf("set items: %v", err)
	}

	plain = setInteractiveSearchFocus(
		plain,
		searchFocusResults,
	)

	expected, _ := Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		plain,
	)

	// The pick: open the picker over the grouped row, move to
	// Atsumoe, Enter.
	m, _ := pickerTestModel(t)

	m, _ = Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	m, _ = handleSearchPickerKey(
		tea.KeyMsg{Type: tea.KeyDown},
		m,
	)

	next, _ := Update(
		tea.KeyMsg{Type: tea.KeyEnter},
		m,
	)

	if next.State != StateForm {
		t.Fatalf(
			"state = %v, want StateForm",
			next.State,
		)
	}

	if next.SelectedMangaProvider != "atsumoe" {
		t.Errorf(
			"selected manga provider = %q, want atsumoe",
			next.SelectedMangaProvider,
		)
	}

	if next.SelectedMangaURL != "atsumoe/BERSERK" {
		t.Errorf(
			"selected manga URL = %q, want atsumoe/BERSERK",
			next.SelectedMangaURL,
		)
	}

	if next.SelectedScanPath != "" {
		t.Errorf(
			"selected scan path = %q, want empty",
			next.SelectedScanPath,
		)
	}

	if next.SearchPickerVisible {
		t.Error("the picker should be closed after the pick")
	}

	// The picked provider drives the same log line and the same
	// screen the plain row path reaches, with its own origin.
	if expected.SelectedMangaProvider == "" {
		t.Fatal("sanity: the plain-row path should have selected")
	}

	lastLog := next.Logs[len(next.Logs)-1]

	if lastLog != "Manga selected: BERSERK. Fetching scan versions..." {
		t.Errorf(
			"log = %q, want the plain-row line for the picked row",
			lastLog,
		)
	}

	if next.Cursor != expected.Cursor ||
		next.SelectedScanPath != expected.SelectedScanPath {
		t.Error("pick must land where a plain row selection lands")
	}
}

func TestSearchPickerKeysMoveCursor(t *testing.T) {
	m, _ := pickerTestModel(t)

	m, _ = Update(tea.KeyMsg{Type: tea.KeyEnter}, m)

	// Cursor stays inside the bounds: two downs from 0 land on the
	// last row, further keys do not wrap.
	for i := 0; i < 5; i++ {
		m, _ = handleSearchPickerKey(
			tea.KeyMsg{Type: tea.KeyDown},
			m,
		)
	}

	if m.SearchPickerCursor != 1 {
		t.Fatalf(
			"cursor = %d, want 1 (last row, no wrap)",
			m.SearchPickerCursor,
		)
	}

	m, _ = handleSearchPickerKey(
		tea.KeyMsg{Type: tea.KeyUp},
		m,
	)

	if m.SearchPickerCursor != 0 {
		t.Fatalf("cursor = %d, want 0", m.SearchPickerCursor)
	}

	// Vim keys behave the same.
	m, _ = handleSearchPickerKey(
		tea.KeyMsg{Type: tea.KeyDown},
		m,
	)

	m, _ = handleSearchPickerKey(
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
		m,
	)

	if m.SearchPickerCursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.SearchPickerCursor)
	}

	m, _ = handleSearchPickerKey(
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}},
		m,
	)

	if m.SearchPickerCursor != 0 {
		t.Fatalf("cursor = %d, want 0", m.SearchPickerCursor)
	}
}

func TestSearchPickerViewRendersOverSearch(t *testing.T) {
	m, _ := pickerTestModel(t)
	m.Width = 80
	m.Height = 24

	m = openSearchPicker(
		m,
		m.SearchList.Items()[0].(SelectionItem),
	)

	view := View(m)

	if !strings.Contains(view, "Atsumoe (English)") {
		t.Error("the picker should list the group's providers")
	}

	if !strings.Contains(view, "Search manga") {
		t.Error(
			"the search view should stay rendered behind the picker",
		)
	}

	if !strings.Contains(view, "Enter pick • Esc back") {
		t.Error("the picker should carry its hint line")
	}
}

func TestSearchPickerKeyRunsThroughUpdate(t *testing.T) {
	m, _ := pickerTestModel(t)

	// While the picker is open, Update routes every key to it.
	m, _ = Update(tea.KeyMsg{Type: tea.KeyEnter}, m)

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyRunes,
			Runes: []rune{
				'x',
			},
		},
		m,
	)

	// Raw letters are ignored by the picker and the search input
	// stays blurred: typing must not leak into the list screen.
	if !next.SearchPickerVisible {
		t.Error("the picker should still be open")
	}

	if next.SearchFocus != searchFocusResults {
		t.Errorf(
			"focus = %v, want searchFocusResults (untouched)",
			next.SearchFocus,
		)
	}

	if currentInteractiveSearchQuery(next) != "berserk" {
		t.Errorf(
			"query = %q, want unchanged",
			currentInteractiveSearchQuery(next),
		)
	}

	next, _ = Update(
		tea.KeyMsg{Type: tea.KeyEsc},
		m,
	)

	if next.SearchPickerVisible {
		t.Fatal("esc while the picker is open should close it")
	}

	if next.State != StateInteractiveSearch {
		t.Errorf(
			"state = %v, want StateInteractiveSearch (search stays open)",
			next.State,
		)
	}
}
