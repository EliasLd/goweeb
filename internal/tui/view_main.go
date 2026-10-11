package tui

import (
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

const (
	mainViewDefaultWidth = 64
	mainViewMaxWidth     = 72
	mainViewMargin       = 8
)

func mainViewWidth(
	terminalWidth int,
) int {
	if terminalWidth <= 0 {
		return mainViewDefaultWidth
	}

	return min(
		mainViewMaxWidth,
		max(
			1,
			terminalWidth-mainViewMargin,
		),
	)
}

func renderMainLogsAction(
	m Model,
) string {
	button :=
		renderActionButton(
			"Logs",
			m.HoveredAction ==
				mouseZoneMainLogs,
		)

	return markMouseZone(
		mouseZoneMainLogs,
		button,
	)
}

func renderMainStatusRow(
	m Model,
	width int,
) string {
	status :=
		renderMainStatus(m)

	logsButton :=
		renderMainLogsAction(m)

	row :=
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			status,
			"  ",
			logsButton,
		)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(
			row,
		)
}

func renderMainSettingRow(
	label string,
	value string,
	focused bool,
	enabled bool,
	width int,
) string {
	labelWidth := min(
		14,
		max(
			8,
			width/3,
		),
	)

	marker := "  "

	labelStyle :=
		mainRowLabelStyle

	valueStyle :=
		mainRowValueStyle

	arrowStyle :=
		mainRowArrowStyle

	if focused {
		marker =
			mainRowMarkerStyle.Render(
				"▌ ",
			)

		labelStyle =
			mainRowFocusedLabelStyle

		valueStyle =
			mainRowFocusedValueStyle

		arrowStyle =
			mainRowFocusedArrowStyle
	}

	renderedLabel := labelStyle.
		Width(labelWidth).
		Render(label)

	renderedArrow :=
		arrowStyle.Render("›")

	fixedWidth :=
		lipgloss.Width(marker) +
			lipgloss.Width(renderedLabel) +
			lipgloss.Width(" ") +
			lipgloss.Width(" ") +
			lipgloss.Width(renderedArrow)

	valueWidth := max(
		1,
		width-fixedWidth,
	)

	value = truncate.StringWithTail(
		value,
		uint(valueWidth),
		"…",
	)

	renderedValue := valueStyle.
		Width(valueWidth).
		Render(value)

	if !enabled {
		return lipgloss.JoinHorizontal(
			lipgloss.Center,
			"  ",
			mainRowDisabledStyle.
				Width(labelWidth).
				Render(label),
			" ",
			mainRowDisabledStyle.
				Width(valueWidth).
				Render(value),
			"  ",
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		marker,
		renderedLabel,
		" ",
		renderedValue,
		" ",
		renderedArrow,
	)
}

func mainOptionsSummary(
	m Model,
) string {
	var options []string

	if m.EbookCheckbox.Checked {
		options = append(
			options,
			"Ebook-friendly",
		)
	}

	if strings.TrimSpace(
		m.DomainInput.Value(),
	) != "" {
		options = append(
			options,
			"Custom domain",
		)
	}

	if len(options) == 0 {
		return "Default"
	}

	return strings.Join(
		options,
		", ",
	)
}

func renderMainSearchAction(
	m Model,
	width int,
) string {
	container := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)

	if m.IsDownloading {
		status := lipgloss.JoinHorizontal(
			lipgloss.Center,
			m.DownloadSpinner.View(),
			" ",
			mainDownloadingTextStyle.Render(
				"Downloading manga...",
			),
		)

		return container.Render(
			status,
		)
	}

	if !m.SearchReady {
		return container.Render(
			mainSearchButtonDisabledStyle.Render(
				"Search manga",
			),
		)
	}

	style := mainSearchButtonStyle

	if m.Cursor == 3 {
		style =
			mainSearchButtonFocusedStyle
	}

	button := style.Render(
		"Search manga",
	)

	button = markMouseZone(
		mouseZoneMainSearch,
		button,
	)

	return container.Render(
		button,
	)
}

func renderMainStatus(
	m Model,
) string {
	switch {
	case m.IsDownloading:
		return mainBusyStatusStyle.Render(
			"● Downloading",
		)

	case strings.TrimSpace(
		m.SelectedProvider,
	) == "":
		return mainIdleStatusStyle.Render(
			"○ Select a provider to start",
		)

	case m.SearchReady:
		return mainReadyStatusStyle.Render(
			"● Ready",
		)

	default:
		return mainIdleStatusStyle.Render(
			"○ Not ready",
		)
	}
}

func renderMainBrand(
	width int,
) string {
	logo := gradientText(
		asciiArt,
		mainGradientStart,
		mainGradientEnd,
	)

	kana :=
		mainBrandKanaStyle.Render(
			mainBrandKatakana,
		)

	tagline :=
		mainTaglineStyle.Render(
			mainBrandTagline,
		)

	brand := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		kana,
		tagline,
	)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(
			brand,
		)
}

func renderMainDeck(
	m Model,
	width int,
) string {
	innerWidth := max(
		1,
		width-mainDeckStyle.GetHorizontalFrameSize(),
	)

	provider := "Select provider"

	if m.SelectedProviderLabel != "" {
		provider =
			m.SelectedProviderLabel
	}

	destinationEnabled :=
		!app.OutputDirLocked()

	destination := renderMainSettingRow(
		"Destination",
		m.DestinationDir,
		m.Cursor == 0,
		destinationEnabled,
		innerWidth,
	)

	if destinationEnabled {
		destination =
			markMouseZone(
				mouseZoneMainDestination,
				destination,
			)
	}

	providerRow := renderMainSettingRow(
		"Provider",
		provider,
		m.Cursor == 1,
		true,
		innerWidth,
	)

	providerRow =
		markMouseZone(
			mouseZoneMainProvider,
			providerRow,
		)

	options := renderMainSettingRow(
		"Options",
		mainOptionsSummary(m),
		m.Cursor == 2,
		true,
		innerWidth,
	)

	options =
		markMouseZone(
			mouseZoneMainOptions,
			options,
		)

	search := renderMainSearchAction(
		m,
		innerWidth,
	)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		destination,
		"",
		providerRow,
		"",
		options,
		"",
		search,
	)

	return mainDeckStyle.Render(
		content,
	)
}
