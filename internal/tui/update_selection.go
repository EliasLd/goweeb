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

		m = updateSearchReady(m)

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

		m.State = StateForm
		m.Cursor = 1
		m = updateFocus(m)

		m.Logs = append(
			m.Logs,
			fmt.Sprintf(
				"Provider selected: %s",
				newProviderLabel,
			),
		)

		return m, nil
	}

	if m.ProviderSelectionModel.Cancelled {
		m.ProviderSelectionModel.Cancelled = false
		m.State = StateForm
		m.Cursor = 1
		m = updateFocus(m)

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
			m.SelectedProvider,
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
