package tui

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// Handles keyboard input on the chapter range screen.
func handleRangeUpdate(
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

		return updateRangeFocus(m), nil

	case "down", "tab":
		if m.Cursor < 2 {
			m.Cursor++
		}

		return updateRangeFocus(m), nil

	case " ":
		if m.Cursor == 1 {
			m.AllCheckbox.Toggle()
			m = updateRangeFocus(m)
		}

		return m, nil

	case "enter":
		switch m.Cursor {
		case 1:
			m.AllCheckbox.Toggle()
			return updateRangeFocus(m), nil

		case 2:
			var selection app.RangeSelection

			if m.AllCheckbox.Checked {
				selection.All = true
			} else {
				input := strings.TrimSpace(
					m.RangeInput.Value(),
				)

				if input == "" {
					m.Logs = append(
						m.Logs,
						errorStyle.Render(
							"[E] Please enter a range or enable 'Download all chapters'.",
						),
					)

					return m, nil
				}

				parsed, err := app.ParseRangeExpression(
					input,
				)
				if err != nil {
					m.Logs = append(
						m.Logs,
						errorStyle.Render(
							fmt.Sprintf(
								"[E] Invalid range: %v",
								err,
							),
						),
					)

					return m, nil
				}

				selection = parsed
			}

			matched := app.FilterEntriesBySelection(
				m.DiscoveredEntryList,
				selection,
			)

			if len(matched) == 0 {
				m.Logs = append(
					m.Logs,
					errorStyle.Render(
						"[E] No available chapters match this selection.",
					),
				)

				return m, nil
			}

			m.SelectedRange = selection

			m.Logs = append(
				m.Logs,
				fmt.Sprintf(
					"Selected %d chapter(s): %s",
					len(matched),
					app.FormatAvailableRanges(matched),
				),
			)

			m.IsDownloading = true
			m.State = StateDownloading

			m.Logs = append(
				m.Logs,
				"Starting download...",
			)

			return m, startDownload(m)
		}
	}

	// Forward input only when the range field is active.
	if m.Cursor == 0 && !m.AllCheckbox.Checked {
		var cmd tea.Cmd

		m.RangeInput, cmd = m.RangeInput.Update(msg)
		m = updateDownloadReady(m)

		return m, cmd
	}

	return m, nil
}
