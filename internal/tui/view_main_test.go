package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestMainViewWidth(
	t *testing.T,
) {
	tests := []struct {
		terminal int
		want     int
	}{
		{
			terminal: 120,
			want:     72,
		},
		{
			terminal: 80,
			want:     72,
		},
		{
			terminal: 60,
			want:     52,
		},
	}

	for _, tt := range tests {
		got :=
			mainViewWidth(
				tt.terminal,
			)

		if got != tt.want {
			t.Errorf(
				"mainViewWidth(%d) = %d, want %d",
				tt.terminal,
				got,
				tt.want,
			)
		}
	}
}

func TestMainOptionsSummary(
	t *testing.T,
) {
	m := InitialModel()

	if got :=
		mainOptionsSummary(m); got != "Default" {
		t.Errorf(
			"summary = %q, want Default",
			got,
		)
	}

	m.EbookCheckbox.Checked = true

	if got :=
		mainOptionsSummary(m); got != "Ebook-friendly" {
		t.Errorf(
			"summary = %q, want Ebook-friendly",
			got,
		)
	}

	m.DomainInput.SetValue(
		"https://example.com",
	)

	if got :=
		mainOptionsSummary(m); got !=
		"Ebook-friendly, Custom domain" {
		t.Errorf(
			"unexpected summary: %q",
			got,
		)
	}
}

func TestMainViewLongDestinationDoesNotExpandLayout(
	t *testing.T,
) {
	m := InitialModel()

	m.Width = 80
	m.Height = 40

	m.DestinationDir =
		"/this/is/a/very/very/very/very/very/very/long/path/to/manga"

	view := viewForm(m)

	if lipgloss.Width(view) > m.Width {
		t.Errorf(
			"view width = %d, terminal width = %d",
			lipgloss.Width(view),
			m.Width,
		)
	}
}
