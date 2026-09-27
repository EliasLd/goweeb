package tui

import "testing"

func TestInitialModel(t *testing.T) {
	m := InitialModel()

	if m.State != StateForm {
		t.Errorf(
			"initial state = %v, want StateForm",
			m.State,
		)
	}

	if m.Title != asciiArt {
		t.Error("initial title does not match ASCII art")
	}

	if !m.MangaInput.Focused() {
		t.Error("manga input should have initial focus")
	}

	if m.RangeInput.Focused() {
		t.Error("range input should not have initial focus")
	}

	if m.MangaInput.Width != 60 {
		t.Errorf(
			"manga input width = %d, want 60",
			m.MangaInput.Width,
		)
	}

	if m.RangeInput.Width != 85 {
		t.Errorf(
			"range input width = %d, want 85",
			m.RangeInput.Width,
		)
	}

	if got := m.ScanDirInput.Value(); got != getDefaultScanDir() {
		t.Errorf(
			"default scan directory = %q, want %q",
			got,
			getDefaultScanDir(),
		)
	}

	if m.AllCheckbox.Checked ||
		m.EbookCheckbox.Checked ||
		m.KeepCheckbox.Checked {
		t.Error("all checkboxes should be unchecked initially")
	}

	if m.DownloadReady {
		t.Error("download should not be ready initially")
	}

	if m.IsDownloading {
		t.Error("download should not be active initially")
	}

	if m.Cursor != 0 {
		t.Errorf(
			"initial cursor = %d, want 0",
			m.Cursor,
		)
	}

	if m.SelectedProvider != "" {
		t.Error("no provider should be selected initially")
	}
}
