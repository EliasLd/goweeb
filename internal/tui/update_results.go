package tui

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

// Handles the results of a manga catalog search.
func handleCatalogSearchResult(
	msg catalogSearchResultMsg,
	m Model,
) (Model, tea.Cmd) {
	if msg.err != nil {
		m.Logs = append(
			m.Logs,
			errorStyle.Render(
				fmt.Sprintf(
					"[E] Catalog search failed: %v",
					msg.err,
				),
			),
		)

		return m, nil
	}

	if len(msg.results) == 0 {
		m.Logs = append(
			m.Logs,
			errorStyle.Render("[E] No manga found"),
		)

		return m, nil
	}

	if len(msg.results) == 1 {
		m.SelectedMangaURL = msg.results[0].URL

		m.Logs = append(
			m.Logs,
			fmt.Sprintf(
				"Manga found: %s. Fetching scan versions...",
				msg.results[0].Title,
			),
		)

		return m, fetchScanPaths(
			m.SelectedProvider,
			m.SelectedMangaURL,
			strings.TrimSpace(m.DomainInput.Value()),
		)
	}

	items := make([]SelectionItem, len(msg.results))

	for i, result := range msg.results {
		items[i] = SelectionItem{
			Label: result.Title,
			Value: result.URL,
		}
	}

	m.SelectionModel = NewSelectionModel(
		"Select a manga",
		items,
	)

	m.SelectionModel.Width = m.Width
	m.SelectionModel.Height = m.Height

	m.State = StateMangaSelection

	return m, nil
}

// Handles the available scan versions returned by a provider.
func handleScanPathResult(
	msg scanPathResultMsg,
	m Model,
) (Model, tea.Cmd) {
	if msg.err != nil {
		m.Logs = append(
			m.Logs,
			errorStyle.Render(
				fmt.Sprintf(
					"[E] Failed to get scan paths: %v",
					msg.err,
				),
			),
		)

		return m, nil
	}

	if len(msg.paths) == 0 {
		m.Logs = append(
			m.Logs,
			errorStyle.Render("[E] No scan versions found"),
		)

		return m, nil
	}

	if len(msg.paths) == 1 {
		m.SelectedScanPath = msg.paths[0].Value

		m.Logs = append(
			m.Logs,
			"Scan version selected. Fetching chapters...",
		)

		return m, fetchEntries(
			m.SelectedProvider,
			m.SelectedMangaURL,
			m.SelectedScanPath,
			strings.TrimSpace(m.DomainInput.Value()),
		)
	}

	items := make([]SelectionItem, len(msg.paths))

	for i, path := range msg.paths {
		items[i] = SelectionItem{
			Label: path.Label,
			Value: path.Value,
		}
	}

	m.SelectionModel = NewSelectionModel(
		"Select a version",
		items,
	)

	m.SelectionModel.Width = m.Width
	m.SelectionModel.Height = m.Height

	m.State = StateScanSelection

	return m, nil
}

// Handles the chapter list returned by a provider.
func handleEntriesResult(
	msg entriesResultMsg,
	m Model,
) (Model, tea.Cmd) {
	if msg.err != nil {
		m.Logs = append(
			m.Logs,
			errorStyle.Render(
				fmt.Sprintf(
					"[E] Failed to fetch chapters: %v",
					msg.err,
				),
			),
		)

		return m, nil
	}

	if len(msg.entries) == 0 {
		m.Logs = append(
			m.Logs,
			errorStyle.Render("[E] No chapters found"),
		)

		return m, nil
	}

	m.DiscoveredEntries = len(msg.entries)
	m.DiscoveredEntryList = msg.entries

	m.AvailableRanges = app.FormatAvailableRanges(
		msg.entries,
	)

	m.SelectedRange = app.RangeSelection{}

	m.State = StateRangeSelection
	m.Cursor = 0

	m.AllCheckbox.Checked = false
	m.RangeInput.SetValue("")
	m.RangeInput.Focus()

	return m, nil
}
