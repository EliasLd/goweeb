package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func handleInteractiveSearchUpdate(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		m = resizeInteractiveSearch(m)

		return m, nil

	case interactiveSearchDebounceMsg:
		return handleInteractiveSearchDebounce(
			msg,
			m,
		)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return closeInteractiveSearch(m), nil
		}

		if m.SearchFocus == searchFocusInput {
			return handleInteractiveSearchInput(
				msg,
				m,
			)
		}

		return handleInteractiveSearchResults(
			msg,
			m,
		)
	}

	return m, nil
}

func handleInteractiveSearchInput(
	msg tea.KeyMsg,
	m Model,
) (Model, tea.Cmd) {
	switch msg.String() {
	case "down", "tab", "enter":
		if interactiveSearchResultsSelectable(m) {
			m = setInteractiveSearchFocus(
				m,
				searchFocusResults,
			)
		}

		return m, nil
	}

	before := currentInteractiveSearchQuery(m)

	var inputCmd tea.Cmd

	m.SearchInput, inputCmd =
		m.SearchInput.Update(msg)

	after := currentInteractiveSearchQuery(m)

	// No provider query change.
	if before == after {
		return m, inputCmd
	}

	// Invalidate all debounce messages/results produced
	// for the previous query.
	m.SearchGeneration++

	m.SearchPending = false
	m.SearchPendingQuery = ""
	m.SearchPendingGeneration = 0

	m.SearchError = ""
	m.SearchHasSearched = false

	if len(after) < minInteractiveSearchLength {
		return m, inputCmd
	}

	// If these exact results are already available,
	// there is no reason to query the provider again.
	if after == m.SearchResultsQuery {
		m.SearchHasSearched = true

		return m, inputCmd
	}

	searchCmd := debounceInteractiveSearch(
		after,
		m.SearchGeneration,
	)

	return m, tea.Batch(
		inputCmd,
		searchCmd,
	)
}

func handleInteractiveSearchResults(
	msg tea.KeyMsg,
	m Model,
) (Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.SearchList.Index() == 0 {
			m = setInteractiveSearchFocus(
				m,
				searchFocusInput,
			)

			return m, nil
		}

	case "shift+tab":
		m = setInteractiveSearchFocus(
			m,
			searchFocusInput,
		)

		return m, nil

	case "enter":
		if !interactiveSearchResultsSelectable(m) {
			return m, nil
		}

		item, ok := m.SearchList.
			SelectedItem().(SelectionItem)

		if !ok {
			return m, nil
		}

		m.SelectedMangaURL = item.Value
		m.SelectedScanPath = ""

		m.State = StateForm
		m.Cursor = 3
		m = updateFocus(m)

		m.Logs = append(
			m.Logs,
			fmt.Sprintf(
				"Manga selected: %s. Fetching scan versions...",
				item.Label,
			),
		)

		return m, fetchScanPaths(
			m.SelectedProvider,
			m.SelectedMangaURL,
			m.DomainInput.Value(),
		)
	}

	var listCmd tea.Cmd

	m.SearchList, listCmd =
		m.SearchList.Update(msg)

	return m, listCmd
}

func handleInteractiveSearchDebounce(
	msg interactiveSearchDebounceMsg,
	m Model,
) (Model, tea.Cmd) {
	if m.State != StateInteractiveSearch {
		return m, nil
	}

	if msg.generation != m.SearchGeneration {
		return m, nil
	}

	if msg.query != currentInteractiveSearchQuery(m) {
		return m, nil
	}

	if len(msg.query) < minInteractiveSearchLength {
		return m, nil
	}

	if m.SearchInFlight {
		m.SearchPending = true
		m.SearchPendingQuery = msg.query
		m.SearchPendingGeneration =
			msg.generation

		return m, nil
	}

	return startInteractiveSearchRequest(
		m,
		msg.query,
		msg.generation,
	)
}

func startInteractiveSearchRequest(
	m Model,
	query string,
	generation uint64,
) (Model, tea.Cmd) {
	m.SearchInFlight = true
	m.SearchRequestGeneration = generation

	m.SearchPending = false
	m.SearchPendingQuery = ""
	m.SearchPendingGeneration = 0

	m.SearchError = ""

	return m, searchInteractiveCatalog(
		m.SelectedProvider,
		query,
		m.DomainInput.Value(),
		generation,
	)
}

func handleInteractiveSearchResult(
	msg interactiveSearchResultMsg,
	m Model,
) (Model, tea.Cmd) {
	// Only the request currently tracked as active may
	// clear SearchInFlight.
	if m.SearchInFlight &&
		msg.generation ==
			m.SearchRequestGeneration {
		m.SearchInFlight = false
	}

	var listCmd tea.Cmd

	isCurrentResult :=
		m.State == StateInteractiveSearch &&
			msg.generation ==
				m.SearchGeneration &&
			msg.query ==
				currentInteractiveSearchQuery(m)

	if isCurrentResult {
		m.SearchHasSearched = true
		m.SearchResultsQuery = msg.query

		if msg.err != nil {
			m.SearchError = msg.err.Error()

			listCmd = m.SearchList.SetItems(nil)
		} else {
			m.SearchError = ""

			items := make(
				[]list.Item,
				len(msg.results),
			)

			for i, result := range msg.results {
				items[i] = SelectionItem{
					Label: result.Title,
					Value: result.URL,
				}
			}

			listCmd = m.SearchList.SetItems(items)
		}
	}

	// A newer debounced query may have been queued while
	// this request was running.
	if !m.SearchInFlight &&
		m.SearchPending {
		query := m.SearchPendingQuery
		generation :=
			m.SearchPendingGeneration

		m.SearchPending = false
		m.SearchPendingQuery = ""
		m.SearchPendingGeneration = 0

		if m.State ==
			StateInteractiveSearch &&
			generation ==
				m.SearchGeneration &&
			query ==
				currentInteractiveSearchQuery(m) {

			var searchCmd tea.Cmd

			m, searchCmd =
				startInteractiveSearchRequest(
					m,
					query,
					generation,
				)

			return m, tea.Batch(
				listCmd,
				searchCmd,
			)
		}
	}

	return m, listCmd
}
