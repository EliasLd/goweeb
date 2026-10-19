package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestRankSearchResult(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		title    string
		expected int
	}{
		{
			name:     "exact match",
			query:    "bleach",
			title:    "Bleach",
			expected: 0,
		},
		{
			name:     "exact match ignores case and spaces",
			query:    "  blue lock  ",
			title:    "Blue Lock",
			expected: 0,
		},
		{
			name:     "word-boundary prefix",
			query:    "one",
			title:    "One Piece",
			expected: 1,
		},
		{
			name:     "prefix ending on punctuation",
			query:    "bleach",
			title:    "Bleach: Thousand-Year Blood War",
			expected: 1,
		},
		{
			name:     "prefix continuing into a word",
			query:    "one",
			title:    "Onepunch Man",
			expected: 3,
		},
		{
			name:     "whole word containment",
			query:    "piece",
			title:    "One Piece",
			expected: 2,
		},
		{
			name:     "substring inside a word",
			query:    "lock",
			title:    "Lockdown",
			expected: 3,
		},
		{
			name:     "no match",
			query:    "frieren",
			title:    "Naruto",
			expected: 4,
		},
		{
			name:     "empty query",
			query:    " ",
			title:    "Bleach",
			expected: 4,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := rankSearchResult(
				testCase.query,
				testCase.title,
			)

			if got != testCase.expected {
				t.Errorf(
					"rank(%q, %q) = %d, want %d",
					testCase.query,
					testCase.title,
					got,
					testCase.expected,
				)
			}
		})
	}
}

func TestSortSearchItems(t *testing.T) {
	items := []list.Item{
		SelectionItem{Label: "Onepunch Man", Value: "3"},
		SelectionItem{Label: "Blue Lock", Value: "0"},
		SelectionItem{Label: "One Piece: Ace", Value: "4"},
		SelectionItem{Label: "One Piece", Value: "1"},
		SelectionItem{Label: "Naruto", Value: "5"},
		SelectionItem{Label: "One Piece Film", Value: "2"},
	}

	sortSearchItems(items, "one piece")

	expected := []string{
		"One Piece",
		"One Piece: Ace",
		"One Piece Film",
		"Onepunch Man",
		"Blue Lock",
		"Naruto",
	}

	// "One Piece: Ace" and "One Piece Film" tie in the prefix bucket
	// and keep their arrival order; the unmatched titles follow.

	if len(items) != len(expected) {
		t.Fatalf("items = %d, want %d", len(items), len(expected))
	}

	for i, want := range expected {
		item, _ := items[i].(SelectionItem)

		if item.Label != want {
			t.Errorf(
				"position %d = %q, want %q",
				i,
				item.Label,
				want,
			)
		}
	}
}
