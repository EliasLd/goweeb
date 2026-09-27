package tui

import "github.com/charmbracelet/bubbles/textinput"

// InitialModel creates the initial TUI state.
func InitialModel() Model {
	manga := textinput.New()
	manga.Placeholder = "e.g. one piece"
	manga.Focus()
	manga.Prompt = "> "
	manga.CharLimit = 100
	manga.Width = 60

	rangeInput := textinput.New()
	rangeInput.Placeholder = "e.g. 7, 1-30, 30- (30 to the end), -5 (last 5)"
	rangeInput.Prompt = "> "
	rangeInput.Width = 85

	scanDir := textinput.New()
	scanDir.Placeholder = "ex: C:\\Users\\<username>\\Documents\\scans\\jjk"
	scanDir.Prompt = "> "
	scanDir.SetValue(getDefaultScanDir())
	scanDir.Width = 70

	domain := textinput.New()
	domain.Placeholder = "Optional custom base URL"
	domain.Prompt = "> "
	domain.Width = 60

	return Model{
		State: StateForm,
		Title: asciiArt,

		MangaInput: manga,

		AllCheckbox: Checkbox{
			Label:   "Download all chapters.",
			Checked: false,
		},

		RangeInput:   rangeInput,
		ScanDirInput: scanDir,
		DomainInput:  domain,

		EbookCheckbox: Checkbox{
			Label:   "Ebook-friendly mode (Chapter XXX folders for KCC, no PDF).",
			Checked: false,
		},

		KeepCheckbox: Checkbox{
			Label:   "Keep images after conversion (not recommended).",
			Checked: false,
		},

		Cursor:        0,
		Width:         0,
		Height:        0,
		DownloadReady: false,
		IsDownloading: false,
		Logs:          []string{},
	}
}
