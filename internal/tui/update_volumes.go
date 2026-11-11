package tui

import (
	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// handleVolumesResult stores the volume listing a provider exposes
// for the current work. An error or an empty listing keeps the
// chapter flow untouched; the checkbox shows up only when there is
// something real to select.
func handleVolumesResult(
	msg volumesResultMsg,
	m Model,
) (Model, tea.Cmd) {
	if msg.err != nil || len(msg.volumes) == 0 {
		m.DiscoveredVolumes = nil
		m.VolumesCheckbox.Checked = false
		m.SelectedVolumeRange = app.RangeSelection{}

		return m, nil
	}

	m.DiscoveredVolumes = msg.volumes

	return m, nil
}
