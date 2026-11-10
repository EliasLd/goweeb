package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

var (
	searchPickerBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Padding(0, 1)

	searchPickerTitleStyle = lipgloss.NewStyle().
				Foreground(brightAccentColor).
				Bold(true)

	searchPickerSelectedStyle = lipgloss.NewStyle().
					Foreground(brightAccentColor).
					Bold(true)

	searchPickerRowStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	searchPickerHintStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true)
)

// searchPickerInnerWidth sizes the picker content: compact, minus
// the terminal margins, box borders and horizontal padding; never
// wider than a fixed comfortable span.
func searchPickerInnerWidth(m Model) int {
	return max(1, min(44, m.Width-12))
}

// searchPickerRow renders one provider of the picker with its mouse
// zone; the cursor row carries the accent.
func searchPickerRow(
	m Model,
	index int,
	item SelectionItem,
	inner int,
) string {
	row := truncate.StringWithTail(
		fmt.Sprintf(
			"%d. %s",
			index+1,
			item.ProviderLabel,
		),
		uint(inner),
		"-",
	)

	var rendered string

	if index == m.SearchPickerCursor {
		rendered = searchPickerSelectedStyle.Render(
			"> " + row,
		)
	} else {
		rendered = searchPickerRowStyle.Render(
			"  " + row,
		)
	}

	return markMouseZone(
		searchPickerZone(index),
		rendered,
	)
}

func viewSearchPicker(
	m Model,
) string {
	inner := searchPickerInnerWidth(m)

	var b strings.Builder

	title := truncate.StringWithTail(
		m.SearchPickerLabel,
		uint(inner),
		"-",
	)

	b.WriteString(
		searchPickerTitleStyle.Render(title),
	)

	b.WriteString("\n\n")

	for index, item := range m.SearchPickerItems {
		b.WriteString(
			searchPickerRow(m, index, item, inner),
		)

		b.WriteString("\n")
	}

	b.WriteString("\n")

	b.WriteString(
		searchPickerHintStyle.Render(
			"↑/↓ navigate • Enter pick • Esc back",
		),
	)

	return searchPickerBoxStyle.
		Width(inner).
		Render(b.String())
}
