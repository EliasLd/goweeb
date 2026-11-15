package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// maxBinarySize caps an archive member so a corrupted or malicious
// release cannot expand without bound.
const maxBinarySize = 128 << 20

// executablePath is the path of the running binary, swappable in
// tests.
var executablePath = func() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}

	return resolved, nil
}

// Apply downloads the release archive, verifies its checksum against
// the digest the release API published, and replaces the running
// binary with the extracted one. The current process keeps running
// its own version; the replacement takes effect on the next start.
func Apply(release *Release, log *logger.Logger) error {
	target, err := executablePath()
	if err != nil {
		return fmt.Errorf(
			"failed to locate the running binary: %w",
			err,
		)
	}

	if err := recoverSwap(target); err != nil {
		return fmt.Errorf(
			"the leftover of an interrupted earlier update could not be repaired: %w",
			err,
		)
	}

	stage := target + ".new"

	_ = os.Remove(stage)

	if err := ensureWritable(filepath.Dir(target)); err != nil {
		return err
	}

	archive, err := downloadArchive(release, log)
	if err != nil {
		return err
	}

	defer os.Remove(archive)

	if err := verifyDigest(archive, release.Digest); err != nil {
		return err
	}

	if err := extractBinary(archive, release.AssetURL, stage); err != nil {
		return err
	}

	if err := swapIn(stage, target); err != nil {
		return err
	}

	if log != nil {
		log.Info(
			"Updated to %s: the new version starts on the next run.\n",
			release.Tag,
		)
	}

	return nil
}

// ensureWritable probes the target directory before anything is
// touched: a /usr/bin the user cannot write to must fail with a
// permission message, not a half-finished update.
func ensureWritable(dir string) error {
	probe, err := os.CreateTemp(dir, ".goweeb-probe-*")
	if err != nil {
		return fmt.Errorf(
			"the binary directory %s is not writable: %w",
			dir,
			err,
		)
	}

	dirProbe := probe.Name()
	_ = probe.Close()
	_ = os.Remove(dirProbe)

	return nil
}

// downloadArchive fetches the release archive into a temporary
// file and returns its path; the caller removes it.
func downloadArchive(
	release *Release,
	log *logger.Logger,
) (string, error) {
	client := &http.Client{
		Timeout: 90 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, release.AssetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	file, err := os.CreateTemp("", "goweeb-update-*"+archiveExtension(release.AssetURL))
	if err != nil {
		return "", fmt.Errorf("failed to create a temporary archive: %w", err)
	}

	dest := file.Name()
	file.Close()

	_ = os.Remove(dest)

	if err := httpx.DownloadToFile(client, req, dest, log); err != nil {
		return "", fmt.Errorf(
			"failed to download the update archive: %w",
			err,
		)
	}

	return dest, nil
}

// archiveExtension reads the archive format off the asset URL.
func archiveExtension(assetURL string) string {
	if strings.HasSuffix(assetURL, ".zip") {
		return ".zip"
	}

	return ".tar.gz"
}

// verifyDigest compares the downloaded archive against the digest
// published with the release.
func verifyDigest(archivePath, digest string) error {
	expected := strings.ToLower(strings.TrimPrefix(digest, "sha256:"))

	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf(
			"failed to open the downloaded archive: %w",
			err,
		)
	}

	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf(
			"failed to hash the downloaded archive: %w",
			err,
		)
	}

	if got := hex.EncodeToString(hash.Sum(nil)); got != expected {
		return fmt.Errorf(
			"the downloaded archive does not match the published digest (%s vs %s), refusing to install it",
			got,
			expected,
		)
	}

	return nil
}

// swapIn puts the staged binary where the running one lives. A
// running Windows executable refuses its own replacement, so the
// current one is renamed aside first, the replacement moved into its
// old path, and the old copy deleted on a best-effort basis (the
// running process still holds it until it exits).
func swapIn(stage, target string) error {
	if runtime.GOOS == "windows" {
		old := target + ".old"

		if err := os.Rename(target, old); err != nil {
			return fmt.Errorf(
				"failed to move the running binary aside: %w",
				err,
			)
		}

		if err := os.Rename(stage, target); err != nil {
			_ = os.Rename(old, target)

			return fmt.Errorf(
				"failed to move the replacement into place: %w",
				err,
			)
		}

		_ = os.Remove(old)

		return nil
	}

	if err := os.Rename(stage, target); err != nil {
		return fmt.Errorf(
			"failed to replace the running binary: %w",
			err,
		)
	}

	return nil
}

// Cleanup repairs the leftovers of an interrupted update before the
// update check runs: on Windows, a dance stopped between its two
// renames left the binary shifted away from its path, and the next
// start restores it. Completed updates leave their old copy behind
// until that same start removes it.
func Cleanup(log *logger.Logger) error {
	target, err := executablePath()
	if err != nil {
		return fmt.Errorf(
			"failed to locate the running binary: %w",
			err,
		)
	}

	return recoverSwap(target)
}

// recoverSwap restores the binary when its path is missing and only
// its shifted old copy remains, otherwise drops a completed update's
// old copy.
func recoverSwap(target string) error {
	old := target + ".old"

	if _, statErr := os.Stat(target); os.IsNotExist(statErr) {
		if _, oldErr := os.Stat(old); oldErr == nil {
			return os.Rename(old, target)
		}

		return nil
	}

	_ = os.Remove(old)

	return nil
}
