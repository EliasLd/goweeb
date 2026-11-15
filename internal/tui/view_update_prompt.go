package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/EliasLd/goweeb/internal/version"
)

var (
	updatePromptBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Padding(0, 1)

	updatePromptTitleStyle = lipgloss.NewStyle().
				Foreground(brightAccentColor).
				Bold(true)

	updatePromptRowStyle = lipgloss.NewStyle().
				Foreground(textColor)

	updatePromptHintStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true)
)

// updatePromptInnerWidth sizes the box content alongside the other
// overlays: compact, inside the terminal margins.
func updatePromptInnerWidth(m Model) int {
	return max(1, min(
		48,
		m.Width-12,
	))
}

// viewUpdatePrompt renders the offer box: what is available, what is
// running, and the yes/no choice, unless an install is already
// running.
func viewUpdatePrompt(m Model) string {
	inner := updatePromptInnerWidth(m)

	var b strings.Builder

	b.WriteString(
		updatePromptTitleStyle.Render("Update available"),
	)

	b.WriteString("\n\n")

	for _, line := range []string{
		fmt.Sprintf("Version %s is available.", m.UpdateOffer.Tag),
		fmt.Sprintf("You are running %s.", version.String()),
	} {
		b.WriteString(
			updatePromptRowStyle.
				Width(inner).
				Render(line),
		)

		b.WriteString("\n")
	}

	if m.UpdateRunning {
		b.WriteString("\n")

		b.WriteString(
			updatePromptHintStyle.Render("Updating..."),
		)

		return updatePromptBoxStyle.
			Width(inner).
			Render(b.String())
	}

	b.WriteString("\n\n")

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		renderButton("Yes", m.UpdatePromptCursor == 0, buttonPrimary),
		"  ",
		renderButton("No", m.UpdatePromptCursor == updateNo, buttonSecondary),
	)

	b.WriteString(buttons)

	b.WriteString("\n\n")

	b.WriteString(
		updatePromptHintStyle.Render("Esc keeps the current version"),
	)

	return updatePromptBoxStyle.
		Width(inner).
		Render(b.String())
}
