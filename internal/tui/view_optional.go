package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var optionalViewStyle = lipgloss.NewStyle().
	Padding(1, 2)

var optionalDescriptionStyle = lipgloss.NewStyle().
	Faint(true).
	PaddingLeft(4)

func viewOptionalSettings(m Model) string {
	var view strings.Builder

	contentWidth := max(20, m.Width-8)

	view.WriteString(
		titleStyle.Render("Optional settings"),
	)

	view.WriteString("\n\n")

	view.WriteString(
		lipgloss.NewStyle().
			Faint(true).
			Width(contentWidth).
			Render(
				"Configure optional behavior for downloaded chapters.",
			),
	)

	view.WriteString("\n\n")

	view.WriteString(
		m.EbookCheckbox.View(
			m.OptionalCursor == 0,
		),
	)

	view.WriteString("\n")

	description := optionalDescriptionStyle.
		Width(contentWidth).
		Render(
			"Save manga pages as images inside per-chapter folders " +
				"instead of generating PDF files. Useful for e-readers " +
				"and tools such as Kindle Comic Converter.",
		)

	view.WriteString(description)

	content := optionalViewStyle.Render(
		view.String(),
	)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		content,
		"↑/↓ navigate • Space/Enter select • Ctrl+C/Esc back",
		lipgloss.Left,
		lipgloss.Top,
	)
}
