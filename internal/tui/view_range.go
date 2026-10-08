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

	rangeInputView :=
		m.RangeInput.View()

	if m.AllCheckbox.Checked {
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					rangeInputView,
				),
		)

		b.WriteString("\n")

		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Italic(true).
				Render(
					"Disabled while downloading all chapters.",
				),
		)
	} else {
		if m.Cursor != 0 {
			rangeInputView =
				lipgloss.NewStyle().
					Faint(true).
					Render(
						rangeInputView,
					)
		}

		rangeInputView =
			markMouseZone(
				mouseZoneRangeInput,
				rangeInputView,
			)

		b.WriteString(
			rangeInputView,
		)
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

	allCheckbox :=
		m.AllCheckbox.View(
			m.Cursor == 1,
		)

	allCheckbox =
		markMouseZone(
			mouseZoneRangeAll,
			allCheckbox,
		)

	b.WriteString(
		allCheckbox,
	)

	b.WriteString("\n\n")

	downloadButton :=
		renderButton(
			"Download",
			m.Cursor == 2,
			buttonPrimary,
		)

	downloadButton =
		markMouseZone(
			mouseZoneRangeDownload,
			downloadButton,
		)

	b.WriteString(
		downloadButton,
	)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		b.String(),
		"↑/↓ navigate • Space/Enter select • Ctrl+C/Esc quit",
		lipgloss.Center,
		lipgloss.Center,
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
