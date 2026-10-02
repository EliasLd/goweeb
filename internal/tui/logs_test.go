package tui

import (
	"strings"
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
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"",
	)

	m := InitialModel()

	m.State = StateDownloading
	m.IsDownloading = true

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlL,
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

func TestLogOverlayOpensWithCtrlL(
	t *testing.T,
) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"",
	)

	m := InitialModel()

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlL,
		},
		m,
	)

	if !next.LogsVisible {
		t.Fatal(
			"log overlay should open with Ctrl+L",
		)
	}
}

func TestLogOverlayClosesWithCtrlL(
	t *testing.T,
) {
	m := InitialModel()
	m.LogsVisible = true

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyCtrlL,
		},
		m,
	)

	if next.LogsVisible {
		t.Fatal(
			"log overlay should close with Ctrl+L",
		)
	}
}

func TestLogOverlayEscapeClosesWithoutQuitting(
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
			"log overlay should close with Esc",
		)
	}

	if cmd != nil {
		t.Error(
			"closing the log overlay should not quit the application",
		)
	}
}

func TestLogOverlayOpensWithLWhenInputIsNotFocused(
	t *testing.T,
) {
	m := InitialModel()

	m.Cursor = 1
	m = updateFocus(m)

	if m.ScanDirInput.Focused() {
		t.Fatal(
			"destination input should not be focused",
		)
	}

	next, _ := Update(
		tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		},
		m,
	)

	if !next.LogsVisible {
		t.Fatal(
			"log overlay should open with l when no text input is focused",
		)
	}
}

func TestAppendLogLineUpdatesVisibleViewport(
	t *testing.T,
) {
	m := InitialModel()

	m.Width = 120
	m.Height = 40
	m = openLogOverlay(m)

	m = appendLogLine(
		m,
		"Downloading Chapter 12...",
	)

	if len(m.Logs) != 1 {
		t.Fatalf(
			"logs count = %d, want 1",
			len(m.Logs),
		)
	}

	if !strings.Contains(
		m.LogViewport.View(),
		"Downloading Chapter 12...",
	) {
		t.Error(
			"visible viewport should contain the appended log",
		)
	}
}
