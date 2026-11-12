package mangaballscraper

import (
	"strings"
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// Volume fixtures reuse the chapter listing server: the site's
// volume tags travel on the same rows (probed 2026-11-08: berserk en
// groups 1,571 rows into 43 volumes that match MangaDex; untagged
// rows are 0).
func TestListVolumesGroupsTaggedRows(t *testing.T) {
	rows := `[
		{"id":"id-5","number":5,"volume":2,"name":"Chapter 5","lang":"en","site":"mangadex"},
		{"id":"id-2","number":2,"volume":0,"name":"Chapter 2","lang":"en","site":"mangadex"},
		{"id":"id-1","number":1,"volume":1,"name":"Chapter 1","lang":"en","site":"comick"},
		{"id":"id-1-b","number":1,"volume":1,"name":"Chapter 1 (other site)","lang":"en","site":"mangadex"},
		{"id":"id-15","number":1.5,"volume":1,"name":"","lang":"en","site":"comick"},
		{"id":"id-3","number":3,"volume":1,"name":"Chapter 3","lang":"en","site":"comick"}
	]`

	server := newChaptersServer(t, 6, rows, "Berserk")

	work, volumes, err := ListVolumes(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("volumes: %v", err)
	}

	if work.Title != "Berserk" {
		t.Errorf("Title = %q, want Berserk", work.Title)
	}

	if work.Kind != sourcetypes.ItemVolume {
		t.Errorf("Kind = %v, want ItemVolume", work.Kind)
	}

	if len(volumes) != 2 {
		t.Fatalf("volumes = %d, want 2 (untagged rows stay out)", len(volumes))
	}

	if volumes[0].Number != "1" || volumes[1].Number != "2" {
		t.Errorf("volume numbers = %q, %q", volumes[0].Number, volumes[1].Number)
	}

	volume1 := volumes[0]

	if len(volume1.Entries) != 3 {
		t.Fatalf("volume 1 chapters = %d, want 3", len(volume1.Entries))
	}

	// Copies of the same chapter inside one volume join their ids.
	if want := server.URL + "/chapter-detail/id-1,id-1-b"; volume1.Entries[0].URL != want {
		t.Errorf("chapter 1 url = %q, want %q", volume1.Entries[0].URL, want)
	}

	if volume1.Entries[1].Number.String() != "1.5" {
		t.Errorf("volume 1 second chapter = %s, want 1.5", volume1.Entries[1].Number.String())
	}
}

func TestListVolumesRefusesIncompleteVolume(t *testing.T) {
	// Volume 2 carries a row without an id: bundling it would ship a
	// wrong archive, so the volume is refused while volume 1 stays.
	rows := `[
		{"id":"id-1","number":1,"volume":1,"name":"Chapter 1","lang":"en","site":"comick"},
		{"id":"id-3","number":3,"volume":2,"name":"Chapter 3","lang":"en","site":"mangadex"},
		{"id":"","number":4,"volume":2,"name":"Chapter 4","lang":"en","site":"mangadex"}
	]`

	server := newChaptersServer(t, 3, rows, "Berserk")

	_, volumes, err := ListVolumes(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("volumes: %v", err)
	}

	if len(volumes) != 1 || volumes[0].Number != "1" {
		t.Errorf("volumes = %+v, want only volume 1", volumes)
	}
}

func TestListVolumesRefusesConflictingVolumes(t *testing.T) {
	// Chapter 5 tagged in volumes 1 and 2: both volumes become
	// ambiguous, so both are refused.
	rows := `[
		{"id":"id-1","number":1,"volume":1,"name":"Chapter 1","lang":"en","site":"comick"},
		{"id":"id-5a","number":5,"volume":1,"name":"Chapter 5","lang":"en","site":"comick"},
		{"id":"id-2","number":2,"volume":2,"name":"Chapter 2","lang":"en","site":"mangadex"},
		{"id":"id-5b","number":5,"volume":2,"name":"Chapter 5","lang":"en","site":"mangadex"}
	]`

	server := newChaptersServer(t, 4, rows, "Berserk")

	_, volumes, err := ListVolumes(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)
	if err != nil {
		t.Fatalf("volumes: %v", err)
	}

	if len(volumes) != 0 {
		t.Errorf("volumes = %+v, want none (conflicting tags)", volumes)
	}
}

func TestListVolumesIncompleteListing(t *testing.T) {
	// Same hard error as the chapter flow: a truncated walk must not
	// present partial volumes.
	rows := `[{"id":"id-1","number":1,"volume":1,"name":"Chapter 1","lang":"en","site":"comick"}]`

	server := newChaptersServer(t, 6, rows, "Berserk")

	_, _, err := ListVolumes(
		server.URL,
		"6851544a702284f8341782f2",
		"en",
		nil,
	)

	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Errorf("error = %v, want the incomplete-listing failure", err)
	}
}
