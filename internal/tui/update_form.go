package tui

import (
	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// Handles keyboard input on the main form.
func handleFormUpdate(
	msg tea.KeyMsg,
	m Model,
) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}

		if app.OutputDirLocked() &&
			m.Cursor == 0 {
			m.Cursor = 1
		}

		return updateFocus(m), nil

	case "down", "tab", "j":
		if m.Cursor < 3 {
			m.Cursor++
		}

		if m.Cursor == 3 && !m.SearchReady {
			m.Cursor--
		}

		return updateFocus(m), nil

	case "enter", " ":
		switch m.Cursor {
		case 0:
			return openDestinationPicker(m)
		case 1:
			return openProviderSelection(m), nil

		case 2:
			m.OptionalCursor = 0
			m.State = StateOptionalSettings

			return m, nil

		case 3:
			if m.SearchReady {
				return openInteractiveSearch(m)
			}
		}

		return m, nil
	}

	return updateSearchReady(m), nil
}
