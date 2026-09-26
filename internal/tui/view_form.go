package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func viewForm(m Model) string {
	var form strings.Builder

	form.WriteString(titleStyle.Render(m.Title))
	form.WriteString("\n\n")

	form.WriteString(labelStyle.Render("Manga title"))
	form.WriteString("\n\n")

	if m.Cursor == 0 {
		form.WriteString(m.MangaInput.View())
	} else {
		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.MangaInput.View()),
		)
	}

	form.WriteString("\n\n")

	form.WriteString(labelStyle.Render("Destination folder"))
	form.WriteString("\n\n")

	if m.Cursor == 1 {
		form.WriteString(m.ScanDirInput.View())
	} else {
		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.ScanDirInput.View()),
		)
	}

	form.WriteString("\n\n")

	form.WriteString(labelStyle.Render("Provider"))
	form.WriteString("\n\n")

	providerButton := "[ Select provider ]"

	if m.SelectedProviderLabel != "" {
		providerButton = fmt.Sprintf(
			"[ Provider: %s ]",
			m.SelectedProviderLabel,
		)
	}

	if m.Cursor == 2 {
		providerButton = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Render(providerButton)
	} else {
		providerButton = lipgloss.NewStyle().
			Faint(true).
			Render(providerButton)
	}

	form.WriteString(providerButton)
	form.WriteString("\n\n")

	form.WriteString(
		m.EbookCheckbox.View(m.Cursor == 3),
	)
	form.WriteString("\n\n")

	form.WriteString(
		m.KeepCheckbox.View(m.Cursor == 4),
	)
	form.WriteString("\n\n")

	if m.DownloadReady {
		button := "[ Search ]"

		if m.Cursor == 5 {
			button = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("226")).
				Render(button)
		} else {
			button = lipgloss.NewStyle().
				Faint(true).
				Render(button)
		}

		form.WriteString(button)
		form.WriteString("\n\n")
	}

	var logs strings.Builder

	const maxLogs = 18

	start := 0

	if len(m.Logs) > maxLogs {
		start = len(m.Logs) - maxLogs
	}

	visibleLogs := m.Logs[start:]

	for _, line := range visibleLogs {
		logs.WriteString(line + "\n")
	}

	if len(m.Logs) == 0 {
		logs.WriteString("No logs yet...")
	}

	for i := len(visibleLogs); i < maxLogs; i++ {
		logs.WriteString("\n")
	}

	logsView := logBoxStyle.Render(
		logs.String(),
	)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		form.String(),
		logsView,
	)

	footer := lipgloss.NewStyle().
		Faint(true).
		Width(m.Width).
		Align(lipgloss.Center).
		Render(
			"↑/↓ navigate • Space/Enter toggle • Enter to select/search • Ctrl+C/Esc quit",
		)

	// Reserve the last line of the terminal for the footer.
	contentHeight := max(0, m.Height-1)

	centeredContent := lipgloss.Place(
		m.Width,
		contentHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredContent,
		footer,
	)
}
