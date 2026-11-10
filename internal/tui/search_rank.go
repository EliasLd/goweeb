package tui

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/list"
)

// rankSearchResult compares a search result title against the query
// and returns a relevance bucket. Lower buckets are more accurate:
// 0 = exact match, 1 = word-boundary prefix, 2 = whole word
// containment, 3 = substring, 4 = no match.
func rankSearchResult(query, title string) int {
	q := normalizeSearchText(query)
	t := normalizeSearchText(title)

	if q == "" || t == "" {
		return 4
	}

	if t == q {
		return 0
	}

	if startsWithWord(t, q) {
		return 1
	}

	if containsAsWord(t, q) {
		return 2
	}

	if strings.Contains(t, q) {
		return 3
	}

	return 4
}

// normalizeSearchText trims and lowercases a search text so ranking
// stays case-insensitive and ignores leading/trailing spaces.
func normalizeSearchText(text string) string {
	return strings.ToLower(
		strings.TrimSpace(text),
	)
}

// isWordRune reports whether r counts as part of a word when
// computing word boundaries.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// startsWithWord reports whether title starts with query followed by
// a word boundary (the end of the title or a non alphanumeric rune).
func startsWithWord(title, query string) bool {
	rest := strings.TrimPrefix(title, query)
	if len(rest) == len(title) {
		return false
	}

	if rest == "" {
		return true
	}

	r, _ := utf8.DecodeRuneInString(rest)

	return !isWordRune(r)
}

// containsAsWord reports whether title contains query as a whole
// word, delimited on both sides by word boundaries.
func containsAsWord(title, query string) bool {
	if query == "" {
		return false
	}

	offset := 0

	for {
		index := strings.Index(title[offset:], query)
		if index < 0 {
			return false
		}

		index += offset

		if isBoundaryBefore(title, index) &&
			isBoundaryAfter(title, index+len(query)) {
			return true
		}

		offset = index + 1
	}
}

// isBoundaryBefore reports whether the rune just before the byte
// index is a word boundary (start of string or non alphanumeric).
func isBoundaryBefore(title string, index int) bool {
	if index == 0 {
		return true
	}

	r, _ := utf8.DecodeLastRuneInString(title[:index])

	return !isWordRune(r)
}

// isBoundaryAfter reports whether the rune at the byte index is a
// word boundary (end of string or non alphanumeric).
func isBoundaryAfter(title string, index int) bool {
	if index >= len(title) {
		return true
	}

	r, _ := utf8.DecodeRuneInString(title[index:])

	return !isWordRune(r)
}

// sortSearchItems orders result items for display: best relevance
// bucket first, then the widest provider support — a work backed by
// several providers is more likely what the user meant — then the
// stable merge order. The sort is stable so ties keep the order the
// providers delivered.
func sortSearchItems(
	items []list.Item,
	query string,
) {
	sort.SliceStable(items, func(i, j int) bool {
		left, _ := items[i].(SelectionItem)
		right, _ := items[j].(SelectionItem)

		leftBucket := bestSearchBucket(query, left)
		rightBucket := bestSearchBucket(query, right)

		if leftBucket != rightBucket {
			return leftBucket < rightBucket
		}

		return searchItemProviderCount(left) >
			searchItemProviderCount(right)
	})
}

// bestSearchBucket ranks a display row by its most relevant
// underlying result: rows of one group can differ in ranking because
// their displayed titles differ while their normalized keys match.
func bestSearchBucket(query string, row SelectionItem) int {
	best := rankSearchResult(query, row.Label)

	for _, member := range row.Members {
		if bucket := rankSearchResult(query, member.Label); bucket < best {
			best = bucket
		}
	}

	return best
}

// searchItemProviderCount returns the provider support a row
// advertises: the distinct providers of a group, one per plain row.
func searchItemProviderCount(row SelectionItem) int {
	if len(row.Members) > 1 {
		return distinctProviders(row.Members)
	}

	return 1
}
