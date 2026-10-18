package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
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

	m.SearchErrors = nil
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

		// Adopting the provider of the selected result keeps the
		// scan versions / chapters / download flow pointed at the
		// provider the result came from. The persistent search
		// filter (SelectedProvider) only changes on the provider
		// screen.
		m.SelectedMangaProvider = item.Provider

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
			m.SelectedMangaProvider,
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
	m.SearchProvidersPending =
		len(searchTargets(m.SelectedProvider))

	m.SearchPending = false
	m.SearchPendingQuery = ""
	m.SearchPendingGeneration = 0

	m.SearchErrors = nil
	m.SearchResultsQuery = ""
	m.SearchHasSearched = false

	// The request shows fresh results only; stale ones are
	// discarded upfront.
	clearCmd := m.SearchList.SetItems(nil)

	searchCmd := searchInteractiveCatalog(
		m.SelectedProvider,
		query,
		m.DomainInput.Value(),
		generation,
	)

	if searchCmd == nil {
		// Nothing matched the filter; nothing is in flight.
		m.SearchInFlight = false
		m.SearchProvidersPending = 0
	}

	return m, tea.Batch(
		searchCmd,
		clearCmd,
	)
}

// newSearchItems maps provider results to selectable items carrying
// their provider origin.
func newSearchItems(
	results []sourcetypes.SearchResult,
	provider string,
	providerLabel string,
) []list.Item {
	items := make(
		[]list.Item,
		len(results),
	)

	for i, result := range results {
		items[i] = SelectionItem{
			Label:         result.Title,
			Value:         result.URL,
			Provider:      provider,
			ProviderLabel: providerLabel,
		}
	}

	return items
}

// Maps provider results to selectable items carrying their origin.
func handleInteractiveSearchResult(
	msg interactiveSearchResultMsg,
	m Model,
) (Model, tea.Cmd) {
	// Only the request currently tracked as active may
	// clear SearchInFlight.
	isRequestResult :=
		m.SearchInFlight &&
			msg.generation ==
				m.SearchRequestGeneration

	if isRequestResult &&
		m.SearchProvidersPending > 0 {
		m.SearchProvidersPending--

		if m.SearchProvidersPending == 0 {
			m.SearchInFlight = false
			m.SearchHasSearched = true
		}
	}

	var listCmd tea.Cmd

	isCurrentResult :=
		m.State == StateInteractiveSearch &&
			msg.generation ==
				m.SearchGeneration &&
			msg.query ==
				currentInteractiveSearchQuery(m)

	if isCurrentResult {
		if msg.err != nil {
			// A failing provider must not wipe the results
			// other providers produced.
			m.SearchErrors = append(
				m.SearchErrors,
				msg.providerLabel,
			)
		} else {
			m.SearchResultsQuery = msg.query

			items := newSearchItems(
				msg.results,
				msg.provider,
				msg.providerLabel,
			)

			// Merge the new results with the ones other
			// providers already delivered, keeping the most
			// accurate matches on top.
			var merged []list.Item

			merged = append(
				merged,
				m.SearchList.Items()...,
			)

			merged = append(
				merged,
				items...,
			)

			sortSearchItems(merged, msg.query)

			listCmd = m.SearchList.SetItems(merged)
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
