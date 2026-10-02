package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	logOverlayStyle = lipgloss.NewStyle().
			Border(
			lipgloss.RoundedBorder(),
		).
		BorderForeground(
			lipgloss.AdaptiveColor{
				Light: "#333333",
				Dark:  "#FFFFFF",
			},
		).
		Padding(1, 2)

	logOverlayHintStyle = lipgloss.NewStyle().
				Faint(true)
)

func viewLogOverlay(m Model) string {
	title := "Logs"

	if m.IsDownloading {
		title = "Logs · live"
	}

	body := lipgloss.NewStyle().
		Width(m.LogViewport.Width).
		Height(m.LogViewport.Height).
		Render(
			m.LogViewport.View(),
		)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		"",
		body,
		"",
		logOverlayHintStyle.Render(
			"↑/↓ scroll • PgUp/PgDn • l/Esc close",
		),
	)

	return logOverlayStyle.Render(
		content,
	)
}
