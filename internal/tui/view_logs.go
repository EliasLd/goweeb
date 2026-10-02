package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	logOverlayBorderColor = lipgloss.AdaptiveColor{
		Light: "#333333",
		Dark:  "#FFFFFF",
	}

	logOverlayStyle = lipgloss.NewStyle().
			Padding(0, 2)

	logViewportTitleStyle = func() lipgloss.Style {
		border := lipgloss.RoundedBorder()
		border.Right = "├"

		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			BorderStyle(border).
			BorderForeground(
				logOverlayBorderColor,
			).
			Bold(true).
			Padding(0, 1)
	}()

	logOverlayHintStyle = lipgloss.NewStyle().
				Faint(true)
)

func logViewportHeader(
	m Model,
) string {
	title := logViewportTitleStyle.Render(
		"Live logs",
	)

	line := strings.Repeat(
		"─",
		max(
			0,
			m.LogViewport.Width-
				lipgloss.Width(title),
		),
	)

	line = lipgloss.NewStyle().
		Foreground(
			logOverlayBorderColor,
		).
		Render(line)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		line,
	)
}

func logViewportFooter(
	m Model,
) string {
	line := strings.Repeat(
		"─",
		max(0, m.LogViewport.Width),
	)

	return lipgloss.NewStyle().
		Foreground(
			logOverlayBorderColor,
		).
		Render(line)
}

func viewLogOverlay(
	m Model,
) string {
	body := lipgloss.NewStyle().
		Width(m.LogViewport.Width).
		Height(m.LogViewport.Height).
		Render(
			m.LogViewport.View(),
		)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		logViewportHeader(m),
		body,
		"",
		logOverlayHintStyle.Render(
			"↑/↓ scroll • PgUp/PgDn • Ctrl+L/Esc close",
		),
		logViewportFooter(m),
	)

	return logOverlayStyle.Render(
		content,
	)
}
