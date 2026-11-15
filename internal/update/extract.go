package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
)

// binaryBase is the bare member name the release archives carry,
// with a Windows extension appended.
func binaryBase() string {
	extension := ""

	if runtime.GOOS == "windows" {
		extension = ".exe"
	}

	return "goweeb" + extension
}

// extractBinary stages the archive's single binary member to
// stagePath, with the archive format read off the asset URL.
func extractBinary(
	archivePath string,
	assetURL string,
	stagePath string,
) error {
	if strings.HasSuffix(assetURL, ".zip") {
		return extractZIP(archivePath, stagePath)
	}

	return extractTAR(archivePath, stagePath)
}

// isFlatBinaryMember accepts the bare binary member only: a nested
// or relative directory member is an unexpected layout, and a
// traversal name is an attack.
func isFlatBinaryMember(name string) bool {
	if !strings.EqualFold(path.Base(name), binaryBase()) {
		return false
	}

	return path.Dir(name) == "."
}

// extractTAR walks a gzip/tar archive for its binary member.
func extractTAR(archivePath, stagePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf(
			"failed to open the downloaded archive: %w",
			err,
		)
	}

	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf(
			"failed to open the archive: %w",
			err,
		)
	}

	reader := tar.NewReader(gzipReader)

	found := false

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf(
				"failed to read the archive: %w",
				err,
			)
		}

		if !header.FileInfo().Mode().IsRegular() ||
			!isFlatBinaryMember(header.Name) {
			continue
		}

		if found {
			return fmt.Errorf(
				"the archive carries several %s members",
				binaryBase(),
			)
		}

		if header.Size <= 0 || header.Size > maxBinarySize {
			return fmt.Errorf(
				"the archive member %s carries an unusable size (%d)",
				header.Name,
				header.Size,
			)
		}

		if err := extractRegular(
			reader,
			stagePath,
			header.Size,
		); err != nil {
			return err
		}

		found = true
	}

	if !found {
		return fmt.Errorf(
			"the archive carries no %s binary",
			binaryBase(),
		)
	}

	return nil
}

// extractZIP walks a zip archive for its binary member.
func extractZIP(archivePath, stagePath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf(
			"failed to open the archive: %w",
			err,
		)
	}

	defer reader.Close()

	found := false

	for _, member := range reader.File {
		if !member.FileInfo().Mode().IsRegular() ||
			!isFlatBinaryMember(member.Name) {
			continue
		}

		if found {
			return fmt.Errorf(
				"the archive carries several %s members",
				binaryBase(),
			)
		}

		if member.UncompressedSize64 == 0 ||
			member.UncompressedSize64 > uint64(maxBinarySize) {
			return fmt.Errorf(
				"the archive member %s carries an unusable size (%d)",
				member.Name,
				member.UncompressedSize64,
			)
		}

		source, err := member.Open()
		if err != nil {
			return fmt.Errorf(
				"failed to open the archive member: %w",
				err,
			)
		}

		err = extractRegular(
			source,
			stagePath,
			int64(member.UncompressedSize64),
		)

		source.Close()

		if err != nil {
			return err
		}

		found = true
	}

	if !found {
		return fmt.Errorf(
			"the archive carries no %s binary",
			binaryBase(),
		)
	}

	return nil
}

// extractRegular copies one member to its staged path, executable
// and ready for the swap.
func extractRegular(
	reader io.Reader,
	stagePath string,
	size int64,
) error {
	out, err := os.OpenFile(
		stagePath,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to stage the replacement binary: %w",
			err,
		)
	}

	defer out.Close()

	written, err := io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf(
			"failed to extract the binary: %w",
			err,
		)
	}

	if written != size {
		return fmt.Errorf(
			"the extraction ended early (%d of %d bytes)",
			written,
			size,
		)
	}

	return os.Chmod(stagePath, 0o755)
}
