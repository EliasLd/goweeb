package tui

import (
	"bufio"
	"io"

	"github.com/EliasLd/goweeb/internal/app"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"go.dalton.dog/bubbleup"
)

type AppState int

const (
	StateForm AppState = iota
	StateProviderSelection
	StateInteractiveSearch
	StateScanSelection
	StateRangeSelection
	StateOptionalSettings
	StateDownloading
)

// Model contains the current state of the TUI.
type Model struct {
	State AppState

	// Main form.
	Title string

	ScanDirInput  textinput.Model
	DomainInput   textinput.Model
	EbookCheckbox Checkbox

	SearchInput textinput.Model
	SearchList  list.Model
	SearchFocus searchFocus

	SelectedProvider      string
	SelectedProviderLabel string
	SearchReady           bool

	// Chapter range selection.
	RangeInput  textinput.Model
	AllCheckbox Checkbox

	DiscoveredEntries   int
	DiscoveredEntryList []sourcetypes.Entry
	AvailableRanges     string
	SelectedRange       app.RangeSelection

	SearchGeneration        uint64
	SearchRequestGeneration uint64

	SearchInFlight bool

	SearchPending           bool
	SearchPendingQuery      string
	SearchPendingGeneration uint64

	SearchResultsQuery string
	SearchHasSearched  bool
	SearchError        string

	// Selection screens.
	ProviderSelectionModel ProviderSelectionModel
	SelectionModel         SelectionModel

	// Current workflow.
	SelectedMangaURL string
	SelectedScanPath string
	IsDownloading    bool

	// Terminal layout and navigation.
	Cursor         int
	OptionalCursor int
	Width          int
	Height         int

	// Download logs.
	Logs        []string
	LogsVisible bool
	LogViewport viewport.Model

	// Notifications.
	AlertModel bubbleup.AlertModel

	pipeReader *io.PipeReader
	scanner    *bufio.Scanner
}

func (m Model) Init() tea.Cmd {
	return m.AlertModel.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return Update(msg, m)
}

func (m Model) View() string {
	return View(m)
}
