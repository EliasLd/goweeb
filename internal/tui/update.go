package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update routes Bubble Tea messages to the appropriate handler.
func Update(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	// Interactive search results must still be consumed
	// after leaving the search screen so an old in-flight
	// request cannot leave SearchInFlight stuck forever.
	if result, ok := msg.(interactiveSearchResultMsg); ok {
		return handleInteractiveSearchResult(
			result,
			m,
		)
	}

	if m.State == StateProviderSelection {
		return handleProviderSelectionUpdate(
			msg,
			m,
		)
	}

	if m.State == StateInteractiveSearch {
		return handleInteractiveSearchUpdate(
			msg,
			m,
		)
	}

	if m.State == StateScanSelection {
		return handleSelectionUpdate(
			msg,
			m,
		)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		return m, nil

	case scanPathResultMsg:
		return handleScanPathResult(
			msg,
			m,
		)

	case entriesResultMsg:
		return handleEntriesResult(
			msg,
			m,
		)

	case tea.KeyMsg:
		if m.IsDownloading &&
			msg.String() != "ctrl+c" &&
			msg.String() != "esc" {
			return m, nil
		}

		if m.State == StateRangeSelection {
			return handleRangeUpdate(
				msg,
				m,
			)
		}

		if m.State == StateOptionalSettings {
			return handleOptionalSettingsUpdate(
				msg,
				m,
			)
		}

		return handleFormUpdate(
			msg,
			m,
		)

	case setupLogPipeMsg:
		return handleSetupLogPipe(
			msg,
			m,
		)

	case logMsg:
		return handleLogMsg(
			msg,
			m,
		)
	}

	return updateSearchReady(m), nil
}
