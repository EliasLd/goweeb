package convert

import (
	"archive/zip"
	"bytes"
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

	if err := ImagesToCBZ(dir, cbzPath); err != nil {
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

	if err := ImagesToCBZ(dir, cbzPath); err != nil {
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
	)
	if err == nil {
		t.Fatal("expected an error when the directory has no images")
	}
}
