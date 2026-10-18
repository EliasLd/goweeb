package convert

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// addImageToZip copies the image file into the zip archive under its
// staged basename, using the STORED method because images are already
// compressed.
func addImageToZip(
	zipWriter *zip.Writer,
	imagePath string,
	name string,
) error {
	header := &zip.FileHeader{
		Name:   name,
		Method: zip.Store,
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf(
			"failed to create zip entry: %w",
			err,
		)
	}

	source, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf(
			"failed to open image: %w",
			err,
		)
	}
	defer source.Close()

	if _, err := io.Copy(writer, source); err != nil {
		return fmt.Errorf(
			"failed to copy image data: %w",
			err,
		)
	}

	return nil
}

// ImagesToCBZ creates a CBZ comic book archive at cbzPath containing
// the image files of imagesDir, one archive per chapter.
//
// Images are stored verbatim at the archive root under their staged
// names (001.jpg, 002.jpg, …), so comic readers keep the correct page
// order. The file contents are not re-encoded: websites may serve
// formats other than JPEG under these names, and readers and tools
// such as Kindle Comic Converter detect the actual format.
func ImagesToCBZ(imagesDir, cbzPath string) error {
	names, err := listImageFiles(imagesDir)
	if err != nil {
		return fmt.Errorf(
			"failed to read image dir: %w",
			err,
		)
	}

	if len(names) == 0 {
		return fmt.Errorf(
			"no image files found in %s",
			imagesDir,
		)
	}

	out, err := os.Create(cbzPath)
	if err != nil {
		return fmt.Errorf(
			"failed to create CBZ file: %w",
			err,
		)
	}

	zipWriter := zip.NewWriter(out)

	for _, name := range names {
		imagePath := filepath.Join(imagesDir, name)

		if err := addImageToZip(
			zipWriter,
			imagePath,
			name,
		); err != nil {
			_ = zipWriter.Close()
			_ = out.Close()

			return fmt.Errorf(
				"failed to add %s to the archive: %w",
				name,
				err,
			)
		}
	}

	// Closing the zip writer flushes the central directory, so it
	// must happen before the file itself is closed.
	if err := zipWriter.Close(); err != nil {
		_ = out.Close()

		return fmt.Errorf(
			"failed to finalize the CBZ archive: %w",
			err,
		)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf(
			"failed to close the CBZ file: %w",
			err,
		)
	}

	return nil
}
