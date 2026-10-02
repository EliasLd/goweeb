package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLogOverlayOpensFromMainView(
	t *testing.T,
) {
	m := InitialModel()

	m.Cursor = 1
	m = updateFocus(m)

	next, _ := Update(
		tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		},
		m,
	)

	if !next.LogsVisible {
		t.Fatal(
			"log overlay should be visible",
		)
	}
}

func TestLogOverlayEscapeClosesOverlay(
	t *testing.T,
) {
	m := InitialModel()
	m.LogsVisible = true

	next, cmd := Update(
		tea.KeyMsg{
			Type: tea.KeyEsc,
		},
		m,
	)

	if next.LogsVisible {
		t.Fatal(
			"log overlay should be closed",
		)
	}

	if cmd != nil {
		t.Error(
			"closing log overlay should not quit the application",
		)
	}
}

func TestLogOverlayCanOpenWhileDownloading(
	t *testing.T,
) {
	m := InitialModel()

	m.State = StateDownloading
	m.IsDownloading = true

	next, _ := Update(
		tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		},
		m,
	)

	if !next.LogsVisible {
		t.Fatal(
			"log overlay should open while downloading",
		)
	}
}

func TestLogShortcutDoesNotStealDestinationInput(
	t *testing.T,
) {
	m := InitialModel()

	m.Cursor = 0
	m = updateFocus(m)

	next, _ := Update(
		tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		},
		m,
	)

	if next.LogsVisible {
		t.Fatal(
			"log overlay should not intercept text input",
		)
	}

	if next.ScanDirInput.Value() ==
		m.ScanDirInput.Value() {
		t.Fatal(
			"destination input should receive the typed character",
		)
	}
}
