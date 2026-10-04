package tui

import "github.com/charmbracelet/lipgloss"

var (
	mainAccentColor = lipgloss.AdaptiveColor{
		Light: "#2563EB",
		Dark:  "#69A7FF",
	}

	mainBrightAccentColor = lipgloss.AdaptiveColor{
		Light: "#0891B2",
		Dark:  "#67E8F9",
	}

	mainTextColor = lipgloss.AdaptiveColor{
		Light: "#1E293B",
		Dark:  "#DDE7F2",
	}

	mainMutedColor = lipgloss.AdaptiveColor{
		Light: "#64748B",
		Dark:  "#718096",
	}

	mainBorderColor = lipgloss.AdaptiveColor{
		Light: "#CBD5E1",
		Dark:  "#26384D",
	}

	mainBrandKanaStyle = lipgloss.NewStyle().
				Foreground(mainBrightAccentColor).
				Bold(true)

	mainTaglineStyle = lipgloss.NewStyle().
				Foreground(mainMutedColor).
				Faint(true)

	mainDeckStyle = lipgloss.NewStyle().
			Border(
			lipgloss.RoundedBorder(),
		).
		BorderForeground(
			mainBorderColor,
		).
		Padding(1, 2)

	mainRowLabelStyle = lipgloss.NewStyle().
				Foreground(mainMutedColor).
				Bold(true)

	mainRowValueStyle = lipgloss.NewStyle().
				Foreground(mainTextColor)

	mainRowFocusedLabelStyle = lipgloss.NewStyle().
					Foreground(mainBrightAccentColor).
					Bold(true)

	mainRowFocusedValueStyle = lipgloss.NewStyle().
					Foreground(mainTextColor).
					Bold(true)

	mainRowMarkerStyle = lipgloss.NewStyle().
				Foreground(mainBrightAccentColor).
				Bold(true)

	mainRowArrowStyle = lipgloss.NewStyle().
				Foreground(mainMutedColor)

	mainRowFocusedArrowStyle = lipgloss.NewStyle().
					Foreground(mainBrightAccentColor)

	mainRowDisabledStyle = lipgloss.NewStyle().
				Foreground(mainMutedColor).
				Faint(true)

	mainSearchButtonStyle = lipgloss.NewStyle().
				Foreground(mainAccentColor).
				Bold(true).
				Padding(0, 2)

	mainSearchButtonFocusedStyle = lipgloss.NewStyle().
					Background(mainBrightAccentColor).
					Foreground(lipgloss.Color("#06111C")).
					Bold(true).
					Padding(0, 2)

	mainSearchButtonDisabledStyle = lipgloss.NewStyle().
					Foreground(mainMutedColor).
					Faint(true).
					Padding(0, 2)

	mainDownloadingTextStyle = lipgloss.NewStyle().
					Foreground(mainTextColor).
					Faint(true)

	mainReadyStatusStyle = lipgloss.NewStyle().
				Foreground(mainBrightAccentColor)

	mainBusyStatusStyle = lipgloss.NewStyle().
				Foreground(mainAccentColor)

	mainIdleStatusStyle = lipgloss.NewStyle().
				Foreground(mainMutedColor).
				Faint(true)

	mainShortcutStyle = lipgloss.NewStyle().
				Foreground(mainMutedColor).
				Faint(true)

	mainLinkStyle = lipgloss.NewStyle().
			Foreground(mainAccentColor).
			Faint(true)
)
