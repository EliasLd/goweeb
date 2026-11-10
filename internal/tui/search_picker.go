package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// openSearchPicker lists the providers behind a grouped search
// result. The result list itself is left untouched: closing the
// picker restores it exactly.
func openSearchPicker(
	m Model,
	group SelectionItem,
) Model {
	m.SearchPickerVisible = true
	m.SearchPickerLabel = group.Label
	m.SearchPickerItems = group.Members
	m.SearchPickerCursor = 0
	m.HoveredAction = ""

	return m
}

// closeSearchPicker dismisses the overlay without touching the
// result list: the search screen resumes exactly where it was.
func closeSearchPicker(m Model) Model {
	m.SearchPickerVisible = false
	m.SearchPickerLabel = ""
	m.SearchPickerItems = nil
	m.SearchPickerCursor = 0
	m.HoveredAction = ""

	return m
}

// handleSearchPickerKey drives the picker while it is open: arrows
// move the cursor, Enter adopts a provider, Esc goes back to the
// result list.
func handleSearchPickerKey(
	msg tea.KeyMsg,
	m Model,
) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		return closeSearchPicker(m), nil

	case "up", "k":
		if m.SearchPickerCursor > 0 {
			m.SearchPickerCursor--
		}

		return m, nil

	case "down", "j":
		if next := m.SearchPickerCursor + 1; next <
			len(m.SearchPickerItems) {
			m.SearchPickerCursor = next
		}

		return m, nil

	case "enter":
		if m.SearchPickerCursor < 0 ||
			m.SearchPickerCursor >= len(m.SearchPickerItems) {
			return closeSearchPicker(m), nil
		}

		picked := m.SearchPickerItems[m.SearchPickerCursor]

		m = closeSearchPicker(m)

		return selectSearchResult(m, picked)
	}

	return m, nil
}

// selectSearchResult adopts a search result as the download target
// and moves on to the scan-path prompt. Both a plain result row and
// a provider picked from a group route through here, keeping the two
// paths identical.
func selectSearchResult(
	m Model,
	item SelectionItem,
) (Model, tea.Cmd) {
	m.SelectedMangaURL = item.Value
	m.SelectedScanPath = ""

	// Adopting the provider of the selected result keeps the
	// scan versions / chapters / download flow pointed at the
	// provider the result came from. The persistent search
	// filter (SelectedProvider) only changes on the provider
	// screen.
	m.SelectedMangaProvider = item.Provider

	m.State = StateForm
	m.Cursor = 2
	m = updateFocus(m)

	m.Logs = append(
		m.Logs,
		fmt.Sprintf(
			"Manga selected: %s. Fetching scan versions...",
			item.Label,
		),
	)

	return m, fetchScanPaths(
		m.SelectedMangaProvider,
		m.SelectedMangaURL,
		m.DomainInput.Value(),
	)
}
