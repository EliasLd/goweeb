package tui

import (
	"fmt"

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

	case "up":
		if m.Cursor > 0 {
			m.Cursor--
		}

		return updateFocus(m), nil

	case "down", "tab":
		if m.Cursor < 5 {
			m.Cursor++
		}

		return updateFocus(m), nil

	case " ":
		if m.Cursor == 0 {
			var cmd tea.Cmd

			m.MangaInput, cmd = m.MangaInput.Update(msg)
			return m, cmd
		}

		switch m.Cursor {
		case 2:
			return openProviderSelection(m), nil

		case 3:
			m.EbookCheckbox.Toggle()

		case 4:
			m.KeepCheckbox.Toggle()
		}

		return m, nil

	case "enter":
		switch m.Cursor {
		case 2:
			return openProviderSelection(m), nil

		case 3:
			m.EbookCheckbox.Toggle()

		case 4:
			m.KeepCheckbox.Toggle()

		case 5:
			if m.DownloadReady {
				m.Logs = append(
					m.Logs,
					fmt.Sprintf(
						"Searching for: %s...",
						m.MangaInput.Value(),
					),
				)

				return m, searchCatalog(
					m.SelectedProvider,
					m.MangaInput.Value(),
					m.DomainInput.Value(),
				)
			}
		}

		return m, nil
	}

	switch m.Cursor {
	case 0:
		var cmd tea.Cmd

		m.MangaInput, cmd = m.MangaInput.Update(msg)
		m = updateDownloadReady(m)

		return m, cmd

	case 1:
		var cmd tea.Cmd

		m.ScanDirInput, cmd = m.ScanDirInput.Update(msg)
		m = updateDownloadReady(m)

		return m, cmd
	}

	return updateDownloadReady(m), nil
}
