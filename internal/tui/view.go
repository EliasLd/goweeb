package tui

import "github.com/charmbracelet/lipgloss"

func View(m Model) string {
	viewModel := m

	if hasBackAction(m.State) {
		viewModel.Height =
			max(
				0,
				m.Height-1,
			)

		viewModel.ProviderSelectionModel.Height =
			viewModel.Height

		viewModel.SelectionModel.Height =
			viewModel.Height
	}

	base :=
		viewCurrentState(
			viewModel,
		)

	if hasBackAction(m.State) {
		base =
			lipgloss.JoinVertical(
				lipgloss.Left,
				renderBackRow(m),
				base,
			)
	}

	if m.LogsVisible &&
		isMainViewState(m.State) {
		base =
			renderCenteredOverlay(
				base,
				viewLogOverlay(m),
				m.Width,
				m.Height,
			)
	}

	rendered :=
		m.AlertModel.Render(base)

	return mouseZones.Scan(
		rendered,
	)
}

func viewCurrentState(m Model) string {
	if m.State == StateDestinationPicker {
		return viewDestinationPicker(m)
	}

	if m.State == StateProviderSelection {
		return m.ProviderSelectionModel.View()
	}

	if m.State == StateScanSelection {
		return m.SelectionModel.View()
	}

	if m.State == StateInteractiveSearch {
		return viewInteractiveSearch(m)
	}

	if m.State == StateRangeSelection {
		return viewRangeSelection(m)
	}

	if m.State == StateOptionalSettings {
		return viewOptionalSettings(m)
	}

	return viewForm(m)
}

func renderBackRow(
	m Model,
) string {
	return lipgloss.NewStyle().
		Width(
			max(
				0,
				m.Width,
			),
		).
		Render(
			renderBackAction(m),
		)
}
