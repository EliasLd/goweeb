package app

import (
	"sort"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func FilterEntriesBySelection(entries []sourcetypes.Entry, selection RangeSelection) []sourcetypes.Entry {
	if len(entries) == 0 {
		return nil
	}

	ordered := append([]sourcetypes.Entry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Number.Compare(ordered[j].Number) < 0 })
	unique := make([]sourcetypes.Entry, 0, len(ordered))
	seen := make(map[string]struct{}, len(ordered))
	for _, e := range ordered {
		key := e.Number.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, e)
	}

	if selection.All {
		return unique
	}
	if len(selection.Ranges) == 0 {
		return nil
	}
	if selection.Ranges[0].Mode == RangeLastN {
		count := selection.Ranges[0].Count
		if count > len(unique) {
			count = len(unique)
		}
		return unique[len(unique)-count:]
	}

	selected := make([]sourcetypes.Entry, 0, len(unique))
	for _, e := range unique {
		for _, r := range selection.Ranges {
			matches := r.Mode == RangeOpenEnded && e.Number.Compare(r.Start) >= 0 ||
				r.Mode == RangeNormal && e.Number.Compare(r.Start) >= 0 && e.Number.Compare(r.End) <= 0
			if matches {
				selected = append(selected, e)
				break
			}
		}
	}
	return selected
}
