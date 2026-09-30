package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const optionalSettingCount = 1

func handleOptionalSettingsUpdate(
	msg tea.KeyMsg,
	m Model,
) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.State = StateForm

		return updateFocus(m), nil

	case "up", "shift+tab":
		if m.OptionalCursor > 0 {
			m.OptionalCursor--
		}

	case "down", "tab":
		if m.OptionalCursor < optionalSettingCount-1 {
			m.OptionalCursor++
		}

	case " ", "enter":
		switch m.OptionalCursor {
		case 0:
			m.EbookCheckbox.Toggle()
		}
	}

	return m, nil
}
