package tui

import (
	"github.com/EliasLd/goweeb/internal/app"
	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	destinationPickerDefaultHeight = 14
	destinationPickerMaxHeight     = 18
)

func newDestinationPicker(
	currentDirectory string,
) filepicker.Model {
	fp := filepicker.New()

	fp.CurrentDirectory = currentDirectory

	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = false

	fp.DirAllowed = true
	fp.FileAllowed = false

	fp.AutoHeight = false

	fp.Styles.Cursor = lipgloss.NewStyle().
		Foreground(
			brightAccentColor,
		)

	fp.Styles.Selected = lipgloss.NewStyle().
		Foreground(
			brightAccentColor,
		).
		Bold(true)

	return fp
}

func openDestinationPicker(
	m Model,
) (Model, tea.Cmd) {
	if app.OutputDirLocked() {
		return m, nil
	}

	m.DestinationPicker =
		newDestinationPicker(
			m.DestinationDir,
		)

	m = resizeDestinationPicker(m)

	m.State = StateDestinationPicker

	return m, m.DestinationPicker.Init()
}

func closeDestinationPicker(
	m Model,
) Model {
	m.State = StateForm
	m.Cursor = 0

	return updateFocus(m)
}

func resizeDestinationPicker(
	m Model,
) Model {
	height := destinationPickerDefaultHeight

	if m.Height > 0 {
		height = min(
			destinationPickerMaxHeight,
			max(
				5,
				m.Height-12,
			),
		)
	}

	m.DestinationPicker.SetHeight(
		height,
	)

	return m
}
