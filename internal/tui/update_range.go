package tui

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// rangeMaxCursor is the bottom selectable row of the range screen,
// the Download button. The volumes checkbox adds a row whenever the
// current work lists volumes.
func rangeMaxCursor(m Model) int {
	if len(m.DiscoveredVolumes) > 0 {
		return 3
	}

	return 2
}

// Handles keyboard input on the chapter range screen.
func handleRangeUpdate(
	msg tea.KeyMsg,
	m Model,
) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.State = StateForm
		m.Cursor = 3
		m = updateFocus(m)

		return m, nil

	case "up":
		if m.Cursor > 0 {
			m.Cursor--
		}

		return updateRangeFocus(m), nil

	case "down", "tab":
		if m.Cursor < rangeMaxCursor(m) {
			m.Cursor++
		}

		return updateRangeFocus(m), nil

	case " ":
		if m.Cursor == 1 {
			m.AllCheckbox.Toggle()
			m = updateRangeFocus(m)
		} else if m.Cursor == 2 && len(m.DiscoveredVolumes) > 0 {
			m.VolumesCheckbox.Toggle()
			m = updateRangeFocus(m)
		}

		return m, nil

	case "enter":
		switch m.Cursor {
		case 1:
			m.AllCheckbox.Toggle()
			return updateRangeFocus(m), nil

		case 2:
			if len(m.DiscoveredVolumes) > 0 {
				m.VolumesCheckbox.Toggle()
				return updateRangeFocus(m), nil
			}

			return handleRangeSubmit(m)

		case 3:
			return handleRangeSubmit(m)
		}
	}

	// Forward input only when the range field is active.
	if m.Cursor == 0 && !m.AllCheckbox.Checked {
		var cmd tea.Cmd

		m.RangeInput, cmd = m.RangeInput.Update(msg)

		return m, cmd
	}

	return m, nil
}

// handleRangeSubmit starts the download with the selection the
// screen currently shows: volumes when the volumes checkbox is on,
// chapters otherwise. The two never combine.
func handleRangeSubmit(m Model) (Model, tea.Cmd) {
	input := strings.TrimSpace(m.RangeInput.Value())

	want := "'Download all chapters'"
	if m.VolumesCheckbox.Checked {
		want = "'Download all volumes'"
	}

	if input == "" && !m.AllCheckbox.Checked {
		m.Logs = append(
			m.Logs,
			errorStyle.Render(
				fmt.Sprintf(
					"[E] Please enter a range or enable %s.",
					want,
				),
			),
		)

		return m, errorAlertCmd(m)
	}

	var selection app.RangeSelection

	if m.AllCheckbox.Checked {
		selection.All = true
	} else {
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

			return m, errorAlertCmd(m)
		}

		selection = parsed
	}

	if m.VolumesCheckbox.Checked {
		matched := app.FilterVolumesBySelection(
			m.DiscoveredVolumes,
			selection,
		)

		if len(matched) == 0 {
			m.Logs = append(
				m.Logs,
				errorStyle.Render(
					"[E] No available volumes match this selection.",
				),
			)

			return m, errorAlertCmd(m)
		}

		m.SelectedVolumeRange = selection
		m.SelectedRange = app.RangeSelection{}

		m.Logs = append(
			m.Logs,
			fmt.Sprintf(
				"Selected %d volume(s): %s",
				len(matched),
				app.FormatVolumeRanges(matched),
			),
		)
	} else {
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

			return m, errorAlertCmd(m)
		}

		m.SelectedRange = selection
		m.SelectedVolumeRange = app.RangeSelection{}

		m.Logs = append(
			m.Logs,
			fmt.Sprintf(
				"Selected %d chapter(s): %s",
				len(matched),
				app.FormatAvailableRanges(matched),
			),
		)
	}

	m.IsDownloading = true
	m.State = StateDownloading

	m.Cursor = 0

	if app.OutputDirLocked() {
		m.Cursor = 1
	}

	m = updateSearchReady(m)
	m = updateFocus(m)

	m.Logs = append(
		m.Logs,
		"Starting download...",
	)

	return m, tea.Batch(
		startDownload(m),
		m.DownloadSpinner.Tick,
	)
}
