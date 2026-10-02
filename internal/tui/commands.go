package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/app"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source"
	tea "github.com/charmbracelet/bubbletea"
)

const interactiveSearchDebounce = 400 * time.Millisecond

func debounceInteractiveSearch(
	query string,
	generation uint64,
) tea.Cmd {
	return tea.Tick(
		interactiveSearchDebounce,
		func(time.Time) tea.Msg {
			return interactiveSearchDebounceMsg{
				query:      query,
				generation: generation,
			}
		},
	)
}

// Searches a provider's catalog asynchronously.
func searchInteractiveCatalog(
	providerName string,
	query string,
	customDomain string,
	generation uint64,
) tea.Cmd {
	return func() tea.Msg {
		log := logger.New(
			io.Discard,
			logger.LevelInfo,
		)

		provider, err := source.New(
			providerName,
			strings.TrimSpace(customDomain),
		)
		if err != nil {
			return catalogSearchResultMsg{
				results: nil,
				err:     err,
			}
		}

		results, err := provider.Search(query, log)

		return catalogSearchResultMsg{
			results: results,
			err:     err,
		}
	}
}

// Fetches the available scan versions asynchronously.
func fetchScanPaths(
	providerName string,
	mangaURL string,
	customDomain string,
) tea.Cmd {
	return func() tea.Msg {
		log := logger.New(
			io.Discard,
			logger.LevelInfo,
		)

		provider, err := source.New(
			providerName,
			strings.TrimSpace(customDomain),
		)
		if err != nil {
			return scanPathResultMsg{
				paths: nil,
				err:   err,
			}
		}

		paths, err := provider.ListScanPaths(
			mangaURL,
			log,
		)
		if err != nil {
			return scanPathResultMsg{
				paths: nil,
				err:   err,
			}
		}

		return scanPathResultMsg{
			paths: paths,
			err:   nil,
		}
	}
}

// Fetches the chapters of a selected scan version.
func fetchEntries(
	providerName string,
	mangaURL string,
	scanPath string,
	customDomain string,
) tea.Cmd {
	return func() tea.Msg {
		log := logger.New(
			io.Discard,
			logger.LevelInfo,
		)

		provider, err := source.New(
			providerName,
			strings.TrimSpace(customDomain),
		)
		if err != nil {
			return entriesResultMsg{
				err: err,
			}
		}

		_, entries, err := provider.ListEntries(
			mangaURL,
			scanPath,
			log,
		)
		if err != nil {
			return entriesResultMsg{
				err: err,
			}
		}

		return entriesResultMsg{
			entries: entries,
		}
	}
}

// Starts a download and forwards its logs through a pipe.
func startDownload(m Model) tea.Cmd {
	return func() tea.Msg {
		opts := app.Options{
			Slug:          m.MangaInput.Value(),
			Source:        m.SelectedProvider,
			Selection:     m.SelectedRange,
			ScanDir:       app.ResolveOutputDir(m.ScanDirInput.Value()),
			CustomDomain:  strings.TrimSpace(m.DomainInput.Value()),
			MangaURL:      m.SelectedMangaURL,
			ScanPath:      m.SelectedScanPath,
			EbookFriendly: m.EbookCheckbox.Checked,
		}

		pr, pw := io.Pipe()

		go func() {
			log := logger.New(
				pw,
				logger.LevelInfo,
			)

			app.RunWithWorkflow(
				opts,
				log,
			)

			_ = pw.Close()
		}()

		return setupLogPipeMsg{
			reader: pr,
		}
	}
}

// Reads one log line asynchronously and sends it to Update.
func readOneLogLine(m Model) tea.Cmd {
	return func() tea.Msg {
		if m.scanner == nil {
			return nil
		}

		if m.scanner.Scan() {
			return logMsg(
				m.scanner.Text(),
			)
		}

		if err := m.scanner.Err(); err != nil {
			return logMsg(
				fmt.Sprintf(
					"Error: failed to read log: %v",
					err,
				),
			)
		}

		return logMsg(
			"Finished downloading",
		)
	}
}
