package tui

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/EliasLd/scan-scraper/internal/app"
	"github.com/EliasLd/scan-scraper/internal/logger"
	"github.com/EliasLd/scan-scraper/internal/source"
	"github.com/EliasLd/scan-scraper/internal/source/common"
	sourcetypes "github.com/EliasLd/scan-scraper/internal/source/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

type logMsg string

type setupLogPipeMsg struct {
	reader *io.PipeReader
}

type catalogSearchResultMsg struct {
	results []sourcetypes.SearchResult
	err     error
}

type scanPathResultMsg struct {
	paths []common.SelectableItem
	err   error
}

type entriesCountResultMsg struct {
	count int
	err   error
}

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	if m.State == StateMangaSelection || m.State == StateScanSelection {
		return handleSelectionUpdate(msg, m)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case catalogSearchResultMsg:
		if msg.err != nil {
			m.Logs = append(m.Logs, errorStyle.Render(fmt.Sprintf("[E] Catalog search failed: %v", msg.err)))
			return m, nil
		}
		if len(msg.results) == 0 {
			m.Logs = append(m.Logs, errorStyle.Render("[E] No manga found"))
			return m, nil
		}

		if len(msg.results) == 1 {
			m.SelectedMangaURL = msg.results[0].URL
			m.Logs = append(m.Logs, "Manga found. Fetching scan versions...")
			return m, fetchScanPaths(m.SelectedMangaURL, strings.TrimSpace(m.DomainInput.Value()))
		}

		items := make([]SelectionItem, len(msg.results))
		for i, r := range msg.results {
			items[i] = SelectionItem{Label: r.Title, Value: r.URL}
		}

		m.SelectionModel = NewSelectionModel("Select a manga", items)
		m.SelectionModel.Width = m.Width
		m.SelectionModel.Height = m.Height
		m.State = StateMangaSelection
		return m, nil

	case scanPathResultMsg:
		if msg.err != nil {
			m.Logs = append(m.Logs, errorStyle.Render(fmt.Sprintf("[E] Failed to get scan paths: %v", msg.err)))
			return m, nil
		}
		if len(msg.paths) == 0 {
			m.Logs = append(m.Logs, errorStyle.Render("[E] No scan versions found"))
			return m, nil
		}

		if len(msg.paths) == 1 {
			m.SelectedScanPath = msg.paths[0].Value
			m.Logs = append(m.Logs, "Scan version selected. Fetching chapter count...")
			return m, fetchEntriesCount(m.SelectedMangaURL, m.SelectedScanPath, strings.TrimSpace(m.DomainInput.Value()))
		}

		items := make([]SelectionItem, len(msg.paths))
		for i, p := range msg.paths {
			items[i] = SelectionItem{Label: p.Label, Value: p.Value}
		}

		m.SelectionModel = NewSelectionModel("Select a version", items)
		m.SelectionModel.Width = m.Width
		m.SelectionModel.Height = m.Height
		m.State = StateScanSelection
		return m, nil

	case entriesCountResultMsg:
		if msg.err != nil {
			m.Logs = append(m.Logs, errorStyle.Render(fmt.Sprintf("[E] Failed to fetch chapter count: %v", msg.err)))
			return m, nil
		}

		m.DiscoveredEntries = msg.count
		m.State = StateRangeSelection
		m.Cursor = 0
		m.AllCheckbox.Checked = false
		m.RangeInput.SetValue("")
		m.RangeInput.Focus()
		return m, nil

	case tea.KeyMsg:
		if m.IsDownloading && msg.String() != "ctrl+c" && msg.String() != "esc" {
			return m, nil
		}

		// Dedicated keyboard handling for range selection screen
		if m.State == StateRangeSelection {
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "up":
				if m.Cursor > 0 {
					m.Cursor--
				}
				m = updateRangeFocus(m)
				return m, nil
			case "down", "tab":
				if m.Cursor < 2 {
					m.Cursor++
				}
				m = updateRangeFocus(m)
				return m, nil
			case " ":
				if m.Cursor == 1 {
					m.AllCheckbox.Toggle()
					m = updateRangeFocus(m)
				}
				return m, nil
			case "enter":
				switch m.Cursor {
				case 1:
					m.AllCheckbox.Toggle()
					m = updateRangeFocus(m)
					return m, nil
				case 2:
					if m.AllCheckbox.Checked || strings.TrimSpace(m.RangeInput.Value()) != "" {
						m.IsDownloading = true
						m.State = StateDownloading
						m.Logs = append(m.Logs, "Starting download...")
						return m, startDownload(m)
					}
					m.Logs = append(m.Logs, errorStyle.Render("[E] Please enter a valid range or enable 'Download all chapters'."))
					return m, nil
				}
			}

			if m.Cursor == 0 && !m.AllCheckbox.Checked {
				var cmd tea.Cmd
				m.RangeInput, cmd = m.RangeInput.Update(msg)
				return m, cmd
			}

			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "up":
			if m.Cursor > 0 {
				m.Cursor--
			}
			m = updateFocus(m)
			return m, nil

		case "down", "tab":
			if m.Cursor < 5 {
				m.Cursor++
			}
			m = updateFocus(m)
			return m, nil

		case " ":
			if m.Cursor == 0 {
				var cmd tea.Cmd
				m.MangaInput, cmd = m.MangaInput.Update(msg)
				return m, cmd
			}
			switch m.Cursor {
			case 3:
				m.EbookCheckbox.Toggle()
			case 4:
				m.KeepCheckbox.Toggle()
			}
			return m, nil

		case "enter":
			switch m.Cursor {
			case 3:
				m.EbookCheckbox.Toggle()
			case 4:
				m.KeepCheckbox.Toggle()
			case 5:
				if m.DownloadReady {
					m.Logs = append(m.Logs, fmt.Sprintf("Searching for: %s...", m.MangaInput.Value()))
					return m, searchCatalog(m.MangaInput.Value(), m.DomainInput.Value())
				}
			}
			return m, nil
		}

		switch m.Cursor {
		case 0:
			var cmd tea.Cmd
			m.MangaInput, cmd = m.MangaInput.Update(msg)
			return m, cmd
		case 1:
			var cmd tea.Cmd
			m.ScanDirInput, cmd = m.ScanDirInput.Update(msg)
			return m, cmd
		case 2:
			var cmd tea.Cmd
			m.DomainInput, cmd = m.DomainInput.Update(msg)
			return m, cmd
		}

	case setupLogPipeMsg:
		m.pipeReader = msg.reader
		m.scanner = bufio.NewScanner(m.pipeReader)
		buf := make([]byte, 64*1024)
		m.scanner.Buffer(buf, 1024*1024)
		return m, readOneLogLine(m)

	case logMsg:
		logLine := string(msg)

		switch {
		case logLine == "Finished downloading":
			styled := highlightStyle.Render("Download complete!")
			m.Logs = append(m.Logs, styled)
			m.IsDownloading = false
			m.State = StateForm
			m.Cursor = 0
			m = updateFocus(m)
		case strings.HasPrefix(logLine, "[DEBUG]"):
		case strings.Contains(logLine, "[ERROR]"):
			m.Logs = append(m.Logs, errorStyle.Render(logLine))
		case strings.Contains(logLine, "[!]"):
			m.Logs = append(m.Logs, logLine)
		default:
			m.Logs = append(m.Logs, logLine)
		}

		if m.IsDownloading {
			return m, readOneLogLine(m)
		}
		return m, nil
	}

	// Main form readiness (manga + destination only)
	m.DownloadReady = strings.TrimSpace(m.MangaInput.Value()) != "" &&
		strings.TrimSpace(m.ScanDirInput.Value()) != ""

	return m, nil
}

func handleSelectionUpdate(msg tea.Msg, m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd
	selectionModel, cmd := m.SelectionModel.Update(msg)
	m.SelectionModel = selectionModel.(SelectionModel)

	if m.SelectionModel.Selected != "" {
		if m.State == StateMangaSelection {
			m.SelectedMangaURL = m.SelectionModel.Selected
			m.State = StateForm
			m.Logs = append(m.Logs, "Manga selected. Fetching scan versions...")
			return m, fetchScanPaths(m.SelectedMangaURL, strings.TrimSpace(m.DomainInput.Value()))
		} else if m.State == StateScanSelection {
			m.SelectedScanPath = m.SelectionModel.Selected
			m.State = StateForm
			m.Logs = append(m.Logs, "Scan version selected. Fetching chapter count...")
			return m, fetchEntriesCount(m.SelectedMangaURL, m.SelectedScanPath, strings.TrimSpace(m.DomainInput.Value()))
		}
	}

	if m.SelectionModel.Cancelled {
		m.State = StateForm
		m.Logs = append(m.Logs, "Selection cancelled")
		return m, nil
	}

	return m, cmd
}

func updateFocus(m Model) Model {
	m.MangaInput.Blur()
	m.ScanDirInput.Blur()
	m.DomainInput.Blur()
	m.RangeInput.Blur()

	switch m.Cursor {
	case 0:
		m.MangaInput.Focus()
	case 1:
		m.ScanDirInput.Focus()
	case 2:
		m.DomainInput.Focus()
	}
	return m
}

func updateRangeFocus(m Model) Model {
	m.RangeInput.Blur()
	if m.Cursor == 0 && !m.AllCheckbox.Checked {
		m.RangeInput.Focus()
	}
	return m
}

func searchCatalog(query, customDomain string) tea.Cmd {
	return func() tea.Msg {
		log := logger.New(io.Discard, logger.LevelInfo)

		provider, err := source.New("animesama", strings.TrimSpace(customDomain))
		if err != nil {
			return catalogSearchResultMsg{results: nil, err: err}
		}

		results, err := provider.Search(query, log)
		return catalogSearchResultMsg{results: results, err: err}
	}
}

func fetchScanPaths(mangaURL string, customDomain string) tea.Cmd {
	return func() tea.Msg {
		log := logger.New(io.Discard, logger.LevelInfo)

		provider, err := source.New("animesama", strings.TrimSpace(customDomain))
		if err != nil {
			return scanPathResultMsg{paths: nil, err: err}
		}

		paths, err := provider.ListScanPaths(mangaURL, log)
		if err != nil {
			return scanPathResultMsg{paths: nil, err: err}
		}

		return scanPathResultMsg{paths: paths, err: nil}
	}
}

func fetchEntriesCount(mangaURL, scanPath, customDomain string) tea.Cmd {
	return func() tea.Msg {
		log := logger.New(io.Discard, logger.LevelInfo)

		provider, err := source.New("animesama", strings.TrimSpace(customDomain))
		if err != nil {
			return entriesCountResultMsg{err: err}
		}

		_, entries, err := provider.ListEntries(mangaURL, scanPath, log)
		if err != nil {
			return entriesCountResultMsg{err: err}
		}

		return entriesCountResultMsg{count: len(entries)}
	}
}

func startDownload(m Model) tea.Cmd {
	return func() tea.Msg {
		opts := app.Options{
			Slug:          m.MangaInput.Value(),
			Source:        "animesama",
			All:           m.AllCheckbox.Checked,
			ScanDir:       m.ScanDirInput.Value(),
			Cleanup:       !m.KeepCheckbox.Checked,
			CustomDomain:  strings.TrimSpace(m.DomainInput.Value()),
			MangaURL:      m.SelectedMangaURL,
			ScanPath:      m.SelectedScanPath,
			EbookFriendly: m.EbookCheckbox.Checked,
		}

		if !opts.All {
			r := strings.TrimSpace(m.RangeInput.Value())
			chapterRange, rangeMode, err := app.ParseRangeString(r)
			if err != nil {
				// Fallback: keep empty range to let backend guard logs report invalid input.
				// This should rarely happen because UI validates before starting.
				opts.RangeMode = app.RangeNone
			} else {
				opts.Range = chapterRange
				opts.RangeMode = rangeMode
			}
		}

		pr, pw := io.Pipe()
		go func() {
			log := logger.New(pw, logger.LevelInfo)
			app.RunWithWorkflow(opts, log)
			_ = pw.Close()
		}()

		return setupLogPipeMsg{reader: pr}
	}
}

func readOneLogLine(m Model) tea.Cmd {
	return func() tea.Msg {
		if m.scanner == nil {
			return nil
		}

		if m.scanner.Scan() {
			return logMsg(m.scanner.Text())
		}

		if err := m.scanner.Err(); err != nil {
			return logMsg(fmt.Sprintf("Error: failed to read log: %v", err))
		}

		return logMsg("Finished downloading")
	}
}
