package tui

import (
	"github.com/EliasLd/goweeb/internal/version"
	"github.com/charmbracelet/lipgloss"
)

var footerStyle = lipgloss.NewStyle().
	Faint(true).
	Align(lipgloss.Center)

// footerVersionStyle matches the helper's faint tone; the view turns
// it accent while an update is available.
var footerVersionStyle = lipgloss.NewStyle().
	Faint(true)

// renderViewWithFooter lays the work out above a footer row that
// carries the helper text centered and the running version in the
// bottom-left corner.
func renderViewWithFooter(
	width int,
	height int,
	content string,
	footerText string,
	horizontalAlignment lipgloss.Position,
	verticalAlignment lipgloss.Position,
) string {
	footer := renderFooterRow(width, footerText)

	footerHeight := lipgloss.Height(footer)
	bodyHeight := max(0, height-footerHeight)

	contentWidth := lipgloss.Width(content)

	leftMargin := 0

	switch horizontalAlignment {
	case lipgloss.Center:
		leftMargin = max(
			0,
			(width-contentWidth)/2,
		)

	case lipgloss.Right:
		leftMargin = max(
			0,
			width-contentWidth,
		)
	}

	positionedContent := lipgloss.NewStyle().
		MarginLeft(leftMargin).
		Render(content)

	body := lipgloss.PlaceVertical(
		bodyHeight,
		verticalAlignment,
		positionedContent,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		body,
		footer,
	)
}

// renderFooterRow centers the helper on the footer line and rides
// the version label in its blank left margin, keeping the helper
// untouched in the middle. On terminals too narrow to carry both,
// the label is dropped rather than pushed into the view.
func renderFooterRow(width int, helperText string) string {
	label := footerVersionStyle.Render(version.String())
	labelWidth := lipgloss.Width(label)

	row := footerStyle.
		Width(width).
		Render(helperText)

	leading := 0

	if helperWidth := lipgloss.Width(helperText); helperWidth < width {
		leading = (width - helperWidth) / 2
	}

	if leading >= labelWidth {
		row = label + row[labelWidth:]
	}

	return row
}
