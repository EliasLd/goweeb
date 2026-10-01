package tui

import "github.com/charmbracelet/lipgloss"

var footerStyle = lipgloss.NewStyle().
	Faint(true).
	Align(lipgloss.Center)

func renderViewWithFooter(
	width int,
	height int,
	content string,
	footerText string,
	horizontalAlignment lipgloss.Position,
	verticalAlignment lipgloss.Position,
) string {
	footer := footerStyle.
		Width(width).
		Render(footerText)

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
