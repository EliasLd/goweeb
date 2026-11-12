package convert

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// encodeTestJPEG writes a deterministic single-color JPEG to path.
func encodeTestJPEG(
	t *testing.T,
	path string,
	pixel color.RGBA,
) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 8, 11))
	img.Set(1, 1, pixel)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode JPEG: %v", err)
	}

	writeTestFile(t, path, buf.Bytes())

	return buf.Bytes()
}

func readTestZip(
	t *testing.T,
	path string,
) *zip.Reader {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	reader, err := zip.NewReader(
		bytes.NewReader(data),
		int64(len(data)),
	)
	if err != nil {
		t.Fatalf("failed to open CBZ: %v", err)
	}

	return reader
}

func readTestZipEntry(
	t *testing.T,
	file *zip.File,
) []byte {
	t.Helper()

	rc, err := file.Open()
	if err != nil {
		t.Fatalf("failed to open zip entry %s: %v", file.Name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read zip entry %s: %v", file.Name, err)
	}

	return data
}

func TestImagesToCBZ(t *testing.T) {
	dir := t.TempDir()

	expected := map[string][]byte{
		"001.jpg": encodeTestJPEG(
			t,
			filepath.Join(dir, "001.jpg"),
			color.RGBA{R: 0x10, A: 0xFF},
		),
		"002.jpg": encodeTestJPEG(
			t,
			filepath.Join(dir, "002.jpg"),
			color.RGBA{R: 0x20, A: 0xFF},
		),
		"003.jpg": encodeTestJPEG(
			t,
			filepath.Join(dir, "003.jpg"),
			color.RGBA{R: 0x30, A: 0xFF},
		),
	}

	// Non-image files must be ignored.
	writeTestFile(
		t,
		filepath.Join(dir, "notes.txt"),
		[]byte("not an image"),
	)

	cbzPath := filepath.Join(dir, "out.cbz")

	if err := ImagesToCBZ(dir, cbzPath, nil); err != nil {
		t.Fatalf("ImagesToCBZ() error: %v", err)
	}

	reader := readTestZip(t, cbzPath)

	if len(reader.File) != len(expected) {
		t.Fatalf(
			"archive contains %d entries, want %d",
			len(reader.File),
			len(expected),
		)
	}

	for i, want := range []string{"001.jpg", "002.jpg", "003.jpg"} {
		file := reader.File[i]

		if file.Name != want {
			t.Errorf(
				"entry %d = %q, want %q",
				i,
				file.Name,
				want,
			)
		}

		if file.Method != zip.Store {
			t.Errorf(
				"entry %s must use zip.Store",
				file.Name,
			)
		}

		if !bytesEqual(
			readTestZipEntry(t, file),
			expected[want],
		) {
			t.Errorf(
				"entry %s contents differ from the source image",
				file.Name,
			)
		}
	}
}

func TestImagesToCBZKeepsContentVerbatim(t *testing.T) {
	dir := t.TempDir()

	// Chapter pages may actually be 16-bit PNGs saved under a .jpg
	// name; archives must keep the original bytes untouched.
	src := filepath.Join(dir, "001.jpg")

	img := image.NewGray16(image.Rect(0, 0, 12, 7))
	img.Set(3, 5, color.Gray16{Y: 0xC0DE})

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}

	writeTestFile(t, src, pngBuf.Bytes())

	cbzPath := filepath.Join(dir, "out.cbz")

	if err := ImagesToCBZ(dir, cbzPath, nil); err != nil {
		t.Fatalf("ImagesToCBZ() error: %v", err)
	}

	reader := readTestZip(t, cbzPath)

	if len(reader.File) != 1 {
		t.Fatalf(
			"archive contains %d entries, want 1",
			len(reader.File),
		)
	}

	entry := readTestZipEntry(t, reader.File[0])

	if !bytesEqual(entry, pngBuf.Bytes()) {
		t.Error("expected the archived page content to be untouched")
	}
}

func TestImagesToCBZEmptyDir(t *testing.T) {
	dir := t.TempDir()

	err := ImagesToCBZ(
		dir,
		filepath.Join(dir, "out.cbz"),
		nil,
	)
	if err == nil {
		t.Fatal("expected an error when the directory has no images")
	}
}

// TestImageTreeToCBZ packs a staged volume tree and keeps the
// chapter-folder layout inside the archive.
func TestImageTreeToCBZ(t *testing.T) {
	volDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(volDir, "001"), 0o755); err != nil {
		t.Fatalf("failed to stage chapter 001: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(volDir, "002"), 0o755); err != nil {
		t.Fatalf("failed to stage chapter 002: %v", err)
	}

	page1 := encodeTestJPEG(
		t,
		filepath.Join(volDir, "001", "001.jpg"),
		color.RGBA{R: 0xFF, A: 0xFF},
	)
	page2 := encodeTestJPEG(
		t,
		filepath.Join(volDir, "001", "002.jpg"),
		color.RGBA{G: 0xFF, A: 0xFF},
	)
	page3 := encodeTestJPEG(
		t,
		filepath.Join(volDir, "002", "001.jpg"),
		color.RGBA{B: 0xFF, A: 0xFF},
	)

	cbzPath := filepath.Join(t.TempDir(), "out.cbz")

	info := ComicInfo{
		Series: "Berserk",
		Number: "",
		Volume: 1,
		Count:  43,
		Title:  "Vol. 01",
		Manga:  MangaYes,
	}

	if err := ImageTreeToCBZ(volDir, cbzPath, &info); err != nil {
		t.Fatalf("ImageTreeToCBZ() error: %v", err)
	}

	reader := readTestZip(t, cbzPath)

	wantNames := []string{
		"ComicInfo.xml",
		"001/001.jpg",
		"001/002.jpg",
		"002/001.jpg",
	}

	if len(reader.File) != len(wantNames) {
		t.Fatalf(
			"archive contains %d entries, want %d",
			len(reader.File),
			len(wantNames),
		)
	}

	for i, want := range wantNames {
		if reader.File[i].Name != want {
			t.Errorf(
				"entry %d = %q, want %q",
				i,
				reader.File[i].Name,
				want,
			)
		}
	}

	// Pages are stored verbatim under their staged names, so
	// readers keep the real formats and the padded tree walks in
	// reading order.
	if got := readTestZipEntry(t, reader.File[1]); !bytes.Equal(got, page1) {
		t.Errorf("001/001.jpg does not carry the staged page verbatim")
	}

	if got := readTestZipEntry(t, reader.File[2]); !bytes.Equal(got, page2) {
		t.Errorf("001/002.jpg does not carry the staged page verbatim")
	}

	if got := readTestZipEntry(t, reader.File[3]); !bytes.Equal(got, page3) {
		t.Errorf("002/001.jpg does not carry the staged page verbatim")
	}

	var decoded ComicInfo

	if err := xml.Unmarshal(
		readTestZipEntry(t, reader.File[0]),
		&decoded,
	); err != nil {
		t.Fatalf("failed to parse ComicInfo.xml: %v", err)
	}

	if decoded.Volume != 1 ||
		decoded.Number != "" ||
		decoded.Title != "Vol. 01" {
		t.Errorf("unexpected metadata: %+v", decoded)
	}

	if decoded.PageCount != 3 {
		t.Errorf("PageCount = %d, want 3", decoded.PageCount)
	}
}

// TestImageTreeToCBZRefusesPartial verifies bundling refuses a
// staging tree that cannot represent a complete volume.
func TestImageTreeToCBZRefusesPartial(t *testing.T) {
	t.Run("no chapter folders", func(t *testing.T) {
		dir := t.TempDir()

		if err := ImageTreeToCBZ(dir, filepath.Join(dir, "out.cbz"), nil); err == nil {
			t.Errorf("empty staging tree must fail")
		}
	})

	t.Run("empty chapter folder", func(t *testing.T) {
		volDir := t.TempDir()

		if err := os.MkdirAll(filepath.Join(volDir, "001"), 0o755); err != nil {
			t.Fatalf("failed to stage chapter: %v", err)
		}

		if err := os.MkdirAll(filepath.Join(volDir, "002"), 0o755); err != nil {
			t.Fatalf("failed to stage chapter: %v", err)
		}

		encodeTestJPEG(
			t,
			filepath.Join(volDir, "001", "001.jpg"),
			color.RGBA{A: 0xFF},
		)

		if err := ImageTreeToCBZ(volDir, filepath.Join(volDir, "out.cbz"), nil); err == nil {
			t.Errorf("a chapter without pages must fail the bundle")
		}
	})
}

// TestImagesToCBZWritesComicInfo exercises the metadata entry of one chapter.
func TestImagesToCBZWritesComicInfo(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"001.jpg", "002.jpg", "003.jpg"} {
		encodeTestJPEG(
			t,
			filepath.Join(dir, name),
			color.RGBA{A: 0xFF},
		)
	}

	cbzPath := filepath.Join(dir, "out.cbz")

	info := ComicInfo{
		Series: "One Piece",
		Number: "10",
		Count:  42,
		Title:  "Chapter 10",
		Manga:  MangaYes,
	}

	if err := ImagesToCBZ(dir, cbzPath, &info); err != nil {
		t.Fatalf("ImagesToCBZ() error: %v", err)
	}

	reader := readTestZip(t, cbzPath)

	if len(reader.File) != 4 {
		t.Fatalf(
			"archive contains %d entries, want 4",
			len(reader.File),
		)
	}

	metadata := reader.File[0]

	if metadata.Name != "ComicInfo.xml" {
		t.Fatalf(
			"first entry = %q, want ComicInfo.xml",
			metadata.Name,
		)
	}

	if metadata.Method != zip.Deflate {
		t.Errorf(
			"entry %s must use zip.Deflate",
			metadata.Name,
		)
	}

	var decoded ComicInfo

	if err := xml.Unmarshal(
		readTestZipEntry(t, metadata),
		&decoded,
	); err != nil {
		t.Fatalf(
			"failed to parse ComicInfo.xml: %v",
			err,
		)
	}

	if decoded.Series != "One Piece" ||
		decoded.Number != "10" ||
		decoded.Count != 42 ||
		decoded.Manga != MangaYes {
		t.Errorf(
			"unexpected metadata: %+v",
			decoded,
		)
	}

	// PageCount is filled by ImagesToCBZ with the real number of
	// archived pages, and the caller's struct stays untouched.
	if decoded.PageCount != 3 {
		t.Errorf(
			"PageCount = %d, want 3",
			decoded.PageCount,
		)
	}

	if info.PageCount != 0 {
		t.Errorf(
			"caller's ComicInfo.PageCount = %d, want untouched 0",
			info.PageCount,
		)
	}

	// Pages keep their verbatim ordering right after the metadata.
	for i, want := range []string{"001.jpg", "002.jpg", "003.jpg"} {
		file := reader.File[i+1]

		if file.Name != want {
			t.Errorf(
				"entry %d = %q, want %q",
				i+1,
				file.Name,
				want,
			)
		}

		if file.Method != zip.Store {
			t.Errorf(
				"entry %s must use zip.Store",
				file.Name,
			)
		}
	}
}
