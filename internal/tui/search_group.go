package tui

import (
	"github.com/charmbracelet/bubbles/list"

	"github.com/EliasLd/goweeb/internal/metadata"
)

// groupSearchItems regroups merged provider results for display.
//
// Rows whose titles are equal under the metadata exact-title
// normalizer merge into a single row once two or more distinct
// providers back them: the row keeps the first title of the list and
// carries its members for the provider picker. Titles backed by one
// provider only — including a provider listing the same work twice —
// stay flat, exactly as ungrouped results.
//
// The pass is idempotent: a list that already contains group rows is
// flattened back to plain rows before regrouping, so merging it with
// fresh arrivals always sees raw rows.
func groupSearchItems(
	items []list.Item,
) []list.Item {
	flat := flattenSearchGroups(items)

	type group struct {
		rows []SelectionItem
	}

	groups := make([]group, 0, len(flat))
	index := make(map[string]int, len(flat))

	for _, it := range flat {
		row, _ := it.(SelectionItem)

		key := metadata.NormalizeTitle(row.Label)

		// A title with no letters or digits has a key shared with
		// every other such title; never merge on an empty key.
		if key == "" {
			groups = append(groups, group{
				rows: []SelectionItem{row},
			})

			continue
		}

		if at, seen := index[key]; seen {
			groups[at].rows = append(groups[at].rows, row)

			continue
		}

		index[key] = len(groups)
		groups = append(groups, group{
			rows: []SelectionItem{row},
		})
	}

	out := make([]list.Item, 0, len(groups))

	for _, g := range groups {
		if distinctProviders(g.rows) < 2 {
			// Nothing to group: keep the rows exactly as they
			// arrived, never labeled with a provider count.
			for _, row := range g.rows {
				out = append(out, row)
			}

			continue
		}

		first := g.rows[0]

		out = append(out, SelectionItem{
			Label:         first.Label,
			Value:         first.Value,
			Provider:      first.Provider,
			ProviderLabel: first.ProviderLabel,
			Members:       g.rows,
		})
	}

	return out
}

// flattenSearchGroups replaces group rows with their members,
// preserving the member order, so regrouping always starts from
// plain rows.
func flattenSearchGroups(items []list.Item) []list.Item {
	flat := make([]list.Item, 0, len(items))

	for _, it := range items {
		row, ok := it.(SelectionItem)

		if !ok || len(row.Members) <= 1 {
			flat = append(flat, it)

			continue
		}

		for _, member := range row.Members {
			flat = append(flat, member)
		}
	}

	return flat
}

// distinctProviders counts the unique provider ids across rows.
func distinctProviders(rows []SelectionItem) int {
	seen := make(map[string]struct{}, len(rows))

	for _, row := range rows {
		seen[row.Provider] = struct{}{}
	}

	return len(seen)
}
