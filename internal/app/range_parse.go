package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/source/common"
)

type RangeSpec struct {
	Start common.ChapterNumber
	End   common.ChapterNumber
	Mode  RangeMode
	Count int // Only for RangeLastN; never used as a chapter number.
}

type RangeSelection struct {
	All    bool
	Ranges []RangeSpec
}

// Parses: 0, 6.5, 1-10.5, 6.5-, -10.
func ParseRangeString(input string) (RangeSpec, error) {
	raw := strings.TrimSpace(input)
	invalid := func() (RangeSpec, error) {
		return RangeSpec{}, fmt.Errorf("invalid range %q; use 0, 6.5, 1-10.5, 6.5- or -10", input)
	}
	if raw == "" {
		return invalid()
	}

	if strings.HasPrefix(raw, "-") {
		count, err := strconv.Atoi(strings.TrimPrefix(raw, "-"))
		if err != nil || count <= 0 {
			return invalid()
		}
		return RangeSpec{Mode: RangeLastN, Count: count}, nil
	}
	if strings.HasSuffix(raw, "-") {
		start, err := common.ParseChapterNumber(strings.TrimSpace(strings.TrimSuffix(raw, "-")))
		if err != nil {
			return invalid()
		}
		return RangeSpec{Mode: RangeOpenEnded, Start: start}, nil
	}
	if strings.Contains(raw, "-") {
		parts := strings.Split(raw, "-")
		if len(parts) != 2 {
			return invalid()
		}
		start, err := common.ParseChapterNumber(strings.TrimSpace(parts[0]))
		if err != nil {
			return invalid()
		}
		end, err := common.ParseChapterNumber(strings.TrimSpace(parts[1]))
		if err != nil || start.Compare(end) > 0 {
			return invalid()
		}
		return RangeSpec{Mode: RangeNormal, Start: start, End: end}, nil
	}
	number, err := common.ParseChapterNumber(raw)
	if err != nil {
		return invalid()
	}
	return RangeSpec{Mode: RangeNormal, Start: number, End: number}, nil
}

func ParseRangeExpression(input string) (RangeSelection, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return RangeSelection{}, fmt.Errorf("invalid range expression: empty input")
	}
	if strings.EqualFold(input, "all") {
		return RangeSelection{All: true}, nil
	}
	parts := strings.Split(input, ",")
	selection := RangeSelection{Ranges: make([]RangeSpec, 0, len(parts))}
	for _, part := range parts {
		spec, err := ParseRangeString(strings.TrimSpace(part))
		if err != nil {
			return RangeSelection{}, err
		}
		if spec.Mode == RangeLastN && len(parts) > 1 {
			return RangeSelection{}, fmt.Errorf("last-N syntax cannot be combined with other ranges")
		}
		selection.Ranges = append(selection.Ranges, spec)
	}
	return selection, nil
}
