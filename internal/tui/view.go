package tui

func View(m Model) string {
	if m.State == StateProviderSelection {
		return m.ProviderSelectionModel.View()
	}

	if m.State == StateMangaSelection ||
		m.State == StateScanSelection {
		return m.SelectionModel.View()
	}

	if m.State == StateRangeSelection {
		return viewRangeSelection(m)
	}

	return viewForm(m)
}
