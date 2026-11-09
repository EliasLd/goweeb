package ledger

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func TestSetMetadataPersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()

	led := openLedgerIn(t, dir)

	changed := led.SetMetadata(sourcetypes.Metadata{
		Author:   "Miura Kentarou",
		CoverURL: "https://uploads.mangadex.org/covers/abc/def.jpg",
		Year:     1989,
	})
	if !changed {
		t.Fatal("expected the first metadata to change the manifest")
	}

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reloaded, loadErr := New(led.path, testIdentity())
	if loadErr != nil {
		t.Fatalf("reload New() error: %v", loadErr)
	}

	stored := reloaded.Metadata()
	if stored == nil {
		t.Fatal("expected the metadata to survive the reload")
	}

	if stored.Author != "Miura Kentarou" || stored.Year != 1989 {
		t.Errorf("stored metadata = %+v, want the recorded one", *stored)
	}
}

func TestSetMetadataEmptyStaysAbsent(t *testing.T) {
	led := openTestLedger(t)

	if led.SetMetadata(sourcetypes.Metadata{}) {
		t.Error("expected empty metadata to change nothing")
	}

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(led.path)
	if err != nil {
		t.Fatalf("failed to read the manifest: %v", err)
	}

	if bytes.Contains(data, []byte(`"metadata"`)) {
		t.Error("expected no metadata key for empty metadata")
	}

	if bytes.Contains(data, []byte(`"coverFile"`)) {
		t.Error("expected no coverFile key without a cover")
	}
}

func TestSetMetadataTwiceIsAdditive(t *testing.T) {
	dir := t.TempDir()
	led := openLedgerIn(t, dir)

	if !led.SetMetadata(sourcetypes.Metadata{Author: "A"}) {
		t.Fatal("expected the first set to change")
	}

	if led.SetMetadata(sourcetypes.Metadata{Author: "A"}) {
		t.Error("expected an unchanged set to be a no-op")
	}

	if !led.SetMetadata(sourcetypes.Metadata{Author: "A", Year: 1990}) {
		t.Fatal("expected a richer set to change")
	}
}

// An additive schema must keep loading documents written before the
// fields existed.
func TestLoadOldManifestWithoutMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "One Piece.goweeb.json")

	if err := os.WriteFile(path, []byte(
		`{"version":1,"series":"One Piece","provider":"mangadex","chapters":[]}`,
	), 0o644); err != nil {
		t.Fatalf("failed to write the manifest: %v", err)
	}

	led, loadErr := New(path, testIdentity())
	if loadErr != nil {
		t.Fatalf("New() error: %v", loadErr)
	}

	if led.Metadata() != nil || led.manifest.CoverFile != "" {
		t.Error("expected no metadata block in an old document")
	}
}

func TestSetCoverFileClearsAndReloads(t *testing.T) {
	dir := t.TempDir()
	led := openLedgerIn(t, dir)

	coverName := "One Piece.cover.jpg"

	if !led.SetCoverFile(coverName) {
		t.Fatal("expected the cover record to change")
	}

	// The recorded name is normalized to a basename: the cover lives
	// next to the manifest.
	if led.SetCoverFile(filepath.Join(dir, coverName)) {
		t.Error("expected the basename of the same cover to be a no-op")
	}

	if err := led.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	reloaded, loadErr := New(led.path, testIdentity())
	if loadErr != nil {
		t.Fatalf("reload New() error: %v", loadErr)
	}

	if reloaded.manifest.CoverFile != coverName {
		t.Errorf(
			"coverFile = %q, want %q",
			reloaded.manifest.CoverFile,
			coverName,
		)
	}

	// The accessor mirrors the recorded value.
	if got := reloaded.CoverFile(); got != coverName {
		t.Errorf("CoverFile() = %q, want %q", got, coverName)
	}

	if !reloaded.SetCoverFile("") {
		t.Error("expected clearing to change the manifest")
	}
}
