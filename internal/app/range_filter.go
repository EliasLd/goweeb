package app

import (
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func FilterEntriesBySelection(
	entries []sourcetypes.Entry,
	selection RangeSelection,
) []sourcetypes.Entry {
	if selection.All {
		return entries
	}

	if len(selection.Ranges) == 0 {
		return nil
	}

	// Last N is intentionally only allowed as a standalone selector.
	if selection.Ranges[0].Mode == RangeLastN {
		n := selection.Ranges[0].Range[1]

		if n > len(entries) {
			n = len(entries)
		}

		return entries[len(entries)-n:]
	}

	selected := make(
		[]sourcetypes.Entry,
		0,
		len(entries),
	)

	seen := make(map[int]struct{})

	for _, entry := range entries {
		for _, requested := range selection.Ranges {
			start := requested.Range[0]
			end := requested.Range[1]

			matches := false

			switch requested.Mode {
			case RangeOpenEnded:
				matches = entry.Number >= start

			case RangeNormal:
				matches =
					entry.Number >= start &&
						entry.Number <= end
			}

			if !matches {
				continue
			}

			if _, exists := seen[entry.Number]; exists {
				break
			}

			seen[entry.Number] = struct{}{}
			selected = append(selected, entry)

			break
		}
	}

	return selected
}
