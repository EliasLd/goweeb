package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func hasBackAction(
	state AppState,
) bool {
	switch state {
	case StateDestinationPicker,
		StateProviderSelection,
		StateOptionalSettings,
		StateInteractiveSearch,
		StateScanSelection,
		StateRangeSelection:
		return true
	}

	return false
}

func handleBackAction(
	m Model,
) (Model, tea.Cmd) {
	return updateApp(
		tea.KeyMsg{
			Type: tea.KeyEsc,
		},
		m,
	)
}
