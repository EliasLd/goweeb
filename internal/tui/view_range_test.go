package tui

import "testing"

func TestWrapAvailableRanges(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "all ranges fit",
			input: "1-6, 6.5, 7-10",
			width: 76,
			want:  "1-6, 6.5, 7-10",
		},
		{
			name:  "wrap at comma",
			input: "1-6, 6.5, 7-10",
			width: 10,
			want:  "1-6, 6.5\n7-10",
		},
		{
			name:  "preserve individual range",
			input: "100-200, 201-300",
			width: 5,
			want:  "100-200\n201-300",
		},
		{
			name:  "ignore empty items",
			input: "1-2, , 3",
			width: 5,
			want:  "1-2\n3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapAvailableRanges(
				tt.input,
				tt.width,
			)

			if got != tt.want {
				t.Errorf(
					"wrapAvailableRanges() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
