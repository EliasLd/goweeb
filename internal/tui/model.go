package tui

import (
	"bufio"
	"io"

	"github.com/EliasLd/goweeb/internal/app"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AppState int

const (
	StateForm AppState = iota
	StateProviderSelection
	StateMangaSelection
	StateScanSelection
	StateRangeSelection
	StateDownloading
)

// Model contains the current state of the TUI.
type Model struct {
	State AppState

	// Main form.
	Title string

	MangaInput    textinput.Model
	ScanDirInput  textinput.Model
	DomainInput   textinput.Model
	EbookCheckbox Checkbox
	KeepCheckbox  Checkbox

	SelectedProvider      string
	SelectedProviderLabel string
	DownloadReady         bool

	// Chapter range selection.
	RangeInput  textinput.Model
	AllCheckbox Checkbox

	DiscoveredEntries   int
	DiscoveredEntryList []sourcetypes.Entry
	AvailableRanges     string
	SelectedRange       app.RangeSelection

	// Selection screens.
	ProviderSelectionModel ProviderSelectionModel
	SelectionModel         SelectionModel

	// Current workflow.
	SelectedMangaURL string
	SelectedScanPath string
	IsDownloading    bool

	// Terminal layout and navigation.
	Cursor int
	Width  int
	Height int

	// Download logs.
	Logs []string

	pipeReader *io.PipeReader
	scanner    *bufio.Scanner
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
