package tui

import "github.com/charmbracelet/lipgloss"

var (
	mainBrandKanaStyle = lipgloss.NewStyle().
				Foreground(brightAccentColor).
				Bold(true)

	mainTaglineStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true)

	mainDeckStyle = lipgloss.NewStyle().
			Border(
			lipgloss.RoundedBorder(),
		).
		BorderForeground(
			borderColor,
		).
		Padding(1, 2)

	mainRowLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true)

	mainRowValueStyle = lipgloss.NewStyle().
				Foreground(textColor)

	mainRowFocusedLabelStyle = lipgloss.NewStyle().
					Foreground(brightAccentColor).
					Bold(true)

	mainRowFocusedValueStyle = lipgloss.NewStyle().
					Foreground(textColor).
					Bold(true)

	mainRowMarkerStyle = lipgloss.NewStyle().
				Foreground(brightAccentColor).
				Bold(true)

	mainRowArrowStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	mainRowFocusedArrowStyle = lipgloss.NewStyle().
					Foreground(brightAccentColor)

	mainRowDisabledStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true)

	mainSearchButtonStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Padding(0, 2)

	mainSearchButtonFocusedStyle = lipgloss.NewStyle().
					Background(brightAccentColor).
					Foreground(lipgloss.Color("#06111C")).
					Bold(true).
					Padding(0, 2)

	mainSearchButtonDisabledStyle = lipgloss.NewStyle().
					Foreground(mutedColor).
					Faint(true).
					Padding(0, 2)

	mainDownloadingTextStyle = lipgloss.NewStyle().
					Foreground(textColor).
					Faint(true)

	mainReadyStatusStyle = lipgloss.NewStyle().
				Foreground(brightAccentColor)

	mainBusyStatusStyle = lipgloss.NewStyle().
				Foreground(accentColor)

	mainIdleStatusStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true)
)
