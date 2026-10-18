package tui

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/source"
	tea "github.com/charmbracelet/bubbletea"
)

// Handles provider selection, confirmation and cancellation.
func handleProviderSelectionUpdate(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	var cmd tea.Cmd

	providerModel, cmd := m.ProviderSelectionModel.Update(msg)
	m.ProviderSelectionModel = providerModel.(ProviderSelectionModel)

	if m.ProviderSelectionModel.Confirmed {
		newProvider := m.ProviderSelectionModel.SelectedID
		newProviderLabel := m.ProviderSelectionModel.SelectedLabel

		newDomain := strings.TrimSpace(
			m.ProviderSelectionModel.DomainInput.Value(),
		)

		oldProvider := m.SelectedProvider
		oldDomain := strings.TrimSpace(m.DomainInput.Value())

		m.SelectedProvider = newProvider
		m.SelectedProviderLabel = newProviderLabel
		m.DomainInput.SetValue(newDomain)

		// Changing provider configuration invalidates
		// selections made with the previous configuration.
		if oldProvider != newProvider ||
			oldDomain != newDomain {
			m.SelectedMangaURL = ""
			m.SelectedScanPath = ""

			m.DiscoveredEntries = 0
			m.DiscoveredEntryList = nil
			m.AvailableRanges = ""
			m.SelectedRange = app.RangeSelection{}
		}

		// The provider screen is only reachable from the search
		// view, so selection returns to it and refreshes the
		// search under the new filter.
		m.SearchGeneration++

		m.SearchInFlight = false
		m.SearchProvidersPending = 0

		m.SearchPending = false
		m.SearchPendingQuery = ""
		m.SearchPendingGeneration = 0

		m.SearchResultsQuery = ""
		m.SearchHasSearched = false
		m.SearchErrors = nil

		var searchCmd tea.Cmd

		clearCmd := m.SearchList.SetItems(nil)

		m.State = StateInteractiveSearch
		m = setInteractiveSearchFocus(
			m,
			searchFocusInput,
		)
		m = applySearchPlaceholder(m)
		m = resizeInteractiveSearch(m)

		// Re-run the query under the new filter when it is
		// already typed.
		query := currentInteractiveSearchQuery(m)

		if len(query) >= minInteractiveSearchLength {
			m, searchCmd = startInteractiveSearchRequest(
				m,
				query,
				m.SearchGeneration,
			)
		}

		m.Logs = append(
			m.Logs,
			fmt.Sprintf(
				"Provider selected: %s",
				newProviderLabel,
			),
		)

		return m, tea.Batch(
			clearCmd,
			searchCmd,
		)
	}

	if m.ProviderSelectionModel.Cancelled {
		m.ProviderSelectionModel.Cancelled = false

		m.State = StateInteractiveSearch
		m = setInteractiveSearchFocus(
			m,
			searchFocusInput,
		)
		m = resizeInteractiveSearch(m)

		return m, nil
	}

	return m, cmd
}

// Opens the provider selection screen.
func openProviderSelection(m Model) Model {
	m.ProviderSelectionModel = NewProviderSelectionModel(
		source.AvailableProviders(),
		m.SelectedProvider,
		strings.TrimSpace(m.DomainInput.Value()),
	)

	m.ProviderSelectionModel.Width = m.Width
	m.ProviderSelectionModel.Height = m.Height

	m.State = StateProviderSelection

	return m
}

func handleSelectionUpdate(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	var cmd tea.Cmd

	selectionModel, cmd :=
		m.SelectionModel.Update(msg)

	m.SelectionModel =
		selectionModel.(SelectionModel)

	if m.SelectionModel.Selected != "" {
		m.SelectedScanPath =
			m.SelectionModel.Selected

		m.State = StateForm

		m.Logs = append(
			m.Logs,
			"Scan version selected. Fetching chapters...",
		)

		return m, fetchEntries(
			m.SelectedMangaProvider,
			m.SelectedMangaURL,
			m.SelectedScanPath,
			strings.TrimSpace(
				m.DomainInput.Value(),
			),
		)
	}

	if m.SelectionModel.Cancelled {
		m.SelectionModel.Cancelled = false
		m.SelectionModel.Selected = ""

		m.Logs = append(
			m.Logs,
			"Selection cancelled",
		)

		m.State = StateForm
		m = updateFocus(m)

		return m, nil
	}

	return m, cmd
}
