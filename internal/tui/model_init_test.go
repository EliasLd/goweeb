package tui

import (
	"testing"

	"github.com/EliasLd/goweeb/internal/app"
)

func TestInitialModel(t *testing.T) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"",
	)

	m := InitialModel()

	if m.State != StateForm {
		t.Errorf(
			"initial state = %v, want StateForm",
			m.State,
		)
	}

	if m.Title != asciiArt {
		t.Error(
			"initial title does not match ASCII art",
		)
	}

	if m.SearchInput.Focused() {
		t.Error(
			"search input should not have initial focus",
		)
	}

	if m.RangeInput.Focused() {
		t.Error(
			"range input should not have initial focus",
		)
	}

	if m.SearchInput.Width != 60 {
		t.Errorf(
			"search input width = %d, want 60",
			m.SearchInput.Width,
		)
	}

	if m.RangeInput.Width != 85 {
		t.Errorf(
			"range input width = %d, want 85",
			m.RangeInput.Width,
		)
	}

	if got := m.DestinationDir; got != app.DefaultTUIScanDir() {
		t.Errorf(
			"default scan directory = %q, want %q",
			got,
			app.DefaultTUIScanDir(),
		)
	}

	if m.AllCheckbox.Checked ||
		m.EbookCheckbox.Checked {
		t.Error(
			"all checkboxes should be unchecked initially",
		)
	}

	if m.SearchReady {
		t.Error(
			"search should not be ready initially",
		)
	}

	if m.IsDownloading {
		t.Error(
			"download should not be active initially",
		)
	}

	if m.Cursor != 0 {
		t.Errorf(
			"initial cursor = %d, want 0",
			m.Cursor,
		)
	}

	if m.SelectedProvider != "" {
		t.Error(
			"no provider should be selected initially",
		)
	}
}

func TestInitialModelSkipsLockedOutputDir(
	t *testing.T,
) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"/home/goweeb/Documents",
	)

	m := InitialModel()

	if m.Cursor != 1 {
		t.Errorf(
			"initial cursor = %d, want 1 when output directory is locked",
			m.Cursor,
		)
	}
}
