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

	b.WriteString(
		lipgloss.NewStyle().
			Faint(true).
			Render(
				fmt.Sprintf(
					"Provider: %s",
					m.SelectedProviderLabel,
				),
			),
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
		b.WriteString(
			lipgloss.NewStyle().
				Faint(true).
				Render(
					"Searching...",
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

	case m.SearchError != "":
		b.WriteString(
			errorStyle.Render(
				fmt.Sprintf(
					"Search failed: %s",
					m.SearchError,
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
		"↓/Tab results • ↑ back to search • Enter select • Esc back to main",
		lipgloss.Center,
		lipgloss.Top,
	)
}
