package tui

import (
	"time"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"go.dalton.dog/bubbleup"
)

// InitialModel creates the initial TUI state.
func InitialModel() Model {
	destinationDir :=
		app.DefaultTUIScanDir()

	destinationPicker :=
		newDestinationPicker(
			destinationDir,
		)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search manga..."
	searchInput.Prompt = "> "
	searchInput.CharLimit = 100
	searchInput.Width = 60

	rangeInput := textinput.New()
	rangeInput.Placeholder = "e.g. 7, 1-30, 30- (30 to the end), -5 (last 5)"
	rangeInput.Prompt = "> "
	rangeInput.Width = 85

	domain := textinput.New()
	domain.Placeholder = "Optional custom base URL"
	domain.Prompt = "> "
	domain.Width = 60

	logViewport := viewport.New(0, 0)

	alertModel := bubbleup.NewAlertModel(
		52,
		false,
		5*time.Second,
	).
		WithMinWidth(28).
		WithPosition(
			bubbleup.TopRightPosition,
		).
		WithUnicodePrefix()

	downloadSpinner := spinner.New()
	downloadSpinner.Spinner = spinner.Dot
	downloadSpinner.Style =
		lipgloss.NewStyle().
			Foreground(
				brightAccentColor,
			)

	cursor := 0

	if app.OutputDirLocked() {
		cursor = 1
	}

	m := Model{
		State: StateForm,
		Title: asciiArt,

		DestinationDir:    destinationDir,
		DestinationPicker: destinationPicker,

		AllCheckbox: Checkbox{
			Label:   "Download all chapters.",
			Checked: false,
		},

		RangeInput:  rangeInput,
		DomainInput: domain,

		EbookCheckbox: Checkbox{
			Label:   "Ebook-friendly output",
			Checked: false,
		},

		SearchInput: searchInput,
		SearchList:  newInteractiveSearchList(),
		SearchFocus: searchFocusInput,

		Cursor:         cursor,
		OptionalCursor: 0,

		Width:  0,
		Height: 0,

		AlertModel: alertModel,

		SearchReady:     false,
		IsDownloading:   false,
		DownloadSpinner: downloadSpinner,
		Logs:            []string{},
		LogsVisible:     false,
		LogViewport:     logViewport,
	}

	return updateFocus(m)
}
