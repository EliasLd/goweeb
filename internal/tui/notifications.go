package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"
)

const errorAlertMessage = "An error occurred. Check Logs for details."

func errorAlertCmd(
	m Model,
) tea.Cmd {
	if m.LogsVisible {
		return nil
	}

	return m.AlertModel.NewAlertCmd(
		bubbleup.ErrorKey,
		errorAlertMessage,
	)
}
