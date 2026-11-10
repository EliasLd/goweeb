package tui

import (
	"strings"
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/charmbracelet/bubbles/list"
)

// groupTestRow builds a search result row for a provider.
func groupTestRow(
	title string,
	provider string,
	providerLabel string,
) SelectionItem {
	return SelectionItem{
		Label:         title,
		Value:         provider + "/" + title,
		Provider:      provider,
		ProviderLabel: providerLabel,
	}
}

func TestGroupSearchItemsMergesAcrossProviders(
	t *testing.T,
) {
	items := groupSearchItems([]list.Item{
		groupTestRow(
			"Berserk", "mangadex", "MangaDex (Multilingual)",
		),
		groupTestRow("BERSERK", "atsumoe", "Atsumoe (English)"),
		groupTestRow(
			"Berserk.", "mangafreak", "MangaFreak (English)",
		),
		groupTestRow(
			"Berserk: The Prototype", "mangadex", "MangaDex (Multilingual)",
		),
	})

	if len(items) != 2 {
		t.Fatalf(
			"items = %d, want 2 (one group + one separate work)",
			len(items),
		)
	}

	group, _ := items[0].(SelectionItem)
	single, _ := items[1].(SelectionItem)

	if group.Members == nil {
		t.Fatal("the merged row should carry its members")
	}

	if len(group.Members) != 3 {
		t.Fatalf(
			"members = %d, want 3",
			len(group.Members),
		)
	}

	if got := distinctProviders(group.Members); got != 3 {
		t.Errorf(
			"distinct providers = %d, want 3",
			got,
		)
	}

	// A group keeps the first title of the list it was built from.
	if group.Label != "Berserk" {
		t.Errorf(
			"group label = %q, want %q",
			group.Label,
			"Berserk",
		)
	}

	if single.Members != nil {
		t.Error(
			"Berserk: The Prototype must stay a separate work",
		)
	}

	if single.Value != "mangadex/Berserk: The Prototype" {
		t.Errorf(
			"single row value = %q, want its own URL",
			single.Value,
		)
	}
}

func TestGroupSearchItemsKeepsSingleProviderRowsFlat(
	t *testing.T,
) {
	rows := []list.Item{
		groupTestRow("Berserk", "mangadex", "MangaDex (Multilingual)"),
		groupTestRow("BERSERK", "mangadex", "MangaDex (Multilingual)"),
	}

	items := groupSearchItems(rows)

	// One provider alone never produces a group row, even when it
	// lists the same work twice.
	if len(items) != 2 {
		t.Fatalf(
			"items = %d, want 2 flat rows",
			len(items),
		)
	}

	for i, it := range items {
		row, _ := it.(SelectionItem)

		if len(row.Members) > 1 {
			t.Errorf(
				"row %d should not carry group members",
				i,
			)
		}
	}
}

func TestGroupSearchItemsNeverMergesEmptyKeys(
	t *testing.T,
) {
	rows := []list.Item{
		groupTestRow("!!!", "mangadex", "MangaDex (Multilingual)"),
		groupTestRow("???", "atsumoe", "Atsumoe (English)"),
	}

	items := groupSearchItems(rows)

	if len(items) != 2 {
		t.Fatalf(
			"items = %d, want 2 (titles with no letters or digits stay apart)",
			len(items),
		)
	}
}

func TestGroupSearchItemsStaysIdempotentAcrossArrivals(
	t *testing.T,
) {
	// First arrival: two providers deliver the work.
	regrouped := groupSearchItems([]list.Item{
		groupTestRow("Berserk", "mangadex", "MangaDex (Multilingual)"),
		groupTestRow("BERSERK", "atsumoe", "Atsumoe (English)"),
	})

	// A later arrival merges into the already grouped list.
	regrouped = append(regrouped,
		groupTestRow("Berserk", "mangafreak", "MangaFreak (English)"),
	)

	items := groupSearchItems(regrouped)

	if len(items) != 1 {
		t.Fatalf(
			"items = %d, want 1 (regrouping must not nest group rows)",
			len(items),
		)
	}

	group, _ := items[0].(SelectionItem)

	if len(group.Members) != 3 {
		t.Fatalf(
			"members = %d, want 3",
			len(group.Members),
		)
	}

	for _, member := range group.Members {
		if len(member.Members) > 1 {
			t.Fatal(
				"regrouping must see plain rows only",
			)
		}
	}

	if got := distinctProviders(group.Members); got != 3 {
		t.Errorf(
			"distinct providers = %d, want 3",
			got,
		)
	}
}

func TestGroupSearchItemsFilterValueKeepsTitle(
	t *testing.T,
) {
	items := groupSearchItems([]list.Item{
		groupTestRow("Berserk", "mangadex", "MangaDex (Multilingual)"),
		groupTestRow("BERSERK", "atsumoe", "Atsumoe (English)"),
	})

	group, _ := items[0].(SelectionItem)

	if group.FilterValue() != group.Label {
		t.Errorf(
			"FilterValue() = %q, want the display title %q",
			group.FilterValue(),
			group.Label,
		)
	}
}

func TestSortSearchItemsOrdersBerserkFixture(
	t *testing.T,
) {
	// The real example from the discussion: one query on all
	// providers delivered 16 rows for 7 works. Arrival order
	// reproduced from the paste, with one mixed-case title first
	// so the group carries it for display.
	merged := []list.Item{
		groupTestRow("Berserk", "mangadex", "MangaDex (Multilingual)"),
		groupTestRow("BERSERK", "atsumoe", "Atsumoe (English)"),
		groupTestRow("Berserk", "mangafreak", "MangaFreak (English)"),
		groupTestRow(
			"Berserk", "animesama", "Anime-Sama (French)",
		),
		groupTestRow("Berserk", "mangavyvy", "Mangavyvy (English)"),
		groupTestRow(
			"Berserk", "mangaball", "MangaBall (Multilingual)",
		),
		groupTestRow(
			"Berserk of Gluttony", "atsumoe", "Atsumoe (English)",
		),
		groupTestRow(
			"Berserk of Gluttony", "mangafreak", "MangaFreak (English)",
		),
		groupTestRow(
			"Berserk of Gluttony", "animesama", "Anime-Sama (French)",
		),
		groupTestRow(
			"Berserk of Gluttony", "weebcentral", "WeebCentral (English)",
		),
		groupTestRow(
			"Berserk Memorial", "atsumoe", "Atsumoe (English)",
		),
		groupTestRow(
			"Berserk Gaiden", "atsumoe", "Atsumoe (English)",
		),
		groupTestRow(
			"Berserk Gaiden", "mangadex", "MangaDex (Multilingual)",
		),
		groupTestRow(
			"Berserk: The Prototype", "atsumoe", "Atsumoe (English)",
		),
		groupTestRow(
			"Berserk Official Guidebook", "atsumoe", "Atsumoe (English)",
		),
		groupTestRow(
			"Berserk (Fan Colored)", "mangadex", "MangaDex (Multilingual)",
		),
	}

	items := groupSearchItems(merged)
	sortSearchItems(items, "berserk")

	want := []struct {
		label       string
		memberCount int
	}{
		{"Berserk", 6},
		{"Berserk of Gluttony", 4},
		{"Berserk Gaiden", 2},
		{"Berserk Memorial", 0},
		{"Berserk: The Prototype", 0},
		{"Berserk Official Guidebook", 0},
		{"Berserk (Fan Colored)", 0},
	}

	if len(items) != len(want) {
		t.Fatalf(
			"items = %d, want %d",
			len(items),
			len(want),
		)
	}

	for i, wantI := range want {
		row, _ := items[i].(SelectionItem)

		if row.Label != wantI.label {
			t.Errorf(
				"position %d = %q, want %q",
				i,
				row.Label,
				wantI.label,
			)
		}

		if len(row.Members) != wantI.memberCount {
			t.Errorf(
				"%q members = %d, want %d",
				row.Label,
				len(row.Members),
				wantI.memberCount,
			)
		}
	}
}

func TestInteractiveSearchGroupsOnlyInAllProvidersMode(
	t *testing.T,
) {
	build := func(providerFilter string) Model {
		m := InitialModel()
		m.State = StateInteractiveSearch

		m.SearchInput.SetValue("berserk")
		m.SearchGeneration = 1
		m.SearchRequestGeneration = 1
		m.SearchInFlight = true
		m.SearchProvidersPending = 2
		m.SelectedProvider = providerFilter

		// MangaDex already delivered; Atsumoe answers now with
		// the same title.
		next, _ := Update(
			interactiveSearchResultMsg{
				query:         "berserk",
				generation:    1,
				provider:      "mangadex",
				providerLabel: "MangaDex (Multilingual)",
				results: []sourcetypes.SearchResult{
					{
						Title: "Berserk",
						URL:   "md/berserk",
					},
				},
			},
			m,
		)

		return next
	}

	// All providers: the two rows collapse into one group.
	all := build("")

	next, _ := Update(
		interactiveSearchResultMsg{
			query:         "berserk",
			generation:    1,
			provider:      "atsumoe",
			providerLabel: "Atsumoe (English)",
			results: []sourcetypes.SearchResult{
				{
					Title: "BERSERK",
					URL:   "atsu/berserk",
				},
			},
		},
		all,
	)

	items := next.SearchList.Items()

	if len(items) != 1 {
		t.Fatalf(
			"all-providers items = %d, want 1 group",
			len(items),
		)
	}

	group, _ := items[0].(SelectionItem)

	if len(group.Members) != 2 {
		t.Fatalf(
			"members = %d, want 2",
			len(group.Members),
		)
	}

	// Filtered to one provider: rows stay flat, byte-identical to
	// the ungrouped behavior.
	filtered := build("mangadex")

	next, _ = Update(
		interactiveSearchResultMsg{
			query:         "berserk",
			generation:    1,
			provider:      "atsumoe",
			providerLabel: "Atsumoe (English)",
			results: []sourcetypes.SearchResult{
				{
					Title: "BERSERK",
					URL:   "atsu/berserk",
				},
			},
		},
		filtered,
	)

	items = next.SearchList.Items()

	if len(items) != 2 {
		t.Fatalf(
			"single-provider items = %d, want 2 flat rows",
			len(items),
		)
	}
}

func TestSearchViewRendersProviderCountTags(
	t *testing.T,
) {
	m := InitialModel()
	m.State = StateInteractiveSearch
	m.Width = 80

	m.SearchInput.SetValue("berserk")
	m.SearchResultsQuery = "berserk"

	items := groupSearchItems([]list.Item{
		groupTestRow("Berserk", "mangadex", "MangaDex (Multilingual)"),
		groupTestRow("BERSERK", "atsumoe", "Atsumoe (English)"),
		groupTestRow(
			"Berserk.", "mangafreak", "MangaFreak (English)",
		),
	})

	_ = m.SearchList.SetItems(items)

	m = setInteractiveSearchFocus(
		m,
		searchFocusInput,
	)

	view := viewInteractiveSearch(m)

	if !strings.Contains(view, "[3 providers]") {
		t.Error("a grouped row should advertise its provider count")
	}

	if strings.Contains(view, "[MangaDex (Multilingual)]") {
		t.Error(
			"a grouped row should not advertise one provider label",
		)
	}
}

func TestSortSearchItemsRanksGroupByBestMember(
	t *testing.T,
) {
	// A group's displayed title can rank worse than one of its
	// members ("Berserk!!!" is a prefix match where "Berserk" is
	// an exact one). D4 ranks a group by its best member, so the
	// exact-match group wins over a larger group whose members all
	// rank in the prefix bucket.
	items := []list.Item{
		SelectionItem{
			Label: "Berserk!!!",
			Value: "a",
			Members: []SelectionItem{
				{Label: "Berserk!!!", Value: "a"},
				{Label: "Berserk", Value: "b"},
			},
		},
		SelectionItem{
			Label: "Berserk Of Gluttony",
			Value: "c",
			Members: []SelectionItem{
				{
					Label: "Berserk Of Gluttony",
					Value: "c",
				},
				{
					Label: "Berserk of Gluttony!!",
					Value: "d",
				},
				{
					Label: "Berserk, of Gluttony",
					Value: "e",
				},
			},
		},
	}

	sortSearchItems(items, "berserk")

	first, _ := items[0].(SelectionItem)

	if first.Label != "Berserk!!!" {
		t.Errorf(
			"position 0 = %q, want the group with an exact match inside",
			first.Label,
		)
	}
}
