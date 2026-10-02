package tui

import (
	"bufio"
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// Initializes the scanner used to receive download logs.
func handleSetupLogPipe(
	msg setupLogPipeMsg,
	m Model,
) (Model, tea.Cmd) {
	m.pipeReader = msg.reader
	m.scanner = bufio.NewScanner(m.pipeReader)

	buf := make([]byte, 64*1024)
	m.scanner.Buffer(buf, 1024*1024)

	return m, readOneLogLine(m)
}

// Handles a log line received during downloading.
func handleLogMsg(
	msg logMsg,
	m Model,
) (Model, tea.Cmd) {
	logLine := string(msg)

	switch {
	case logLine == "Finished downloading":
		styled := highlightStyle.Render(
			"Download complete!",
		)

		m = appendLogLine(
			m,
			styled,
		)

		m.IsDownloading = false
		m.State = StateForm
		m.Cursor = 0

		if app.OutputDirLocked() {
			m.Cursor = 1
		}

		if !m.LogsVisible {
			m = updateFocus(m)
		}

	case strings.HasPrefix(
		logLine,
		"[DEBUG]",
	):
		// Debug lines remain hidden from the TUI.

	case strings.Contains(
		logLine,
		"[ERROR]",
	):
		m = appendLogLine(
			m,
			errorStyle.Render(logLine),
		)

	case strings.Contains(
		logLine,
		"[!]",
	):
		m = appendLogLine(
			m,
			logLine,
		)

	default:
		m = appendLogLine(
			m,
			logLine,
		)
	}

	if m.IsDownloading {
		return m, readOneLogLine(m)
	}

	return m, nil
}
