package app

import (
	"path/filepath"
	"testing"

	"github.com/EliasLd/goweeb/internal/source/common"
)

func TestParseLayout(t *testing.T) {
	for raw, want := range map[string]Layout{
		"series":   LayoutSeries,
		"flat":     LayoutFlat,
		" Series ": LayoutSeries,
		"FLAT":     LayoutFlat,
	} {
		layout, err := ParseLayout(raw)
		if err != nil {
			t.Fatalf(
				"ParseLayout(%q) error: %v",
				raw,
				err,
			)
		}

		if layout != want {
			t.Errorf(
				"ParseLayout(%q) = %q, want %q",
				raw,
				layout,
				want,
			)
		}
	}

	for raw := range map[string]struct{}{
		"":          {},
		"scattered": {},
	} {
		if _, err := ParseLayout(raw); err == nil {
			t.Errorf(
				"ParseLayout(%q) = nil error, want an error",
				raw,
			)
		}
	}
}

func TestSanitizeSegment(t *testing.T) {
	for input, want := range map[string]string{
		"One Piece":                     "One Piece",
		"Fate/Zero":                     "Fate Zero",
		"Attack on Titan: Final Season": "Attack on Titan Final Season",
		`a:b*c?d"e<f>g|h\i`:             "a b c d e f g h i",
		"  spaced \t title  ":           "spaced title",
		"One Piece - ":                  "One Piece",
		"Vol 1.":                        "Vol 1",
		"???":                           "download",
	} {
		if got := sanitizeSegment(input); got != want {
			t.Errorf(
				"sanitizeSegment(%q) = %q, want %q",
				input,
				got,
				want,
			)
		}
	}
}

// chapter builds a canonical entry number, failing the test on an
// invalid literal.
func chapter(t *testing.T, number string) common.ChapterNumber {
	t.Helper()

	parsed, err := common.ParseChapterNumber(number)
	if err != nil {
		t.Fatalf("invalid chapter number %q: %v", number, err)
	}

	return parsed
}

func TestLayoutEntryPath(t *testing.T) {
	tests := []struct {
		name   string
		layout Layout
		series string
		number string
		digits int
		ext    string
		want   string
	}{{
		name:   "series names files and zeroes pads the number",
		layout: LayoutSeries,
		series: "One Piece",
		number: "10",
		digits: 3,
		ext:    ".cbz",
		want:   filepath.Join("scan", "One Piece", "One Piece - c010.cbz"),
	}, {
		name:   "series keeps fractional numbers readable",
		layout: LayoutSeries,
		series: "One Piece",
		number: "6.5",
		digits: 3,
		ext:    ".pdf",
		want:   filepath.Join("scan", "One Piece", "One Piece - c006.5.pdf"),
	}, {
		name:   "series sanitizes the folder and filename",
		layout: LayoutSeries,
		series: "Fate/Zero",
		number: "1",
		digits: 3,
		ext:    ".cbz",
		want:   filepath.Join("scan", "Fate Zero", "Fate Zero - c001.cbz"),
	}, {
		name:   "series pads to the widest chapter width",
		layout: LayoutSeries,
		series: "One Piece",
		number: "12",
		digits: 4,
		ext:    ".cbz",
		want:   filepath.Join("scan", "One Piece", "One Piece - c0012.cbz"),
	}, {
		name:   "flat keeps the historical naming verbatim",
		layout: LayoutFlat,
		series: "One Piece",
		number: "10",
		digits: 3,
		ext:    ".cbz",
		want:   filepath.Join("scan", "One Piece_10.cbz"),
	}, {
		name:   "flat ignores sanitization",
		layout: LayoutFlat,
		series: "Fate/Zero",
		number: "1",
		digits: 3,
		ext:    ".cbz",
		want:   filepath.Join("scan", "Fate", "Zero_1.cbz"),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutEntryPath(
				tt.layout,
				"scan",
				tt.series,
				chapter(t, tt.number),
				tt.digits,
				tt.ext,
			)

			if got != tt.want {
				t.Errorf(
					"layoutEntryPath() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestLayoutEntryImageDir(t *testing.T) {
	tests := []struct {
		name   string
		layout Layout
		series string
		prefix string
		number string
		digits int
		want   string
	}{{
		name:   "series reuses the archive name for image folders",
		layout: LayoutSeries,
		series: "One Piece",
		prefix: "Chapter",
		number: "10",
		digits: 3,
		want:   filepath.Join("scan", "One Piece", "One Piece - c010"),
	}, {
		name:   "series names volumes like chapters",
		layout: LayoutSeries,
		series: "One Piece",
		prefix: "Volume",
		number: "12",
		digits: 3,
		want:   filepath.Join("scan", "One Piece", "One Piece - c012"),
	}, {
		name:   "flat keeps the historical prefix folders",
		layout: LayoutFlat,
		series: "One Piece",
		prefix: "Chapter",
		number: "10",
		digits: 3,
		want:   filepath.Join("scan", "Chapter 010"),
	}, {
		name:   "flat keeps volume folders",
		layout: LayoutFlat,
		series: "One Piece",
		prefix: "Volume",
		number: "12",
		digits: 3,
		want:   filepath.Join("scan", "Volume 012"),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutEntryImageDir(
				tt.layout,
				"scan",
				tt.series,
				tt.prefix,
				chapter(t, tt.number),
				tt.digits,
			)

			if got != tt.want {
				t.Errorf(
					"layoutEntryImageDir() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
