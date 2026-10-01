package tui

import (
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
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

	if app.OutputDirLocked() {
		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.ScanDirInput.Value()),
		)

		form.WriteString("\n")

		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Italic(true).
				Render(
					"Managed by the container volume.",
				),
		)
	} else if m.Cursor == 1 {
		form.WriteString(m.ScanDirInput.View())
	} else {
		form.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.ScanDirInput.View()),
		)
	}

	form.WriteString("\n\n")

	providerLabel := "[ Select provider ]"

	if m.SelectedProviderLabel != "" {
		providerLabel = "Provider: " + m.SelectedProviderLabel
	}

	form.WriteString(
		renderButton(
			providerLabel,
			m.Cursor == 2,
			buttonSecondary,
		),
	)

	form.WriteString("\n\n")

	// Optional settings
	form.WriteString(
		renderButton(
			"[ Optional settings ]",
			m.Cursor == 3,
			buttonSecondary,
		),
	)
	form.WriteString("\n\n")

	if m.DownloadReady {
		form.WriteString(
			renderButton(
				"[ Start Search ]",
				m.Cursor == 4,
				buttonPrimary,
			),
		)
	} else {
		form.WriteString(
			disabledButtonStyle.Render("Start Search"),
		)
	}

	form.WriteString("\n\n")

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

	return renderViewWithFooter(
		m.Width,
		m.Height,
		content,
		"↑/↓ navigate • Space/Enter select • Ctrl+C/Esc quit",
		lipgloss.Center,
		lipgloss.Center,
	)
}
