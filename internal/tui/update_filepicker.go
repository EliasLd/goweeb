package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func handleDestinationPickerUpdate(
	msg tea.Msg,
	m Model,
) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return closeDestinationPicker(m), nil
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		m = resizeDestinationPicker(m)
	}

	var cmd tea.Cmd

	m.DestinationPicker, cmd =
		m.DestinationPicker.Update(msg)

	if didSelect, path :=
		m.DestinationPicker.DidSelectFile(msg); didSelect {

		m.DestinationDir = path

		m.State = StateForm
		m.Cursor = 0

		m = updateSearchReady(m)
		m = updateFocus(m)

		// The filepicker may have returned a command to
		// enter the selected directory. We are leaving the
		// picker instead, so it is intentionally discarded.
		return m, nil
	}

	return m, cmd
}
