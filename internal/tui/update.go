package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update routes Bubble Tea messages to the appropriate handler.
func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	// Selection screens manage their own keyboard events.
	if m.State == StateProviderSelection {
		return handleProviderSelectionUpdate(msg, m)
	}

	if m.State == StateMangaSelection ||
		m.State == StateScanSelection {
		return handleSelectionUpdate(msg, m)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case catalogSearchResultMsg:
		return handleCatalogSearchResult(msg, m)

	case scanPathResultMsg:
		return handleScanPathResult(msg, m)

	case entriesResultMsg:
		return handleEntriesResult(msg, m)

	case tea.KeyMsg:
		// Keep the current behavior while downloading:
		// only the quit shortcuts are accepted.
		if m.IsDownloading &&
			msg.String() != "ctrl+c" &&
			msg.String() != "esc" {
			return m, nil
		}

		if m.State == StateRangeSelection {
			return handleRangeUpdate(msg, m)
		}

		return handleFormUpdate(msg, m)

	case setupLogPipeMsg:
		return handleSetupLogPipe(msg, m)

	case logMsg:
		return handleLogMsg(msg, m)
	}

	return updateDownloadReady(m), nil
}
