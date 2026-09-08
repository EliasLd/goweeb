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
	manga.Placeholder = "ex: one piece"
	manga.Focus()
	manga.Prompt = "> "
	manga.CharLimit = 100
	manga.Width = 60

	rangeInput := textinput.New()
	rangeInput.Placeholder = "ex: 7, 1-30, 30- (30 jusqu'à la fin), -5 (5 derniers)"
	rangeInput.Prompt = "> "
	rangeInput.Width = 85

	scanDir := textinput.New()
	scanDir.Placeholder = "ex: C:\\Users\\<username>\\Documents\\scans\\jjk"
	scanDir.Prompt = "> "
	scanDir.SetValue(getDefaultScanDir())
	scanDir.Width = 70

	domain := textinput.New()
	domain.Placeholder = "Optionnel (ex https://anime-sama.tv)"
	domain.Prompt = "> "
	domain.Width = 60

	return Model{
		State:         StateForm,
		Title:         asciiArt,
		MangaInput:    manga,
		AllCheckbox:   Checkbox{Label: "Télécharger tous les chapitres.", Checked: false},
		RangeInput:    rangeInput,
		ScanDirInput:  scanDir,
		DomainInput:   domain,
		EbookCheckbox: Checkbox{Label: "Mode ebook-friendly (dossiers Chapter XXX pour KCC, sans PDF).", Checked: false},
		KeepCheckbox:  Checkbox{Label: "Garder les images après conversion (déconseillé).", Checked: false},
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
