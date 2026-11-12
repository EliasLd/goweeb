package convert

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
//
// When info is not nil, a ComicInfo.xml metadata entry is written at
// the archive root before the pages; PageCount is filled inside with
// the number of archived pages, the most reliable count.
func ImagesToCBZ(
	imagesDir string,
	cbzPath string,
	info *ComicInfo,
) error {
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

	if info != nil {
		// Copy so the caller never sees PageCount filled in.
		comicInfo := *info
		comicInfo.PageCount = len(names)

		if err := addComicInfoToZip(
			zipWriter,
			&comicInfo,
		); err != nil {
			_ = zipWriter.Close()
			_ = out.Close()

			return fmt.Errorf(
				"failed to write the metadata: %w",
				err,
			)
		}
	}

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

// ImageTreeToCBZ creates a CBZ comic book archive at cbzPath from a
// staged directory tree: every immediate subdirectory of imagesDir
// becomes one folder inside the archive (one chapter per folder,
// named after its staged directory), pages stored verbatim under
// their staged names. Staged folder and page names are zero-padded,
// so readers walk the tree in reading order. Volume mode uses this
// to keep the on-disk series layout mirroring the archive layout.
//
// When info is not nil, a ComicInfo.xml metadata entry is written at
// the archive root before the pages; PageCount covers every archived
// page across folders.
func ImageTreeToCBZ(
	imagesDir string,
	cbzPath string,
	info *ComicInfo,
) error {
	folders, err := os.ReadDir(imagesDir)
	if err != nil {
		return fmt.Errorf(
			"failed to read image dir: %w",
			err,
		)
	}

	var folderNames []string
	for _, folder := range folders {
		if folder.IsDir() {
			folderNames = append(folderNames, folder.Name())
		}
	}
	sort.Strings(folderNames)

	if len(folderNames) == 0 {
		return fmt.Errorf(
			"no chapter folders found in %s",
			imagesDir,
		)
	}

	// Every page is required: a volume bundles its chapters whole.
	pages := make(map[string][]string, len(folderNames))
	totalPages := 0

	for _, folder := range folderNames {
		files, err := listImageFiles(
			filepath.Join(imagesDir, folder),
		)
		if err != nil {
			return fmt.Errorf(
				"failed to read the chapter folder %s: %w",
				folder,
				err,
			)
		}

		if len(files) == 0 {
			return fmt.Errorf(
				"no image files found in %s",
				filepath.Join(imagesDir, folder),
			)
		}

		pages[folder] = files
		totalPages += len(files)
	}

	out, err := os.Create(cbzPath)
	if err != nil {
		return fmt.Errorf(
			"failed to create CBZ file: %w",
			err,
		)
	}

	zipWriter := zip.NewWriter(out)

	if info != nil {
		// Copy so the caller never sees PageCount filled in.
		comicInfo := *info
		comicInfo.PageCount = totalPages

		if err := addComicInfoToZip(
			zipWriter,
			&comicInfo,
		); err != nil {
			_ = zipWriter.Close()
			_ = out.Close()

			return fmt.Errorf(
				"failed to write the metadata: %w",
				err,
			)
		}
	}

	for _, folder := range folderNames {
		for _, name := range pages[folder] {
			imagePath := filepath.Join(imagesDir, folder, name)

			if err := addImageToZip(
				zipWriter,
				imagePath,
				folder+"/"+name,
			); err != nil {
				_ = zipWriter.Close()
				_ = out.Close()

				return fmt.Errorf(
					"failed to add %s to the archive: %w",
					folder+"/"+name,
					err,
				)
			}
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
