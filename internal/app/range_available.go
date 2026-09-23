package app

import (
	"math/big"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

type AvailableRange struct{ Start, End common.ChapterNumber }

func nextInteger(previous, next common.ChapterNumber) bool {
	if !previous.IsInteger() || !next.IsInteger() {
		return false
	}
	value, ok := new(big.Int).SetString(previous.String(), 10)
	if !ok {
		return false
	}
	value.Add(value, big.NewInt(1))
	return value.String() == next.String()
}

func AvailableChapterRanges(entries []sourcetypes.Entry) []AvailableRange {
	if len(entries) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(entries))
	numbers := make([]common.ChapterNumber, 0, len(entries))
	for _, e := range entries {
		if _, ok := seen[e.Number.String()]; ok {
			continue
		}
		seen[e.Number.String()] = struct{}{}
		numbers = append(numbers, e.Number)
	}
	sort.Slice(numbers, func(i, j int) bool { return numbers[i].Compare(numbers[j]) < 0 })
	ranges := make([]AvailableRange, 0, len(numbers))
	start, previous := numbers[0], numbers[0]
	for _, number := range numbers[1:] {
		if nextInteger(previous, number) {
			previous = number
			continue
		}
		ranges = append(ranges, AvailableRange{Start: start, End: previous})
		start, previous = number, number
	}
	return append(ranges, AvailableRange{Start: start, End: previous})
}

func FormatAvailableRanges(entries []sourcetypes.Entry) string {
	ranges := AvailableChapterRanges(entries)
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		if r.Start == r.End {
			parts = append(parts, r.Start.String())
			continue
		}
		parts = append(parts, r.Start.String()+"-"+r.End.String())
	}
	return strings.Join(parts, ", ")
}
