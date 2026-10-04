package tui

import "github.com/charmbracelet/lipgloss"

var (
	accentColor = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#69A7FF",
	}

	brightAccentColor = lipgloss.AdaptiveColor{
		Light: "#0891B2",
		Dark:  "#67E8F9",
	}

	textColor = lipgloss.AdaptiveColor{
		Light: "#1E293B",
		Dark:  "#DDE7F2",
	}

	mutedColor = lipgloss.AdaptiveColor{
		Light: "#64748B",
		Dark:  "#718096",
	}

	borderColor = lipgloss.AdaptiveColor{
		Light: "#CBD5E1",
		Dark:  "#26384D",
	}

	errorColor = lipgloss.AdaptiveColor{
		Light: "#DC2626",
		Dark:  "#FF6B6B",
	}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Align(lipgloss.Center)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor)

	highlightStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Faint(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)
)
