package weebcentralscraper

import "testing"

func TestParseChapterLabel(t *testing.T) {
	tests := []struct {
		label string
		want  string
		valid bool
	}{
		{"Chapter 271.5", "271.5", true},
		{"Episode 139.5", "139.5", true},
		{"Episode 9", "9", true},
		{"#100", "100", true},
		{"# 12.5", "12.5", true},
		{"Part 12 - Bonus", "12", true},
		{"Chapter 0", "0", true},
		{"12.5", "12.5", true},
		{"Chapter 12: Bonus 2", "12", true},
		{"Last Read", "", false},
		{"Chapter ???", "", false},
		{"Special", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			number, _, ok := parseChapterLabel(tt.label)

			if ok != tt.valid {
				t.Fatalf(
					"valid = %v, want %v",
					ok,
					tt.valid,
				)
			}

			if !tt.valid {
				return
			}

			if got := number.String(); got != tt.want {
				t.Errorf(
					"number = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
