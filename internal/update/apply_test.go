package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newBinaryArchive builds an archive of the requested format from
// an ordered member list and serves it; the return is the archive
// and its sha256 digest.
func newBinaryArchive(
	t *testing.T,
	members []struct{ name, body string },
	useZIP bool,
) (path, digest string) {
	t.Helper()

	output := filepath.Join(t.TempDir(), "archive.tar.gz")

	if useZIP {
		output = filepath.Join(t.TempDir(), "archive.zip")

		file, err := os.Create(output)
		if err != nil {
			t.Fatal(err)
		}

		writer := zip.NewWriter(file)

		for _, member := range members {
			entry, err := writer.Create(member.name)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := fmt.Fprint(entry, member.body); err != nil {
				t.Fatal(err)
			}
		}

		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		file, err := os.Create(output)
		if err != nil {
			t.Fatal(err)
		}

		compressor := gzip.NewWriter(file)

		writer := tar.NewWriter(compressor)

		for _, member := range members {
			if err := writer.WriteHeader(&tar.Header{
				Name: member.name,
				Mode: 0o755,
				Size: int64(len(member.body)),
			}); err != nil {
				t.Fatal(err)
			}

			if _, err := fmt.Fprint(writer, member.body); err != nil {
				t.Fatal(err)
			}
		}

		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		if err := compressor.Close(); err != nil {
			t.Fatal(err)
		}

		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}

	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}

	return output, fmt.Sprintf(
		"sha256:%x",
		sha256.Sum256(body),
	)
}

func stubExecutable(t *testing.T, dir string) string {
	t.Helper()

	original := executablePath

	target := filepath.Join(dir, "goweeb")

	executablePath = func() (string, error) {
		return target, nil
	}

	t.Cleanup(func() { executablePath = original })

	return target
}

// serve hosts the archive as a release asset would; the return is
// the asset URL to hand an update.
func serve(t *testing.T, archivePath string) string {
	t.Helper()

	body, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/dist/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")

		_, _ = w.Write(body)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server.URL + "/dist/" + filepath.Base(archivePath)
}

func seedCurrentBinary(t *testing.T, target string) {
	t.Helper()

	if err := os.WriteFile(
		target,
		[]byte("OLD BINARY"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
}

func applyRelease(t *testing.T, archivePath, digest string) error {
	t.Helper()

	url := serve(t, archivePath)

	release := &Release{
		Tag:      "v9.9.9",
		AssetURL: url,
		Digest:   digest,
	}

	return Apply(release, nil)
}

func TestApplyReplacesTheRunningBinary(t *testing.T) {
	dir := t.TempDir()
	target := stubExecutable(t, dir)
	seedCurrentBinary(t, target)

	archive, digest := newBinaryArchive(t, []struct{ name, body string }{
		{"goweeb", "NEW BINARY"},
	}, false)

	if err := applyRelease(t, archive, digest); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o755 {
		t.Errorf("replaced binary mode = %v", info.Mode().Perm())
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "NEW BINARY" {
		t.Errorf("target content = %q", content)
	}

	if _, err := os.Stat(target + ".new"); !os.IsNotExist(err) {
		t.Errorf("the staged binary survived: %v", err)
	}
}

func TestApplyServesAWindowsArchive(t *testing.T) {
	dir := t.TempDir()
	target := stubExecutable(t, dir)
	seedCurrentBinary(t, target)

	archive, digest := newBinaryArchive(t, []struct{ name, body string }{
		{"goweeb", "NEW BINARY"},
	}, true)

	if err := applyRelease(t, archive, digest); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	content, _ := os.ReadFile(target)

	if string(content) != "NEW BINARY" {
		t.Errorf("target content = %q", content)
	}
}

func TestApplyRefusesAMismatchedDigest(t *testing.T) {
	dir := t.TempDir()
	target := stubExecutable(t, dir)
	seedCurrentBinary(t, target)

	archive, _ := newBinaryArchive(t, []struct{ name, body string }{
		{"goweeb", "NEW BINARY"},
	}, false)

	wrong := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("unrelated")))

	err := applyRelease(t, archive, wrong)
	if err == nil {
		t.Fatal("expected the mismatched digest to refuse the install")
	}

	if !strings.Contains(err.Error(), "does not match the published digest") {
		t.Errorf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(target)

	if string(content) != "OLD BINARY" {
		t.Errorf("the running binary was touched: %q", content)
	}
}

func TestApplySkipsTraversalMembers(t *testing.T) {
	dir := t.TempDir()
	target := stubExecutable(t, dir)
	seedCurrentBinary(t, target)

	archive, digest := newBinaryArchive(t, []struct{ name, body string }{
		{"../escaped", "EVIL"},
		{"goweeb", "NEW BINARY"},
	}, false)

	// The evil member is skipped, the real binary replaced.
	if err := applyRelease(t, archive, digest); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "escaped")); !os.IsNotExist(err) {
		t.Errorf("the traversal member escaped: %v", err)
	}

	content, _ := os.ReadFile(target)

	if string(content) != "NEW BINARY" {
		t.Errorf("target content = %q", content)
	}
}

func TestCleanupRestoresTheShiftedBinary(t *testing.T) {
	dir := t.TempDir()
	target := stubExecutable(t, dir)

	// A Windows dance stopped between its two renames.
	if err := os.WriteFile(target+".old", []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Cleanup(nil); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Errorf("the shifted binary was not restored: %v", err)
	}

	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Errorf("the old copy survived the restore: %v", err)
	}
}

func TestCleanupRemovesTheCompletedOldCopy(t *testing.T) {
	dir := t.TempDir()
	target := stubExecutable(t, dir)
	seedCurrentBinary(t, target)

	if err := os.WriteFile(target+".old", []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Cleanup(nil); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}

	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Errorf("the old copy survived: %v", err)
	}

	content, _ := os.ReadFile(target)

	if string(content) != "OLD BINARY" {
		t.Errorf("the current binary changed: %q", content)
	}
}
