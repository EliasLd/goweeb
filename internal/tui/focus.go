package tui

import (
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
)

// Updates the focused input on the main form.
func updateFocus(m Model) Model {
	m.ScanDirInput.Blur()
	m.DomainInput.Blur()
	m.RangeInput.Blur()
	m.SearchInput.Blur()

	if m.Cursor == 0 &&
		!app.OutputDirLocked() {
		m.ScanDirInput.Focus()
	}

	return m
}

// Updates the focused input on the chapter range screen.
func updateRangeFocus(m Model) Model {
	m.RangeInput.Blur()

	if m.Cursor == 0 &&
		!m.AllCheckbox.Checked {
		m.RangeInput.Focus()
	}

	return m
}

// Determines whether the main form can open manga search.
func updateSearchReady(m Model) Model {
	m.SearchReady =
		!m.IsDownloading &&
			strings.TrimSpace(
				m.ScanDirInput.Value(),
			) != "" &&
			strings.TrimSpace(
				m.SelectedProvider,
			) != ""

	return m
}
