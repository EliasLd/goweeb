package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var destinationPickerCurrentStyle = lipgloss.NewStyle().
	Faint(true)

func viewDestinationPicker(
	m Model,
) string {
	var content strings.Builder

	content.WriteString(
		titleStyle.Render(
			"Select directory to download manga to",
		),
	)

	content.WriteString("\n")

	content.WriteString(
		destinationPickerCurrentStyle.Render(
			fmt.Sprintf(
				"Current: %s",
				m.DestinationPicker.CurrentDirectory,
			),
		),
	)

	content.WriteString("\n\n")

	content.WriteString(
		m.DestinationPicker.View(),
	)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		content.String(),
		"↑/k ↓/j navigate • →/l open • ←/h back • Enter select • Esc back",
		lipgloss.Center,
		lipgloss.Center,
	)
}
