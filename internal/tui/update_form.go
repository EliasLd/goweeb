package tui

import (
	"fmt"

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

	case "up":
		if m.Cursor > 0 {
			m.Cursor--

			if app.OutputDirLocked() && m.Cursor == 1 {
				m.Cursor--
			}
		}

		return updateFocus(m), nil

	case "down", "tab":
		if m.Cursor < 4 {
			m.Cursor++

			if app.OutputDirLocked() && m.Cursor == 1 {
				m.Cursor++
			}
		}

		if m.Cursor == 4 && !m.DownloadReady {
			m.Cursor--
		}

		return updateFocus(m), nil

	case "enter", " ":
		switch m.Cursor {
		case 0:
			var cmd tea.Cmd

			m.MangaInput, cmd = m.MangaInput.Update(msg)
			return m, cmd

		case 2:
			return openProviderSelection(m), nil

		case 3:
			m.OptionalCursor = 0
			m.State = StateOptionalSettings

			return m, nil

		case 4:
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
		if app.OutputDirLocked() {
			return m, nil
		}

		var cmd tea.Cmd

		m.ScanDirInput, cmd = m.ScanDirInput.Update(msg)
		m = updateDownloadReady(m)

		return m, cmd
	}

	return updateDownloadReady(m), nil
}
