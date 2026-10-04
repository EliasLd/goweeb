package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func viewForm(
	m Model,
) string {
	width :=
		mainViewWidth(
			m.Width,
		)

	brand :=
		renderMainBrand(
			width,
		)

	deck :=
		renderMainDeck(
			m,
			width,
		)

	status :=
		renderMainStatus(
			m,
			width,
		)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		brand,
		"",
		"",
		deck,
		"",
		status,
	)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		content,
		"↑/↓ navigate • Space/Enter select • Ctrl+L logs • Ctrl+C/Esc quit",
		lipgloss.Center,
		lipgloss.Center,
	)
}
