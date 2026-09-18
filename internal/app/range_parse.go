package app

import (
	"fmt"
	"strings"
)

type RangeSpec struct {
	Range [2]int
	Mode  RangeMode
}

type RangeSelection struct {
	All    bool
	Ranges []RangeSpec
}

// Parses:
// n, n-m, n-, -n
func ParseRangeString(rangeStr string) ([2]int, RangeMode, error) {
	var chapterRange [2]int
	var rangeMode RangeMode = RangeNone

	rangeStr = strings.TrimSpace(rangeStr)
	if rangeStr == "" {
		return chapterRange, rangeMode, fmt.Errorf("invalid range format: empty input")
	}

	// Handle last N chapters: -N
	if strings.HasPrefix(rangeStr, "-") && len(rangeStr) > 1 {
		var nLast int
		n, err := fmt.Sscanf(rangeStr, "-%d", &nLast)
		if err != nil || n != 1 || nLast <= 0 {
			return chapterRange, rangeMode, fmt.Errorf("invalid range format: %s. Use 1-10, 10, -10, 10- or all", rangeStr)
		}
		chapterRange[0] = 0
		chapterRange[1] = nLast
		rangeMode = RangeLastN
		return chapterRange, rangeMode, nil
	}

	// Handle open-ended: N-
	if strings.HasSuffix(rangeStr, "-") {
		var start int
		trimmed := strings.TrimSuffix(rangeStr, "-")
		n, err := fmt.Sscanf(trimmed, "%d", &start)
		if err != nil || n != 1 || start <= 0 {
			return chapterRange, rangeMode, fmt.Errorf("invalid range format: %s. Use 1-10, 10, -10, 10- or all", rangeStr)
		}
		chapterRange[0] = start
		chapterRange[1] = 0
		rangeMode = RangeOpenEnded
		return chapterRange, rangeMode, nil
	}

	// Handle normal range: N-M
	var start, end int
	n, err := fmt.Sscanf(rangeStr, "%d-%d", &start, &end)
	if err == nil && n == 2 && start > 0 && end > 0 && start <= end {
		chapterRange[0] = start
		chapterRange[1] = end
		rangeMode = RangeNormal
		return chapterRange, rangeMode, nil
	}

	// Handle single chapter: N
	var solo int
	n, err = fmt.Sscanf(rangeStr, "%d", &solo)
	if err == nil && n == 1 && solo > 0 {
		chapterRange[0] = solo
		chapterRange[1] = solo
		rangeMode = RangeNormal
		return chapterRange, rangeMode, nil
	}

	return chapterRange, rangeMode, fmt.Errorf("invalid range format: %s. Use 1-10, 10, -10, 10- or all", rangeStr)
}

func ParseRangeExpression(input string) (RangeSelection, error) {
	var selection RangeSelection

	input = strings.TrimSpace(input)
	if input == "" {
		return selection, fmt.Errorf("invalid range format: empty input")
	}

	if strings.EqualFold(input, "all") {
		selection.All = true
		return selection, nil
	}

	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "" {
			return selection, fmt.Errorf(
				"invalid range expression: %s",
				input,
			)
		}

		chapterRange, mode, err := ParseRangeString(part)
		if err != nil {
			return selection, err
		}

		// "-10" means "last 10 available chapters".
		// Mixing that with explicit numerical ranges would make
		// the semantics unnecessarily confusing.
		if mode == RangeLastN && len(parts) > 1 {
			return selection, fmt.Errorf(
				"last-N syntax cannot be combined with other ranges",
			)
		}

		selection.Ranges = append(
			selection.Ranges,
			RangeSpec{
				Range: chapterRange,
				Mode:  mode,
			},
		)
	}

	return selection, nil
}
