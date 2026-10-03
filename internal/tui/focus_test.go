package tui

import "testing"

func TestUpdateSearchReady(t *testing.T) {
	m := InitialModel()

	m.DestinationDir = "/tmp/manga"
	m.SelectedProvider = "mangadex"

	m = updateSearchReady(m)

	if !m.SearchReady {
		t.Fatal(
			"search should be ready when destination and provider are set",
		)
	}

	m.DestinationDir = " "

	m = updateSearchReady(m)

	if m.SearchReady {
		t.Error(
			"search should not be ready with an empty destination",
		)
	}

	m.DestinationDir = "/tmp/manga"
	m.SelectedProvider = ""

	m = updateSearchReady(m)

	if m.SearchReady {
		t.Error(
			"search should not be ready without a provider",
		)
	}
}

func TestUpdateFocus(t *testing.T) {
	t.Setenv(
		"GOWEEB_OUTPUT_DIR",
		"",
	)

	m := InitialModel()

	m.Cursor = 0
	m = updateFocus(m)

	if m.SearchInput.Focused() {
		t.Error(
			"search input should not be focused on the main form",
		)
	}
}

func TestUpdateRangeFocus(t *testing.T) {
	m := InitialModel()

	m.Cursor = 0
	m.AllCheckbox.Checked = false

	m = updateRangeFocus(m)

	if !m.RangeInput.Focused() {
		t.Error(
			"range input should be focused",
		)
	}

	m.AllCheckbox.Checked = true
	m = updateRangeFocus(m)

	if m.RangeInput.Focused() {
		t.Error(
			"range input should be blurred when all chapters are selected",
		)
	}
}
