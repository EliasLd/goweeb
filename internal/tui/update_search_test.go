package tui

import (
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInteractiveSearchEscapeReturnsToForm(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEsc,
		},
		m,
	)

	if next.State != StateForm {
		t.Fatalf(
			"state = %v, want StateForm",
			next.State,
		)
	}
}

func TestInteractiveSearchIgnoresStaleResult(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("blue lock")
	m.SearchGeneration = 2
	m.SearchRequestGeneration = 1
	m.SearchInFlight = true

	next, _ := Update(
		interactiveSearchResultMsg{
			query:      "blue",
			generation: 1,
			results: []sourcetypes.SearchResult{
				{
					Title: "Old result",
					URL:   "old",
				},
			},
		},
		m,
	)

	if len(next.SearchList.Items()) != 0 {
		t.Fatal(
			"stale search results should be ignored",
		)
	}
}

func TestInteractiveSearchAppliesCurrentResult(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("blue lock")
	m.SearchGeneration = 1
	m.SearchRequestGeneration = 1
	m.SearchInFlight = true

	next, _ := Update(
		interactiveSearchResultMsg{
			query:      "blue lock",
			generation: 1,
			results: []sourcetypes.SearchResult{
				{
					Title: "Blue Lock",
					URL:   "https://example.com/blue-lock",
				},
			},
		},
		m,
	)

	if len(next.SearchList.Items()) != 1 {
		t.Fatalf(
			"results = %d, want 1",
			len(next.SearchList.Items()),
		)
	}
}

func TestInteractiveSearchDownFocusesResults(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("blue lock")
	m.SearchResultsQuery = "blue lock"
	m.SearchHasSearched = true

	_ = m.SearchList.SetItems(
		[]list.Item{
			SelectionItem{
				Label: "Blue Lock",
				Value: "url",
			},
		},
	)

	m = setInteractiveSearchFocus(
		m,
		searchFocusInput,
	)

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyDown,
		},
		m,
	)

	if next.SearchFocus !=
		searchFocusResults {
		t.Fatal(
			"search results should receive focus",
		)
	}
}
