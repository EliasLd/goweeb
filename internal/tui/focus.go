package tui

import "strings"

// Updates the focused input on the main form.
func updateFocus(m Model) Model {
	m.MangaInput.Blur()
	m.ScanDirInput.Blur()
	m.DomainInput.Blur()
	m.RangeInput.Blur()

	switch m.Cursor {
	case 0:
		m.MangaInput.Focus()

	case 1:
		m.ScanDirInput.Focus()
	}

	return m
}

// Updates the focused input on the chapter range screen.
func updateRangeFocus(m Model) Model {
	m.RangeInput.Blur()

	if m.Cursor == 0 && !m.AllCheckbox.Checked {
		m.RangeInput.Focus()
	}

	return m
}

// Determines whether the main form can start a search.
func updateDownloadReady(m Model) Model {
	m.DownloadReady =
		strings.TrimSpace(m.MangaInput.Value()) != "" &&
			strings.TrimSpace(m.ScanDirInput.Value()) != "" &&
			strings.TrimSpace(m.SelectedProvider) != ""

	return m
}
