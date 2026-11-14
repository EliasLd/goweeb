package toonkotsuscraper

import (
	"strings"
	"testing"
)

func TestWorkCodeFromURL(t *testing.T) {
	cases := []struct {
		name    string
		workURL string
		code    string
		wantErr bool
	}{
		{
			name:    "site-canonical URL",
			workURL: "https://toonkotsu.com/title/bm0ssx-jujutsu-kaisen",
			code:    "bm0ssx",
		},
		{
			name:    "slug with hyphens keeps the first segment",
			workURL: "https://toonkotsu.com/title/abc123-one-piece-color-edition",
			code:    "abc123",
		},
		{
			name:    "code only",
			workURL: "https://toonkotsu.com/title/bm0ssx",
			code:    "bm0ssx",
		},
		{
			name:    "query and fragment are tolerated",
			workURL: "https://toonkotsu.com/title/abc123-work?x=1#frag",
			code:    "abc123",
		},
		{
			name:    "wrong path",
			workURL: "https://toonkotsu.com/manga/abc123-work",
			wantErr: true,
		},
		{
			name:    "empty code",
			workURL: "https://toonkotsu.com/title/-work-slug",
			wantErr: true,
		},
		{
			name:    "trailing slash",
			workURL: "https://toonkotsu.com/title/abc123-work/",
			code:    "abc123",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			code, err := WorkCodeFromURL(testCase.workURL)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf(
						"WorkCodeFromURL(%q) = %q, want error",
						testCase.workURL,
						code,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"WorkCodeFromURL(%q) error: %v",
					testCase.workURL,
					err,
				)
			}

			if code != testCase.code {
				t.Errorf(
					"WorkCodeFromURL(%q) = %q, want %q",
					testCase.workURL,
					code,
					testCase.code,
				)
			}
		})
	}
}

func TestGetSeriesDecodesNullAndFractionalNumbers(t *testing.T) {
	server := newSeriesServer(t, "onepiece", `{"series":{"title":"One Piece"},"chapters":[`+
		`{"id":4581681,"n":1,"label":"Romance Dawn","alts":[{"id":4581682}]},`+
		`{"id":4581683,"n":1043.1,"label":"Captain","alts":[]},`+
		`{"id":4581684,"n":null,"label":"Volume 112","alts":[]}]}`)

	series, err := GetSeries(server.URL, "onepiece", testDiscardLogger())
	if err != nil {
		t.Fatalf("GetSeries() error: %v", err)
	}

	if series.Series.Title != "One Piece" {
		t.Errorf("unexpected series title: %q", series.Series.Title)
	}

	if len(series.Chapters) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(series.Chapters))
	}

	// json.Number keeps the exact literal, null decodes as nil.
	if series.Chapters[1].N == nil || series.Chapters[1].N.String() != "1043.1" {
		t.Errorf(
			"row n = %v, want the exact decimal",
			series.Chapters[1].N,
		)
	}

	if series.Chapters[2].N != nil {
		t.Errorf("row n = %v, want nil", series.Chapters[2].N)
	}
}

func TestGetSeriesDecodesIdsOfBothShapes(t *testing.T) {
	// Rows and alternates deliver numeric ids today; the decoder
	// tolerates string ids so a payload change cannot drop rows.
	server := newSeriesServer(t, "mixed", `{"series":{"title":"M"},"chapters":[`+
		`{"id":4581691,"n":1,"label":"One","alts":[{"id":4581692},{"id":"alt-text"}]},`+
		`{"id":"4581693","n":2,"label":"Two","alts":[]}]}`)

	series, err := GetSeries(server.URL, "mixed", testDiscardLogger())
	if err != nil {
		t.Fatalf("GetSeries() error: %v", err)
	}

	if string(series.Chapters[0].ID) != "4581691" {
		t.Errorf("numeric id = %q", series.Chapters[0].ID)
	}

	if len(series.Chapters[0].Alts) != 2 ||
		string(series.Chapters[0].Alts[0].ID) != "4581692" ||
		string(series.Chapters[0].Alts[1].ID) != "alt-text" {
		t.Errorf("unexpected alts: %+v", series.Chapters[0].Alts)
	}

	if string(series.Chapters[1].ID) != "4581693" {
		t.Errorf("string id = %q", series.Chapters[1].ID)
	}
}

func TestGetSeriesMalformedPayload(t *testing.T) {
	server := newSeriesServer(t, "bad", `{"series":{"title":"X"},"chapters":[`)

	_, err := GetSeries(server.URL, "bad", testDiscardLogger())
	if err == nil {
		t.Fatal("expected a decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode toonkotsu series bad") {
		t.Errorf("expected the decode error, got %v", err)
	}
}
