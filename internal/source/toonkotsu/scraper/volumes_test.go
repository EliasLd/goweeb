package toonkotsuscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// volumeListing is one volume-tagged listing: two volumes
// ("07"/"7" canonicalize together, "08" alone), an untagged chapter,
// and an unnumbered extra the tag can never rescue.
func volumeListingFixture() string {
	return `{"series":{"title":"Jujutsu Kaisen"},"chapters":[` +
		`{"id":12,"n":2,"label":"Chapter 2","vol":"7","alts":[]},` +
		`{"id":10,"n":1,"label":"Chapter 1","vol":"07","alts":[{"id":11}]},` +
		`{"id":13,"n":3,"label":"Chapter 3","vol":"08","alts":[]},` +
		`{"id":14,"n":4,"label":"Omake","vol":null,"alts":[]},` +
		`{"id":15,"n":5,"label":"Special","vol":"","alts":[]},` +
		`{"id":16,"n":null,"vol":"8","label":"Epilogue","alts":[]}]}`
}

func TestListVolumesGroupsByVolumeTags(t *testing.T) {
	server := newSeriesServer(t, "bm0ssx", volumeListingFixture())

	title, volumes, err := ListVolumes(
		server.URL,
		"bm0ssx",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	if title != "Jujutsu Kaisen" {
		t.Errorf("unexpected title: %q", title)
	}

	if len(volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(volumes))
	}

	// Volumes ascending in their canonical tag.
	if volumes[0].Number != "7" {
		t.Errorf("volume number = %q, want 7", volumes[0].Number)
	}

	if volumes[1].Number != "8" {
		t.Errorf("volume number = %q, want 8", volumes[1].Number)
	}

	first := volumes[0].Chapters

	if len(first) != 2 {
		t.Fatalf("expected 2 chapters in volume 7, got %d", len(first))
	}

	if first[0].Number.String() != "1" || first[1].Number.String() != "2" {
		t.Errorf(
			"expected reading order 1, 2, got %s, %s",
			first[0].Number.String(),
			first[1].Number.String(),
		)
	}

	// The merged uploads of chapter 1 travel as one candidate URL.
	if first[0].URL != server.URL+"/manga/chapter/10,11" {
		t.Errorf("unexpected chapter URL: %q", first[0].URL)
	}

	if len(volumes[1].Chapters) != 1 ||
		volumes[1].Chapters[0].Number.String() != "3" {
		t.Errorf("unexpected volume 8 contents: %+v", volumes[1].Chapters)
	}
}

func TestListVolumesRefusesSeveralVolumeTags(t *testing.T) {
	server := newSeriesServer(t, "glitched", `{"series":{"title":"X"},"chapters":[`+
		`{"id":20,"n":7,"label":"A","vol":"1","alts":[]},`+
		`{"id":21,"n":7,"label":"B","vol":"2","alts":[]},`+
		`{"id":22,"n":8,"label":"C","vol":"1","alts":[]}]}`)

	title, volumes, err := ListVolumes(
		server.URL,
		"glitched",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	// Both bundles the ambiguous chapter touches are refused:
	// a mis-bundled volume is worse than no volume mode.
	if title != "X" {
		t.Errorf("unexpected title: %q", title)
	}

	if len(volumes) != 0 {
		t.Errorf("expected the conflicting volumes to be refused, got %+v", volumes)
	}
}

func TestListVolumesEmptyListing(t *testing.T) {
	server := newSeriesServer(t, "empty", `{"series":{"title":"X"},"chapters":[]}`)

	_, volumes, err := ListVolumes(
		server.URL,
		"empty",
		"",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	if len(volumes) != 0 {
		t.Errorf("expected no volumes, got %d", len(volumes))
	}
}

func TestListVolumesSelectsTheEditionListing(t *testing.T) {
	var requested []string

	mux := http.NewServeMux()

	mux.HandleFunc(seriesPath+"main", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		requested = append(requested, "main")

		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, `{"series":{"title":"Main"},"chapters":[{"id":1,"n":1,"vol":"1"}]}`)
	})
	mux.HandleFunc(seriesPath+"color1", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		requested = append(requested, "color1")

		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, `{"series":{"title":"Colored"},"chapters":[{"id":2,"n":1,"vol":"1"}]}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	title, volumes, err := ListVolumes(
		server.URL,
		"main",
		"color1",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	if len(requested) != 1 || requested[0] != "color1" {
		t.Errorf("expected only the edition listing, got %v", requested)
	}

	if title != "Colored" {
		t.Errorf("expected the edition title, got %q", title)
	}

	if len(volumes) != 1 || volumes[0].Number != "1" {
		t.Errorf("expected the edition volume, got %+v", volumes)
	}
}

func TestListVolumesMalformedPayload(t *testing.T) {
	server := newSeriesServer(t, "bad", `{"series":{"title":"X"},"chapters":[`)

	_, _, err := ListVolumes(server.URL, "bad", "", testDiscardLogger())
	if err == nil {
		t.Fatal("expected a decode error")
	}

	if !strings.Contains(err.Error(), "failed to decode toonkotsu series bad") {
		t.Errorf("expected the decode error, got %v", err)
	}
}
