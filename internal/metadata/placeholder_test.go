package metadata

import (
	"bytes"
	"image/png"
	"testing"
)

func TestCoverPlaceholderPNGIsValidAndDeterministic(t *testing.T) {
	first, err := CoverPlaceholderPNG("One Piece")
	if err != nil {
		t.Fatalf("CoverPlaceholderPNG() error: %v", err)
	}

	if len(first) == 0 ||
		string(first[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("expected a PNG document")
	}

	decoded, err := png.Decode(bytes.NewReader(first))
	if err != nil {
		t.Fatalf("png.Decode() error: %v", err)
	}

	bounds := decoded.Bounds()

	if bounds.Dx() != placeholderWidth ||
		bounds.Dy() != placeholderHeight {
		t.Errorf(
			"bounds = %dx%d, want %dx%d",
			bounds.Dx(),
			bounds.Dy(),
			placeholderWidth,
			placeholderHeight,
		)
	}

	// The exact same title must render the exact same placeholder.
	second, err := CoverPlaceholderPNG("One Piece")
	if err != nil {
		t.Fatalf("second CoverPlaceholderPNG() error: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Error("expected identical bytes for identical titles")
	}
}

func TestCoverPlaceholderPNGDiffersByTitle(t *testing.T) {
	first, err := CoverPlaceholderPNG("Berserk")
	if err != nil {
		t.Fatalf("CoverPlaceholderPNG() error: %v", err)
	}

	other, err := CoverPlaceholderPNG("One Piece")
	if err != nil {
		t.Fatalf("second CoverPlaceholderPNG() error: %v", err)
	}

	if bytes.Equal(first, other) {
		t.Error("expected different titles to render differently")
	}
}

func TestInitialsFor(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{title: "One Piece", want: "OP"},
		{title: "Berserk", want: "B"},
		{title: "  One Piece  ", want: "OP"},
		{title: "オトナリ Complex", want: "オC"},
		{title: "3-Gatsu no Lion", want: "3G"},
		{title: "!!!", want: "#"},
		{title: "", want: "#"},
	}

	for _, test := range tests {
		if got := initialsFor(test.title); got != test.want {
			t.Errorf(
				"initialsFor(%q) = %q, want %q",
				test.title,
				got,
				test.want,
			)
		}
	}
}
