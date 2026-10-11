package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	logOverlayStyle = lipgloss.NewStyle().
			Padding(0, 2)

	logViewportTitleStyle = func() lipgloss.Style {
		border := lipgloss.RoundedBorder()
		border.Right = "├"

		return lipgloss.NewStyle().
			Foreground(brightAccentColor).
			BorderStyle(border).
			BorderForeground(
				borderColor,
			).
			Bold(true).
			Padding(0, 1)
	}()

	logOverlayHintStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true)
)

func logViewportHeader(
	m Model,
) string {
	title :=
		logViewportTitleStyle.Render(
			"Live logs",
		)

	closeButton :=
		renderActionButton(
			"Close",
			m.HoveredAction ==
				mouseZoneLogClose,
		)

	closeWidth :=
		lipgloss.Width(
			closeButton,
		)

	closeButton =
		markMouseZone(
			mouseZoneLogClose,
			closeButton,
		)

	line := strings.Repeat(
		"─",
		max(
			0,
			m.LogViewport.Width-
				lipgloss.Width(title)-
				closeWidth,
		),
	)

	line =
		lipgloss.NewStyle().
			Foreground(
				borderColor,
			).
			Render(
				line,
			)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		line,
		closeButton,
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
			borderColor,
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
