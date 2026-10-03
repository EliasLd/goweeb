package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDestinationPickerUsesCurrentDestination(
	t *testing.T,
) {
	m := InitialModel()

	m.DestinationDir = t.TempDir()

	next, _ := openDestinationPicker(m)

	if next.State != StateDestinationPicker {
		t.Fatalf(
			"state = %v, want StateDestinationPicker",
			next.State,
		)
	}

	if next.DestinationPicker.CurrentDirectory !=
		m.DestinationDir {
		t.Errorf(
			"current directory = %q, want %q",
			next.DestinationPicker.CurrentDirectory,
			m.DestinationDir,
		)
	}
}

func TestDestinationPickerEscapeReturnsToForm(
	t *testing.T,
) {
	m := InitialModel()

	original :=
		m.DestinationDir

	m.State =
		StateDestinationPicker

	next, _ :=
		handleDestinationPickerUpdate(
			tea.KeyMsg{
				Type: tea.KeyEsc,
			},
			m,
		)

	if next.State != StateForm {
		t.Fatalf(
			"state = %v, want StateForm",
			next.State,
		)
	}

	if next.DestinationDir != original {
		t.Error(
			"cancelling should not change destination",
		)
	}
}

func TestDestinationPickerSelectsDirectory(
	t *testing.T,
) {
	root := t.TempDir()

	selected := filepath.Join(
		root,
		"manga",
	)

	if err := os.Mkdir(
		selected,
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	m := InitialModel()
	m.DestinationDir = root

	var cmd tea.Cmd

	m, cmd =
		openDestinationPicker(m)

	if cmd == nil {
		t.Fatal(
			"filepicker should schedule directory loading",
		)
	}

	// Feed the filepicker its asynchronous directory
	// listing result.
	msg := cmd()

	m, _ =
		handleDestinationPickerUpdate(
			msg,
			m,
		)

	next, _ :=
		handleDestinationPickerUpdate(
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			m,
		)

	if next.State != StateForm {
		t.Fatalf(
			"state = %v, want StateForm",
			next.State,
		)
	}

	if next.DestinationDir != selected {
		t.Errorf(
			"destination = %q, want %q",
			next.DestinationDir,
			selected,
		)
	}
}

func TestDestinationPickerSupportsVimNavigation(
	t *testing.T,
) {
	root := t.TempDir()

	first := filepath.Join(
		root,
		"alpha",
	)

	second := filepath.Join(
		root,
		"beta",
	)

	for _, path := range []string{
		first,
		second,
	} {
		if err := os.Mkdir(
			path,
			0o755,
		); err != nil {
			t.Fatal(err)
		}
	}

	m := InitialModel()
	m.DestinationDir = root

	m, cmd :=
		openDestinationPicker(m)

	m, _ =
		handleDestinationPickerUpdate(
			cmd(),
			m,
		)

	// j -> second directory
	m, _ =
		handleDestinationPickerUpdate(
			tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{'j'},
			},
			m,
		)

	// Enter -> select it
	m, _ =
		handleDestinationPickerUpdate(
			tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			m,
		)

	if m.DestinationDir != second {
		t.Errorf(
			"destination = %q, want %q",
			m.DestinationDir,
			second,
		)
	}
}
