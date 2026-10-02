package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func renderCenteredOverlay(
	background string,
	foreground string,
	width int,
	height int,
) string {
	if width <= 0 || height <= 0 {
		return foreground
	}

	foregroundWidth := min(
		lipgloss.Width(foreground),
		width,
	)

	foregroundHeight := min(
		lipgloss.Height(foreground),
		height,
	)

	x := max(
		0,
		(width-foregroundWidth)/2,
	)

	y := max(
		0,
		(height-foregroundHeight)/2,
	)

	return renderOverlayAt(
		background,
		foreground,
		width,
		height,
		x,
		y,
	)
}

func renderOverlayAt(
	background string,
	foreground string,
	width int,
	height int,
	x int,
	y int,
) string {
	backgroundLines := strings.Split(
		background,
		"\n",
	)

	foregroundLines := strings.Split(
		foreground,
		"\n",
	)

	canvas := make(
		[]string,
		height,
	)

	for row := 0; row < height; row++ {
		line := ""

		if row < len(backgroundLines) {
			line = backgroundLines[row]
		}

		canvas[row] = fitLine(
			line,
			width,
		)
	}

	overlayWidth := min(
		lipgloss.Width(foreground),
		width-x,
	)

	for row, overlayLine := range foregroundLines {
		targetRow := y + row

		if targetRow < 0 ||
			targetRow >= height {
			continue
		}

		overlayLine = fitLine(
			overlayLine,
			overlayWidth,
		)

		backgroundLine :=
			canvas[targetRow]

		left := ansi.Cut(
			backgroundLine,
			0,
			x,
		)

		right := ansi.Cut(
			backgroundLine,
			x+overlayWidth,
			width,
		)

		left = fitLine(
			left,
			x,
		)

		canvas[targetRow] =
			left +
				overlayLine +
				right
	}

	return strings.Join(
		canvas,
		"\n",
	)
}

func fitLine(
	line string,
	width int,
) string {
	if width <= 0 {
		return ""
	}

	lineWidth := lipgloss.Width(line)

	if lineWidth > width {
		return ansi.Cut(
			line,
			0,
			width,
		)
	}

	if lineWidth < width {
		line += strings.Repeat(
			" ",
			width-lineWidth,
		)
	}

	return line
}
