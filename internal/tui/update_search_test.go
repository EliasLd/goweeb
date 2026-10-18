package tui

import (
	"fmt"
	"testing"

	"github.com/EliasLd/goweeb/internal/app"
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

func TestInteractiveSearchMergesAndSortsProviderResults(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("blue lock")
	m.SearchGeneration = 1
	m.SearchRequestGeneration = 1
	m.SearchInFlight = true
	m.SearchProvidersPending = 2

	// First provider answers with an exact match and a weak match.
	next, _ := Update(
		interactiveSearchResultMsg{
			query:         "blue lock",
			generation:    1,
			provider:      "mangadex",
			providerLabel: "MangaDex (Multilingual)",
			results: []sourcetypes.SearchResult{
				{
					Title: "Blue Lock",
					URL:   "md/blue-lock",
				},
				{
					Title: "Locks and Keys",
					URL:   "md/locks",
				},
			},
		},
		m,
	)

	if next.SearchProvidersPending != 1 {
		t.Fatalf(
			"pending = %d, want 1",
			next.SearchProvidersPending,
		)
	}

	// Second provider answers with a prefix match; the merged list
	// must place the exact match first.
	m = next

	next, _ = Update(
		interactiveSearchResultMsg{
			query:         "blue lock",
			generation:    1,
			provider:      "atsumoe",
			providerLabel: "Atsumoe (English)",
			results: []sourcetypes.SearchResult{
				{
					Title: "Blue Lock: Episode Nagi",
					URL:   "atsu/nagi",
				},
			},
		},
		m,
	)

	if next.SearchProvidersPending != 0 {
		t.Errorf(
			"pending = %d, want 0",
			next.SearchProvidersPending,
		)
	}

	if next.SearchInFlight {
		t.Error("search should not be in flight anymore")
	}

	if !next.SearchHasSearched {
		t.Error("search should be marked as searched")
	}

	expected := []struct {
		title    string
		provider string
	}{
		{"Blue Lock", "mangadex"},
		{"Blue Lock: Episode Nagi", "atsumoe"},
		{"Locks and Keys", "mangadex"},
	}

	items := next.SearchList.Items()

	if len(items) != len(expected) {
		t.Fatalf(
			"items = %d, want %d",
			len(items),
			len(expected),
		)
	}

	for i, want := range expected {
		item, _ := items[i].(SelectionItem)

		if item.Label != want.title {
			t.Errorf(
				"item %d = %q, want %q",
				i,
				item.Label,
				want.title,
			)
		}

		if item.Provider != want.provider {
			t.Errorf(
				"item %q provider = %q, want %q",
				item.Label,
				item.Provider,
				want.provider,
			)
		}

		if item.ProviderLabel == "" {
			t.Errorf(
				"item %q should carry its provider label",
				item.Label,
			)
		}
	}
}

func TestInteractiveSearchKeepsResultsWhenProviderFails(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("blue lock")
	m.SearchGeneration = 1
	m.SearchRequestGeneration = 1
	m.SearchInFlight = true
	m.SearchProvidersPending = 2

	next, _ := Update(
		interactiveSearchResultMsg{
			query:         "blue lock",
			generation:    1,
			provider:      "mangadex",
			providerLabel: "MangaDex (Multilingual)",
			results: []sourcetypes.SearchResult{
				{
					Title: "Blue Lock",
					URL:   "md/blue-lock",
				},
			},
		},
		m,
	)

	// One provider fails…
	m = next

	next, _ = Update(
		interactiveSearchResultMsg{
			query:         "blue lock",
			generation:    1,
			provider:      "animesama",
			providerLabel: "Anime-Sama (French)",
			err:           fmt.Errorf("network down"),
		},
		m,
	)

	if next.SearchProvidersPending != 0 {
		t.Fatalf(
			"pending = %d, want 0",
			next.SearchProvidersPending,
		)
	}

	if len(next.SearchList.Items()) != 1 {
		t.Fatalf(
			"items = %d, want 1 (a failing provider must not wipe results)",
			len(next.SearchList.Items()),
		)
	}

	if len(next.SearchErrors) != 1 ||
		next.SearchErrors[0] != "Anime-Sama (French)" {
		t.Errorf(
			"search errors = %v, want [Anime-Sama (French)]",
			next.SearchErrors,
		)
	}
}

func TestInteractiveSearchSelectsProviderOfResult(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch

	m.SearchInput.SetValue("blue lock")
	m.SearchResultsQuery = "blue lock"
	m.SearchHasSearched = true

	listCmd := m.SearchList.SetItems(
		[]list.Item{
			SelectionItem{
				Label:         "Blue Lock",
				Value:         "atsu/blue-lock",
				Provider:      "atsumoe",
				ProviderLabel: "Atsumoe (English)",
			},
		},
	)
	_ = listCmd

	m = setInteractiveSearchFocus(
		m,
		searchFocusResults,
	)

	next, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if next.State != StateForm {
		t.Fatalf("state = %v, want StateForm", next.State)
	}

	// The selected result adopts its provider for the scan /
	// chapters / download flow.
	if next.SelectedMangaProvider != "atsumoe" {
		t.Errorf(
			"selected manga provider = %q, want atsumoe",
			next.SelectedMangaProvider,
		)
	}

	// The persistent search filter stays untouched: it only
	// changes on the provider selection screen.
	if next.SelectedProvider != "" {
		t.Errorf(
			"selected provider = %q, want empty (all providers)",
			next.SelectedProvider,
		)
	}

	if next.SelectedProviderLabel != "" {
		t.Errorf(
			"selected provider label = %q, want empty",
			next.SelectedProviderLabel,
		)
	}
}

func TestInteractiveSearchKeepsFilterAfterResultSelection(
	t *testing.T,
) {
	m := InitialModel()
	m.DestinationDir = app.DefaultTUIScanDir()

	// Selecting a result (all-providers filter) adopts the result's
	// provider for the flow, then returns to the main form; the
	// user then re-enters the search view.
	m.State = StateInteractiveSearch
	m.SearchInput.SetValue("berserk")
	m.SearchResultsQuery = "berserk"
	m.SearchHasSearched = true

	_ = m.SearchList.SetItems(
		[]list.Item{
			SelectionItem{
				Label:         "Berserk",
				Value:         "mangadot/42",
				Provider:      "mangadotnet",
				ProviderLabel: "MangaDot (English)",
			},
		},
	)

	m = setInteractiveSearchFocus(
		m,
		searchFocusResults,
	)

	selected, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		m,
	)

	if selected.State != StateForm {
		t.Fatalf("state = %v, want StateForm", selected.State)
	}

	// Re-entering the search view restores the filter: hit the
	// search button again (cursor 2, where the selection left it).
	opened, _ := Update(
		tea.KeyMsg{
			Type: tea.KeyEnter,
		},
		selected,
	)

	if opened.State != StateInteractiveSearch {
		t.Fatalf(
			"state = %v, want StateInteractiveSearch",
			opened.State,
		)
	}

	if opened.SelectedProvider != "" ||
		opened.SelectedProviderLabel != "" {
		t.Errorf(
			"filter = %q / %q, want empty (all providers)",
			opened.SelectedProvider,
			opened.SelectedProviderLabel,
		)
	}

	if opened.SearchInput.Placeholder != "Search across all providers..." {
		t.Errorf(
			"placeholder = %q, want the all-providers one",
			opened.SearchInput.Placeholder,
		)
	}

	if opened.SelectedMangaProvider != "mangadotnet" {
		t.Errorf(
			"selected manga provider = %q, want mangadotnet",
			opened.SelectedMangaProvider,
		)
	}
}
