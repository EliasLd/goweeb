package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"
)

func Update(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	alertModel, alertCmd :=
		m.AlertModel.Update(msg)

	m.AlertModel =
		alertModel.(bubbleup.AlertModel)

	next, appCmd := updateApp(
		msg,
		m,
	)

	return next, tea.Batch(
		alertCmd,
		appCmd,
	)
}

// Update routes Bubble Tea messages to the appropriate handler.
func updateApp(
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

	switch msg := msg.(type) {
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

	case spinner.TickMsg:
		if !m.IsDownloading {
			return m, nil
		}

		var cmd tea.Cmd
		m.DownloadSpinner, cmd = m.DownloadSpinner.Update(msg)
		return m, cmd
	}

	if mouseMsg, ok :=
		msg.(tea.MouseMsg); ok {

		if next, cmd, handled :=
			handleMouseUpdate(
				mouseMsg,
				m,
			); handled {
			return next, cmd
		}
	}

	if isMainViewState(m.State) {
		if m.LogsVisible {
			switch msg.(type) {
			case tea.KeyMsg, tea.WindowSizeMsg:
				return handleLogOverlayUpdate(
					msg,
					m,
				)
			}
		}

		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if shouldToggleLogs(keyMsg) {
				return openLogOverlay(m), nil
			}
		}
	}

	if m.State == StateDestinationPicker {
		return handleDestinationPickerUpdate(
			msg,
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

		m = resizeLogViewport(m)

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
	}

	return updateSearchReady(m), nil
}
