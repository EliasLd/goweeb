package common

import (
	"fmt"
	"strings"
)

// ChapterNumber stores a canonical, non-negative decimal number exactly.
// Examples: "0", "6.5", "14.25", "100". Never convert it to float64.
type ChapterNumber string

func ParseChapterNumber(raw string) (ChapterNumber, error) {
	text := strings.TrimSpace(raw)
	parts := strings.Split(text, ".")
	if len(parts) < 1 || len(parts) > 2 || !decimalDigits(parts[0]) ||
		(len(parts) == 2 && !decimalDigits(parts[1])) {
		return "", fmt.Errorf("invalid chapter number %q", raw)
	}

	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if len(parts) == 1 {
		return ChapterNumber(whole), nil
	}

	fraction := strings.TrimRight(parts[1], "0")
	if fraction == "" {
		return ChapterNumber(whole), nil
	}
	return ChapterNumber(whole + "." + fraction), nil
}

func decimalDigits(text string) bool {
	if text == "" {
		return false
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

func (n ChapterNumber) String() string { return string(n) }

func (n ChapterNumber) IsInteger() bool {
	return !strings.Contains(string(n), ".")
}

func (n ChapterNumber) IntegerDigits() int {
	parts := strings.SplitN(string(n), ".", 2)
	return len(parts[0])
}

func (n ChapterNumber) Padded(width int) string {
	parts := strings.SplitN(string(n), ".", 2)
	whole := parts[0]
	if len(whole) < width {
		whole = strings.Repeat("0", width-len(whole)) + whole
	}
	if len(parts) == 2 {
		return whole + "." + parts[1]
	}
	return whole
}

// Returns -1, 0 or 1. Both operands must come from ParseChapterNumber.
func (n ChapterNumber) Compare(other ChapterNumber) int {
	left := strings.SplitN(string(n), ".", 2)
	right := strings.SplitN(string(other), ".", 2)
	if len(left[0]) < len(right[0]) {
		return -1
	}
	if len(left[0]) > len(right[0]) {
		return 1
	}
	if c := strings.Compare(left[0], right[0]); c != 0 {
		return c
	}

	a, b := "", ""
	if len(left) == 2 {
		a = left[1]
	}
	if len(right) == 2 {
		b = right[1]
	}
	size := max(len(a), len(b))
	a += strings.Repeat("0", size-len(a))
	b += strings.Repeat("0", size-len(b))
	return strings.Compare(a, b)
}
