package tui

func View(m Model) string {
	base := viewCurrentState(m)

	if m.LogsVisible &&
		isMainViewState(m.State) {
		base = renderCenteredOverlay(
			base,
			viewLogOverlay(m),
			m.Width,
			m.Height,
		)
	}

	return m.AlertModel.Render(base)
}

func viewCurrentState(m Model) string {
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
