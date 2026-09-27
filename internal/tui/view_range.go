package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func viewRangeSelection(m Model) string {
	var b strings.Builder

	title := fmt.Sprintf(
		"Found %d chapter(s)",
		m.DiscoveredEntries,
	)

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	if strings.TrimSpace(m.AvailableRanges) != "" {
		b.WriteString(
			labelStyle.Render("Available chapters"),
		)
		b.WriteString("\n")

		rangeWidth := 76

		if m.Width > 0 {
			rangeWidth = min(
				rangeWidth,
				max(1, m.Width-4),
			)
		}

		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					wrapAvailableRanges(
						m.AvailableRanges,
						rangeWidth,
					),
				),
		)

		b.WriteString("\n\n")
	}

	b.WriteString(labelStyle.Render("Chapter range"))
	b.WriteString("\n\n")

	if m.AllCheckbox.Checked {
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(m.RangeInput.View()),
		)

		b.WriteString("\n")

		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Italic(true).
				Render(
					"Disabled because 'Download all chapters' is enabled.",
				),
		)
	} else {
		if m.Cursor == 0 {
			b.WriteString(m.RangeInput.View())
		} else {
			b.WriteString(
				lipgloss.NewStyle().
					Faint(true).
					Render(m.RangeInput.View()),
			)
		}
	}

	b.WriteString("\n\n")

	b.WriteString(
		lipgloss.NewStyle().
			Faint(true).
			Render(
				"Accepted formats: 10, 1-10, 10-, -10, 1-10,20-30",
			),
	)

	b.WriteString("\n")

	b.WriteString(
		lipgloss.NewStyle().
			Faint(true).
			Italic(true).
			Render(
				"Unavailable chapters inside a range are skipped automatically.",
			),
	)

	b.WriteString("\n\n")

	b.WriteString(
		m.AllCheckbox.View(m.Cursor == 1),
	)

	b.WriteString("\n\n")

	btn := "[ Download ]"

	if m.Cursor == 2 {
		btn = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Render(btn)
	} else {
		btn = lipgloss.NewStyle().
			Faint(true).
			Render(btn)
	}

	b.WriteString(btn)
	b.WriteString("\n\n")

	footerStyle := lipgloss.NewStyle().
		Faint(true)

	b.WriteString(
		footerStyle.Render(
			"↑/↓ navigate • Space/Enter toggle • Enter on Download • Ctrl+C/Esc quit",
		),
	)

	boxWidth := lipgloss.Width(b.String())
	boxHeight := lipgloss.Height(b.String())

	horizontalMargin := max(
		0,
		(m.Width-boxWidth)/2,
	)

	verticalMargin := max(
		0,
		(m.Height-boxHeight)/2,
	)

	boxStyle := lipgloss.NewStyle().
		MarginTop(verticalMargin).
		MarginLeft(horizontalMargin)

	return boxStyle.Render(
		b.String(),
	)
}

func wrapAvailableRanges(
	ranges string,
	width int,
) string {
	var lines []string

	current := ""

	for _, part := range strings.Split(ranges, ",") {
		part = strings.TrimSpace(part)

		if part == "" {
			continue
		}

		next := part

		if current != "" {
			next = current + ", " + part
		}

		if current != "" &&
			lipgloss.Width(next) > width {
			lines = append(lines, current)
			current = part
		} else {
			current = next
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return strings.Join(lines, "\n")
}
