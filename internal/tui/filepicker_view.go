package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

func destinationPickerContentWidth(
	terminalWidth int,
) int {
	if terminalWidth <= 0 {
		return 60
	}

	return min(
		80,
		max(
			30,
			terminalWidth-8,
		),
	)
}

var destinationPickerCurrentStyle = lipgloss.NewStyle().
	Foreground(mutedColor).
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

	pickerWidth := destinationPickerContentWidth(
		m.Width,
	)

	current := fmt.Sprintf(
		"Current: %s",
		m.DestinationPicker.CurrentDirectory,
	)

	current = wrap.String(
		current,
		pickerWidth,
	)

	content.WriteString(
		destinationPickerCurrentStyle.Render(
			current,
		),
	)

	content.WriteString("\n\n")

	content.WriteString(
		m.DestinationPicker.View(),
	)

	renderedContent := lipgloss.NewStyle().
		Width(pickerWidth).
		Render(
			content.String(),
		)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		renderedContent,
		"↑/k ↓/j navigate • →/l open • ←/h back • Enter select • Esc back",
		lipgloss.Center,
		lipgloss.Center,
	)
}
