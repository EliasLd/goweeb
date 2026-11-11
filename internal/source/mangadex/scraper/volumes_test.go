package mangadexscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// Fixture shapes come from real aggregate payloads (probed
// 2026-11-08 and 2026-11-11: berserk en carries 43 tagged volumes
// plus a "none" bucket; the chapters container of each volume is
// keyed by chapter number and each chapter carries isUnavailable and
// same-number others with a count), trimmed to the fields the code
// reads. The volumes container mixes volume objects with the bare
// "count" total.
const aggregateFixture = `{
	"result": "ok",
	"volumes": {
		"count": 2,
		"none": {"volume": "none", "count": 2, "chapters": {
			"10": {"chapter": "10", "id": "c10", "count": 1},
			"9": {"chapter": "9", "id": "c9", "count": 1}
		}},
		"1": {"volume": "1", "count": 3, "chapters": {
			"1": {"chapter": "1", "id": "c1", "count": 2, "others": ["c1b"]},
			"1.5": {"chapter": "1.5", "id": "c15", "count": 1},
			"2": {"chapter": "2", "id": "c2", "count": 1}
		}},
		"2": {"volume": "2", "count": 2, "chapters": {
			"3": {"chapter": "3", "id": "c3", "count": 1},
			"4": {"chapter": "4", "id": "c4", "count": 1, "isUnavailable": true}
		}}
	}
}`

const aggregateDetailFixture = detailFixture

// newVolumesServer serves the detail endpoint the title resolution
// walks and the aggregate endpoint under one host.
func newVolumesServer(t *testing.T, aggregate string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/manga/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/aggregate") {
			_, _ = fmt.Fprint(w, aggregate)

			return
		}

		_, _ = fmt.Fprint(w, aggregateDetailFixture)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func listTestVolumes(t *testing.T, aggregate string) (sourcetypes.Work, []sourcetypes.Volume, error) {
	t.Helper()

	server := newVolumesServer(t, aggregate)

	return ListVolumes(
		server.URL+"/manga/801513ba",
		server.URL+"/manga/801513ba-a712-498c-8f57-cae55b38cc92",
		"en",
		tLogger(t),
	)
}

func TestListVolumes(t *testing.T) {
	work, volumes, err := listTestVolumes(t, aggregateFixture)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	if work.Title != "Berserk" {
		t.Errorf("Title = %q, want Berserk", work.Title)
	}

	if work.Kind != sourcetypes.ItemVolume {
		t.Errorf("Kind = %v, want ItemVolume", work.Kind)
	}

	// Volume 2 bundles an unavailable chapter and is refused; the
	// "none" bucket stays chapter-mode only; "count" is not a
	// volume. Volume 1 keeps its chapters deduplicated by number.
	if len(volumes) != 1 {
		t.Fatalf("len(volumes) = %d, want 1 volume: %+v", len(volumes), volumes)
	}

	volume := volumes[0]

	if volume.Number != "1" {
		t.Errorf("Number = %q, want \"1\"", volume.Number)
	}

	numbers := ""

	for _, entry := range volume.Entries {
		numbers += entry.Number.String() + " "
	}

	if numbers != "1 1.5 2 " {
		t.Errorf("chapters = %q, want \"1 1.5 2 \"", numbers)
	}

	first := volume.Entries[0]

	if !strings.HasSuffix(first.URL, "/chapter/c1") {
		t.Errorf("first entry URL = %q, want a /chapter/c1 suffix", first.URL)
	}

	if first.Label != "Chapter 1" {
		t.Errorf("first entry label = %q, want \"Chapter 1\"", first.Label)
	}
}

func TestListVolumesUnavailableVolume(t *testing.T) {
	_, volumes, err := listTestVolumes(t, aggregateFixture)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	for _, volume := range volumes {
		if volume.Number == "2" {
			t.Errorf("volume 2 bundles an unavailable chapter and should be refused: %+v", volume)
		}
	}
}

func TestListVolumesUnnumberedChapter(t *testing.T) {
	fixture := `{
		"result": "ok",
		"volumes": {
			"count": 1,
			"1": {"volume": "1", "count": 2, "chapters": {
				"1": {"chapter": "1", "id": "c1", "count": 1},
				"2": {"chapter": null, "id": "c2", "count": 1}
			}}
		}
	}`

	_, volumes, err := listTestVolumes(t, fixture)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	if len(volumes) != 0 {
		t.Errorf("volumes = %+v, want none (volume 1 has an unnumbered chapter)", volumes)
	}
}

func TestListVolumesNoneOnly(t *testing.T) {
	fixture := `{
		"result": "ok",
		"volumes": {
			"count": 0,
			"none": {"volume": "none", "count": 1, "chapters": {
				"1": {"chapter": "1", "id": "c1", "count": 1}
			}}
		}
	}`

	work, volumes, err := listTestVolumes(t, fixture)
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}

	if work.Title != "Berserk" {
		t.Errorf("Title = %q, want Berserk", work.Title)
	}

	if len(volumes) != 0 {
		t.Errorf("volumes = %+v, want none (untagged chapters only)", volumes)
	}
}

func TestListVolumesEmptyLanguage(t *testing.T) {
	server := newVolumesServer(t, aggregateFixture)

	_, _, err := ListVolumes(
		server.URL+"/manga/801513ba",
		server.URL+"/manga/801513ba-a712-498c-8f57-cae55b38cc92",
		"  ",
		tLogger(t),
	)

	if err == nil || !strings.Contains(err.Error(), "language is empty") {
		t.Errorf("ListVolumes() error = %v, want empty-language failure", err)
	}
}
