package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func isMainViewState(state AppState) bool {
	return state == StateForm ||
		state == StateDownloading
}

func logOverlaySize(
	width int,
	height int,
) (int, int) {
	overlayWidth := max(
		36,
		width*3/5,
	)

	overlayHeight := max(
		10,
		height*3/5,
	)

	overlayWidth = min(
		overlayWidth,
		max(1, width-4),
	)

	overlayHeight = min(
		overlayHeight,
		max(1, height-2),
	)

	return overlayWidth, overlayHeight
}

func resizeLogViewport(m Model) Model {
	width, height := logOverlaySize(
		m.Width,
		m.Height,
	)

	m.LogViewport.Width = max(
		1,
		width-4,
	)

	// Header = 3 lines
	// Hint = 1
	// Bottom line = 1
	m.LogViewport.Height = max(
		1,
		height-5,
	)

	return m
}

func logContent(logs []string) string {
	if len(logs) == 0 {
		return "No logs yet..."
	}

	return strings.Join(
		logs,
		"\n",
	)
}

func syncLogViewport(
	m Model,
	forceBottom bool,
) Model {
	wasAtBottom := m.LogViewport.AtBottom()

	m.LogViewport.SetContent(
		logContent(m.Logs),
	)

	if forceBottom || wasAtBottom {
		m.LogViewport.GotoBottom()
	}

	return m
}

func openLogOverlay(m Model) Model {
	m.LogsVisible = true
	m = resizeLogViewport(m)
	m = syncLogViewport(m, true)

	return m
}

func closeLogOverlay(m Model) Model {
	m.LogsVisible = false

	if m.State == StateForm {
		m = updateFocus(m)
	}

	return m
}

func handleLogOverlayUpdate(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		m = resizeLogViewport(m)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "l", "ctrl+l", "esc":
			return closeLogOverlay(m), nil
		}
	}

	var cmd tea.Cmd

	m.LogViewport, cmd =
		m.LogViewport.Update(msg)

	return m, cmd
}

func shouldToggleLogs(
	msg tea.KeyMsg,
) bool {
	switch msg.String() {
	case "ctrl+l":
		return true
	}

	return false
}

func appendLogLine(
	m Model,
	line string,
) Model {
	followBottom :=
		!m.LogsVisible ||
			m.LogViewport.AtBottom()

	m.Logs = append(
		m.Logs,
		line,
	)

	if !m.LogsVisible {
		return m
	}

	m.LogViewport.SetContent(
		logContent(m.Logs),
	)

	if followBottom {
		m.LogViewport.GotoBottom()
	}

	return m
}
