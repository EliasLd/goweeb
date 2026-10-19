package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type searchFocus int

const (
	minInteractiveSearchLength                = 2
	interactiveSearchDefaultWidth             = 60
	interactiveSearchMaxWidth                 = 80
	interactiveSearchMargin                   = 8
	searchFocusInput              searchFocus = iota
	searchFocusResults
	searchFocusProvider
)

func interactiveSeachWidth(
	terminalWidth int,
) int {
	if terminalWidth <= 0 {
		return interactiveSearchDefaultWidth
	}

	return min(
		interactiveSearchMaxWidth,
		max(
			1,
			terminalWidth-interactiveSearchMargin,
		),
	)
}

func newInteractiveSearchList() list.Model {
	l := list.New(
		[]list.Item{},
		itemDelegate{Focused: false, ZonePrefix: "search-result-"},
		60,
		14,
	)

	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.SetShowHelp(false)

	// The provider search input replaces Bubbles' local filtering.
	l.SetFilteringEnabled(false)

	return l
}

// applySearchPlaceholder matches the search input placeholder with
// the current provider filter.
func applySearchPlaceholder(m Model) Model {
	providerLabel := strings.TrimSpace(
		m.SelectedProviderLabel,
	)

	if providerLabel == "" {
		m.SearchInput.Placeholder =
			"Search across all providers..."
	} else {
		m.SearchInput.Placeholder = fmt.Sprintf(
			"Search on %s...",
			providerLabel,
		)
	}

	return m
}

func openInteractiveSearch(m Model) (Model, tea.Cmd) {
	// Invalidate debounce/results belonging to a previous search session.
	m.SearchGeneration++

	m.SearchPending = false
	m.SearchPendingQuery = ""
	m.SearchPendingGeneration = 0

	m.SearchResultsQuery = ""
	m.SearchHasSearched = false
	m.SearchErrors = nil
	m.SearchInFlight = false
	m.SearchProvidersPending = 0

	m.SearchInput.SetValue("")

	m = applySearchPlaceholder(m)

	clearCmd := m.SearchList.SetItems(nil)

	m.State = StateInteractiveSearch
	m = setInteractiveSearchFocus(
		m,
		searchFocusInput,
	)

	m = resizeInteractiveSearch(m)

	return m, clearCmd
}

func closeInteractiveSearch(m Model) Model {
	// Invalidate pending debounce messages and results.
	m.SearchGeneration++

	m.SearchPending = false
	m.SearchPendingQuery = ""
	m.SearchPendingGeneration = 0

	m.SearchInput.Blur()

	m.State = StateForm
	m.Cursor = 2

	return updateFocus(m)
}

func setInteractiveSearchFocus(
	m Model,
	focus searchFocus,
) Model {
	m.SearchFocus = focus

	switch focus {
	case searchFocusResults:
		m.SearchInput.Blur()

		m.SearchList.SetDelegate(
			itemDelegate{
				Focused:    true,
				ZonePrefix: "search-result-",
			},
		)

	case searchFocusProvider:
		// Neither the input nor the list is focused; the
		// provider row carries the accent instead.
		m.SearchInput.Blur()

		m.SearchList.SetDelegate(
			itemDelegate{
				Focused:    false,
				ZonePrefix: "search-result-",
			},
		)

	default:
		m.SearchInput.Focus()

		m.SearchList.SetDelegate(
			itemDelegate{
				Focused:    false,
				ZonePrefix: "search-result-",
			},
		)
	}

	return m
}

func resizeInteractiveSearch(m Model) Model {
	searchWidth := interactiveSeachWidth(m.Width)
	listHeight := 14

	if m.Height > 0 {
		listHeight = min(
			18,
			max(1, m.Height-10),
		)
	}

	m.SearchInput.Width = max(
		1,
		searchWidth-2,
	)

	m.SearchList.SetWidth(searchWidth)
	m.SearchList.SetHeight(listHeight)

	return m
}

func currentInteractiveSearchQuery(
	m Model,
) string {
	return strings.TrimSpace(
		m.SearchInput.Value(),
	)
}

func interactiveSearchResultsCurrent(
	m Model,
) bool {
	query := currentInteractiveSearchQuery(m)

	return query != "" &&
		query == m.SearchResultsQuery
}

func interactiveSearchResultsSelectable(
	m Model,
) bool {
	return interactiveSearchResultsCurrent(m) &&
		len(m.SearchList.Items()) > 0
}
