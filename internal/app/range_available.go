package app

import (
	"fmt"
	"sort"
	"strings"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

type AvailableRange struct {
	Start int
	End   int
}

func AvailableChapterRanges(
	entries []sourcetypes.Entry,
) []AvailableRange {
	if len(entries) == 0 {
		return nil
	}

	numbers := make([]int, 0, len(entries))
	seen := make(map[int]struct{})

	for _, entry := range entries {
		if _, exists := seen[entry.Number]; exists {
			continue
		}

		seen[entry.Number] = struct{}{}
		numbers = append(numbers, entry.Number)
	}

	sort.Ints(numbers)

	ranges := make([]AvailableRange, 0)

	start := numbers[0]
	previous := numbers[0]

	for _, number := range numbers[1:] {
		if number == previous+1 {
			previous = number
			continue
		}

		ranges = append(
			ranges,
			AvailableRange{
				Start: start,
				End:   previous,
			},
		)

		start = number
		previous = number
	}

	ranges = append(
		ranges,
		AvailableRange{
			Start: start,
			End:   previous,
		},
	)

	return ranges
}

// Joints ranges of available chapters/volumes in a single string
// according to the format:
// n_1-m_1,n_2-m_2,...,n_n-m_n
func FormatAvailableRanges(
	entries []sourcetypes.Entry,
) string {
	ranges := AvailableChapterRanges(entries)

	parts := make([]string, 0, len(ranges))

	for _, r := range ranges {
		if r.Start == r.End {
			parts = append(
				parts,
				fmt.Sprintf("%d", r.Start),
			)

			continue
		}

		parts = append(
			parts,
			fmt.Sprintf("%d-%d", r.Start, r.End),
		)
	}

	return strings.Join(parts, ", ")
}
