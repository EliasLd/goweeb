package app

import (
	"fmt"
	"strings"
)

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
