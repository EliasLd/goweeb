package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const optionalSettingCount = 5

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

			// The output modes are mutually exclusive.
			if m.EbookCheckbox.Checked {
				m.CBZCheckbox.Checked = false
				m.EPUBCheckbox.Checked = false
			}

		case 1:
			m.CBZCheckbox.Toggle()

			if m.CBZCheckbox.Checked {
				m.EbookCheckbox.Checked = false
				m.EPUBCheckbox.Checked = false
			}

		case 2:
			m.EPUBCheckbox.Toggle()

			if m.EPUBCheckbox.Checked {
				m.EbookCheckbox.Checked = false
				m.CBZCheckbox.Checked = false
			}

		case 3:
			// Independent of the output mode.
			m.ContinueCheckbox.Toggle()

		case 4:
			// Independent of the output mode.
			m.ExternalCoversCheckbox.Toggle()
		}
	}

	return m, nil
}
