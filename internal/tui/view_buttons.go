package tui

import "github.com/charmbracelet/lipgloss"

type buttonVariant int

const (
	buttonSecondary buttonVariant = iota
	buttonPrimary
)

var (
	primaryButtonColor = lipgloss.AdaptiveColor{
		Light: "#B77900",
		Dark:  "#FFD75F",
	}

	secondaryButtonColor = lipgloss.AdaptiveColor{
		Light: "#333333",
		Dark:  "#FFFFFF",
	}

	buttonBaseStyle = lipgloss.NewStyle().
			Padding(0, 1)

	primaryButtonStyle = buttonBaseStyle.
				Foreground(primaryButtonColor)

	primaryButtonFocusedStyle = buttonBaseStyle.
					Background(primaryButtonColor).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)

	secondaryButtonStyle = buttonBaseStyle.
				Foreground(secondaryButtonColor)

	secondaryButtonFocusedStyle = buttonBaseStyle.
					Background(secondaryButtonColor).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)

	disabledButtonStyle = buttonBaseStyle.
				Foreground(lipgloss.Color("240")).
				Faint(true)

	downloadingTextStyle = lipgloss.NewStyle().
				Foreground(
			lipgloss.AdaptiveColor{
				Light: "#333333",
				Dark:  "#FFFFFF",
			},
		).
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
