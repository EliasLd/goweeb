package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var interactiveSearchViewStyle = lipgloss.NewStyle().
	Padding(1, 2)

func viewInteractiveSearch(
	m Model,
) string {
	var b strings.Builder

	b.WriteString(
		titleStyle.Render(
			"Search manga",
		),
	)

	b.WriteString("\n\n")

	providerLabel := m.SelectedProviderLabel

	if providerLabel == "" {
		providerLabel = "All providers"
	}

	providerRow := fmt.Sprintf(
		"Provider: %s ›",
		providerLabel,
	)

	if m.SearchFocus == searchFocusProvider {
		providerRow =
			lipgloss.NewStyle().
				Foreground(
					brightAccentColor,
				).
				Render(
					providerRow,
				)
	} else {
		providerRow =
			lipgloss.NewStyle().
				Faint(true).
				Render(
					providerRow,
				)
	}

	providerRow =
		markMouseZone(
			mouseZoneSearchProvider,
			providerRow,
		)

	b.WriteString(
		providerRow,
	)

	b.WriteString("\n\n")

	searchInput :=
		m.SearchInput.View()

	searchInput =
		markMouseZone(
			mouseZoneSearchInput,
			searchInput,
		)

	b.WriteString(
		searchInput,
	)

	b.WriteString("\n\n")

	query := currentInteractiveSearchQuery(m)

	pending := m.SearchProvidersPending

	switch {
	case len(query) <
		minInteractiveSearchLength:
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					"Type at least 2 characters to search.",
				),
		)

	case m.SearchInFlight &&
		m.SearchRequestGeneration ==
			m.SearchGeneration:
		status := "Searching..."

		if pending > 1 {
			status = fmt.Sprintf(
				"Searching... (%d left)",
				pending,
			)
		}

		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					status,
				),
		)

	case m.SearchPending &&
		m.SearchPendingGeneration ==
			m.SearchGeneration:
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					"Search queued...",
				),
		)

	case m.SearchProvidersPending == 0 &&
		len(m.SearchErrors) > 0 &&
		len(m.SearchList.Items()) == 0:
		b.WriteString(
			errorStyle.Render(
				fmt.Sprintf(
					"Search failed on: %s",
					strings.Join(
						m.SearchErrors,
						", ",
					),
				),
			),
		)

	case m.SearchHasSearched &&
		interactiveSearchResultsCurrent(m) &&
		len(m.SearchList.Items()) == 0:
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					fmt.Sprintf(
						"No results found for %q.",
						query,
					),
				),
		)
	}

	// Some providers failed while others produced results.
	if m.SearchProvidersPending == 0 &&
		len(m.SearchErrors) > 0 &&
		len(m.SearchList.Items()) > 0 {
		b.WriteString("\n\n")

		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					fmt.Sprintf(
						"Some providers failed: %s",
						strings.Join(
							m.SearchErrors,
							", ",
						),
					),
				),
		)
	}

	if len(m.SearchList.Items()) > 0 {
		b.WriteString("\n\n")

		listView := m.SearchList.View()

		if !interactiveSearchResultsCurrent(m) {
			listView = lipgloss.NewStyle().
				Faint(true).
				Render(listView)
		}

		b.WriteString(listView)
	}

	searchWidth := interactiveSeachWidth(m.Width)

	content :=
		interactiveSearchViewStyle.
			Width(searchWidth).
			Render(
				b.String(),
			)

	return renderViewWithFooter(
		m.Width,
		m.Height,
		content,
		"↓/Tab results • ↑ back to search • Enter select • Ctrl+P provider • Esc back to main",
		lipgloss.Center,
		lipgloss.Top,
	)
}
