package tui

import "testing"

func TestUpdateDownloadReady(t *testing.T) {
	m := InitialModel()

	m.MangaInput.SetValue("jjk")
	m.ScanDirInput.SetValue("/tmp/manga")
	m.SelectedProvider = "mangadex"

	m = updateDownloadReady(m)

	if !m.DownloadReady {
		t.Fatal("form should be ready when all required fields are set")
	}

	m.MangaInput.SetValue(" ")

	m = updateDownloadReady(m)

	if m.DownloadReady {
		t.Error("form should not be ready with an empty manga title")
	}

	m.MangaInput.SetValue("jjk")
	m.SelectedProvider = ""

	m = updateDownloadReady(m)

	if m.DownloadReady {
		t.Error("form should not be ready without a provider")
	}
}

func TestUpdateFocus(t *testing.T) {
	m := InitialModel()

	m.Cursor = 1
	m = updateFocus(m)

	if !m.ScanDirInput.Focused() {
		t.Error("destination input should be focused")
	}

	if m.MangaInput.Focused() {
		t.Error("manga input should not remain focused")
	}
}

func TestUpdateRangeFocus(t *testing.T) {
	m := InitialModel()

	m.Cursor = 0
	m.AllCheckbox.Checked = false

	m = updateRangeFocus(m)

	if !m.RangeInput.Focused() {
		t.Error("range input should be focused")
	}

	m.AllCheckbox.Checked = true
	m = updateRangeFocus(m)

	if m.RangeInput.Focused() {
		t.Error("range input should be blurred when all chapters are selected")
	}
}
