package tui

import "github.com/charmbracelet/lipgloss"

type buttonVariant int

const (
	buttonSecondary buttonVariant = iota
	buttonPrimary
)

var (
	buttonBaseStyle = lipgloss.NewStyle().
			Padding(0, 2)

	primaryButtonStyle = buttonBaseStyle.
				Foreground(accentColor).
				Bold(true)

	primaryButtonFocusedStyle = buttonBaseStyle.
					Background(brightAccentColor).
					Foreground(lipgloss.Color("#06111C")).
					Bold(true)

	secondaryButtonStyle = buttonBaseStyle.
				Foreground(textColor)

	secondaryButtonFocusedStyle = buttonBaseStyle.
					Background(borderColor).
					Foreground(brightAccentColor).
					Bold(true)

	disabledButtonStyle = buttonBaseStyle.
				Foreground(mutedColor).
				Faint(true)
)

func renderButton(
	label string,
	focused bool,
	variant buttonVariant,
) string {
	switch variant {
	case buttonPrimary:
		if focused {
			return primaryButtonFocusedStyle.Render(label)
		}

		return primaryButtonStyle.Render(label)

	default:
		if focused {
			return secondaryButtonFocusedStyle.Render(label)
		}

		return secondaryButtonStyle.Render(label)
	}
}
