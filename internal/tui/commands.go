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

// searchTargets resolves the providers a fan-out search queries.
func searchTargets(
	providerFilter string,
) []source.ProviderInfo {
	if strings.TrimSpace(providerFilter) == "" {
		return source.AvailableProviders()
	}

	for _, provider := range source.AvailableProviders() {
		if provider.ID == strings.TrimSpace(providerFilter) {
			return []source.ProviderInfo{provider}
		}
	}

	return nil
}

// Searches a provider's catalog asynchronously. When the filter is
// empty, every provider is queried concurrently, each provider
// sending its own result message.
func searchInteractiveCatalog(
	providerFilter string,
	query string,
	customDomain string,
	generation uint64,
) tea.Cmd {
	targets := searchTargets(providerFilter)
	if len(targets) == 0 {
		return nil
	}

	// A custom domain belongs to a single provider: it cannot be
	// honored by the all-providers search.
	domain := customDomain

	if strings.TrimSpace(providerFilter) == "" {
		domain = ""
	}

	cmds := make([]tea.Cmd, 0, len(targets))

	for _, target := range targets {
		target := target

		cmds = append(
			cmds,
			func() tea.Msg {
				log := logger.New(
					io.Discard,
					logger.LevelInfo,
				)

				provider, err := source.New(
					target.ID,
					domain,
				)

				if err != nil {
					return interactiveSearchResultMsg{
						query:         query,
						generation:    generation,
						provider:      target.ID,
						providerLabel: target.Label,
						err:           err,
					}
				}

				results, err := provider.Search(
					query,
					log,
				)

				return interactiveSearchResultMsg{
					query:         query,
					generation:    generation,
					provider:      target.ID,
					providerLabel: target.Label,
					results:       results,
					err:           err,
				}
			},
		)
	}

	return tea.Batch(cmds...)
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
			Slug:          m.SearchInput.Value(),
			Source:        m.SelectedMangaProvider,
			Selection:     m.SelectedRange,
			ScanDir:       app.ResolveOutputDir(m.DestinationDir),
			CustomDomain:  strings.TrimSpace(m.DomainInput.Value()),
			MangaURL:      m.SelectedMangaURL,
			ScanPath:      m.SelectedScanPath,
			EbookFriendly: m.EbookCheckbox.Checked,
			CBZ:           m.CBZCheckbox.Checked,
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
