package app

import (
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"testing"
)

func entries(numbers ...string) []sourcetypes.Entry {
	out := make([]sourcetypes.Entry, 0, len(numbers))
	for _, s := range numbers {
		n, err := common.ParseChapterNumber(s)
		if err != nil {
			panic(err)
		}
		out = append(out, sourcetypes.Entry{Number: n})
	}
	return out
}
func labels(list []sourcetypes.Entry) []string {
	out := make([]string, 0, len(list))
	for _, e := range list {
		out = append(out, e.Number.String())
	}
	return out
}
func TestAvailableRangesWithDecimalBreaks(t *testing.T) {
	input := entries("6.5", "3", "2", "1", "5", "4", "6", "7", "8", "8", "0")
	got := FormatAvailableRanges(input)
	if got != "0-6, 6.5, 7-8" {
		t.Fatalf("got %q", got)
	}
}
func TestParseAndFilter(t *testing.T) {
	input := entries("8", "7", "6.5", "6", "5", "0.5", "0", "0", "1")
	tests := []struct {
		expression string
		want       string
	}{
		{"0", "0"}, {"0-", "0,0.5,1,5,6,6.5,7,8"},
		{"6.5", "6.5"}, {"6-7", "6,6.5,7"},
		{"0-1,6.5", "0,0.5,1,6.5"}, {"-3", "6.5,7,8"},
		{"all", "0,0.5,1,5,6,6.5,7,8"},
	}
	for _, tc := range tests {
		selection, err := ParseRangeExpression(tc.expression)
		if err != nil {
			t.Errorf("%s: %v", tc.expression, err)
			continue
		}
		got := stringsJoin(labels(FilterEntriesBySelection(input, selection)))
		if got != tc.want {
			t.Errorf("%s got %q want %q", tc.expression, got, tc.want)
		}
	}
	for _, invalid := range []string{"-0", "-1.5", "1.2.3", "2-1", "1-2-3", "-2,1-3", "1,", "1-abc"} {
		if _, err := ParseRangeExpression(invalid); err == nil {
			t.Errorf("expected invalid: %q", invalid)
		}
	}
}
func stringsJoin(items []string) string {
	if len(items) == 0 {
		return ""
	}
	s := items[0]
	for _, item := range items[1:] {
		s += "," + item
	}
	return s
}
