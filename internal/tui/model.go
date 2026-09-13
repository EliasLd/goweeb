package tui

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var asciiArt string = `
                                                            
                                                   ▄▄       
                                                   ██       
  ▄███▄██   ▄████▄  ██      ██  ▄████▄    ▄████▄   ██▄███▄  
 ██▀  ▀██  ██▀  ▀██ ▀█  ██  █▀ ██▄▄▄▄██  ██▄▄▄▄██  ██▀  ▀██ 
 ██    ██  ██    ██  ██▄██▄██  ██▀▀▀▀▀▀  ██▀▀▀▀▀▀  ██    ██ 
 ▀██▄▄███  ▀██▄▄██▀  ▀██  ██▀  ▀██▄▄▄▄█  ▀██▄▄▄▄█  ███▄▄██▀ 
  ▄▀▀▀ ██    ▀▀▀▀     ▀▀  ▀▀     ▀▀▀▀▀     ▀▀▀▀▀   ▀▀ ▀▀▀   
  ▀████▀▀                                                   
                                                            
`

type AppState int

const (
	StateForm AppState = iota
	StateProviderSelection
	StateMangaSelection
	StateScanSelection
	StateRangeSelection
	StateDownloading
)

type Model struct {
	State AppState

	Title         string
	MangaInput    textinput.Model
	AllCheckbox   Checkbox
	RangeInput    textinput.Model
	ScanDirInput  textinput.Model
	KeepCheckbox  Checkbox
	EbookCheckbox Checkbox
	DomainInput   textinput.Model

	SelectedProvider      string
	SelectedProviderLabel string

	ProviderSelectionModel ProviderSelectionModel

	Cursor        int
	Width         int
	Height        int
	DownloadReady bool
	IsDownloading bool
	Logs          []string

	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	scanner    *bufio.Scanner

	SelectionModel SelectionModel

	// Temporary data for multi-step workflow
	SelectedMangaURL string
	SelectedScanPath string

	DiscoveredEntries int
}

func getDefaultScanDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, "Documents")
}

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
		State:         StateForm,
		Title:         asciiArt,
		MangaInput:    manga,
		AllCheckbox:   Checkbox{Label: "Download all chapters.", Checked: false},
		RangeInput:    rangeInput,
		ScanDirInput:  scanDir,
		DomainInput:   domain,
		EbookCheckbox: Checkbox{Label: "Ebook-friendly mode (Chapter XXX folders for KCC, no PDF).", Checked: false},
		KeepCheckbox:  Checkbox{Label: "Keep images after conversion (not recommended).", Checked: false},
		Cursor:        0,
		Width:         0,
		Height:        0,
		DownloadReady: false,
		IsDownloading: false,
		Logs:          []string{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return Update(msg, m)
}

func (m Model) View() string {
	return View(m)
}
