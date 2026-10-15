package convert

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

func discardLogger() *logger.Logger {
	return logger.New(io.Discard, logger.LevelError)
}

func writeTestFile(
	t *testing.T,
	path string,
	data []byte,
) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf(
			"failed to write %s: %v",
			path,
			err,
		)
	}
}

func encodeTestImage(
	t *testing.T,
	path string,
	img image.Image,
) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf(
			"failed to create %s: %v",
			path,
			err,
		)
	}

	encodeErr := png.Encode(f, img)
	closeErr := f.Close()

	if encodeErr != nil {
		t.Fatalf("failed to encode PNG: %v", encodeErr)
	}

	if closeErr != nil {
		t.Fatalf("failed to close PNG: %v", closeErr)
	}
}

func readTestFile(
	t *testing.T,
	path string,
) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	return data
}

func TestIsWebP(t *testing.T) {
	dir := t.TempDir()

	webpPath := filepath.Join(dir, "sample.webp")
	writeTestFile(
		t,
		webpPath,
		[]byte("RIFF\x00\x00\x00\x00WEBP\x00\x00"),
	)

	jpgPath := filepath.Join(dir, "sample.jpg")
	writeTestFile(
		t,
		jpgPath,
		[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
	)

	isWebP, err := IsWebP(webpPath)
	if err != nil {
		t.Fatalf("IsWebP() error: %v", err)
	}

	if !isWebP {
		t.Error("expected a WebP payload to be detected")
	}

	isWebP, err = IsWebP(jpgPath)
	if err != nil {
		t.Fatalf("IsWebP() error: %v", err)
	}

	if isWebP {
		t.Error("expected a JPEG payload not to be detected as WebP")
	}
}

func TestIsSixteenBitPNG(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name     string
		filename string
		image    image.Image
		expected bool
	}{
		{
			name:     "16-bit grayscale PNG",
			filename: "gray16.jpg",
			image: func() image.Image {
				img := image.NewGray16(
					image.Rect(0, 0, 8, 11),
				)
				img.Set(0, 0, color.Gray16{Y: 0xFFFF})
				return img
			}(),
			expected: true,
		},
		{
			name:     "16-bit color PNG",
			filename: "nrgba64.jpg",
			image: func() image.Image {
				img := image.NewNRGBA64(
					image.Rect(0, 0, 8, 11),
				)
				img.Set(0, 0, color.NRGBA64{R: 0x1111, A: 0xFFFF})
				return img
			}(),
			expected: true,
		},
		{
			name:     "8-bit grayscale PNG",
			filename: "gray.jpg",
			image: func() image.Image {
				img := image.NewGray(image.Rect(0, 0, 8, 11))
				img.Set(0, 0, color.Gray{Y: 0xFF})
				return img
			}(),
			expected: false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(dir, testCase.filename)
			encodeTestImage(t, path, testCase.image)

			got, err := IsSixteenBitPNG(path)
			if err != nil {
				t.Fatalf("IsSixteenBitPNG() error: %v", err)
			}

			if got != testCase.expected {
				t.Errorf(
					"expected 16-bit detection %v, got %v",
					testCase.expected,
					got,
				)
			}
		})
	}
}

func TestIsSixteenBitPNGOtherContent(t *testing.T) {
	dir := t.TempDir()

	jpgPath := filepath.Join(dir, "photo.jpg")
	writeTestFile(
		t,
		jpgPath,
		[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
	)

	got, err := IsSixteenBitPNG(jpgPath)
	if err != nil {
		t.Fatalf("IsSixteenBitPNG() error: %v", err)
	}

	if got {
		t.Error("expected a JPEG payload not to be detected as 16-bit PNG")
	}

	// A file too short to hold a PNG header is not detected either.
	shortPath := filepath.Join(dir, "short.jpg")
	writeTestFile(t, shortPath, []byte("tiny"))

	got, err = IsSixteenBitPNG(shortPath)
	if err != nil {
		t.Fatalf("IsSixteenBitPNG() error: %v", err)
	}

	if got {
		t.Error("expected a short payload not to be detected as 16-bit PNG")
	}
}

func TestPNGToJPG(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "001.jpg")

	img := image.NewGray16(image.Rect(0, 0, 12, 7))
	img.Set(3, 5, color.Gray16{Y: 0xC0DE})
	encodeTestImage(t, src, img)

	original := readTestFile(t, src)

	if err := PNGToJPG(src, src); err != nil {
		t.Fatalf("PNGToJPG() error: %v", err)
	}

	converted := readTestFile(t, src)

	// The file was overwritten with JPEG data.
	if len(converted) < 3 ||
		converted[0] != 0xFF || converted[1] != 0xD8 {
		t.Error("expected the converted file to start with JPEG magic bytes")
	}

	if bytesEqual(converted, original) {
		t.Error("expected the source file content to change")
	}
}

func TestImagesToPDFConvertsSixteenBitPNGs(t *testing.T) {
	dir := t.TempDir()

	encodeTestImage(
		t,
		filepath.Join(dir, "001.jpg"),
		func() image.Image {
			img := image.NewGray16(image.Rect(0, 0, 584, 754))
			img.Set(0, 0, color.Gray16{Y: 0xFFFF})
			return img
		}(),
	)

	encodeTestImage(
		t,
		filepath.Join(dir, "002.jpg"),
		func() image.Image {
			img := image.NewNRGBA64(image.Rect(0, 0, 584, 754))
			img.Set(0, 0, color.NRGBA64{R: 0x1234, A: 0xFFFF})
			return img
		}(),
	)

	pdfPath := filepath.Join(dir, "out.pdf")

	if err := ImagesToPDF(dir, pdfPath, discardLogger()); err != nil {
		t.Fatalf("ImagesToPDF() error: %v", err)
	}

	info, err := os.Stat(pdfPath)
	if err != nil {
		t.Fatalf("expected a generated PDF: %v", err)
	}

	if info.Size() == 0 {
		t.Error("expected a non-empty PDF")
	}
}

func TestImagesToPDFKeepsEightBitPNGs(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "001.jpg")

	img := image.NewGray(image.Rect(0, 0, 20, 10))
	img.Set(1, 1, color.Gray{Y: 0xFF})
	encodeTestImage(t, path, img)

	original := readTestFile(t, path)

	pdfPath := filepath.Join(dir, "out.pdf")

	if err := ImagesToPDF(dir, pdfPath, discardLogger()); err != nil {
		t.Fatalf("ImagesToPDF() error: %v", err)
	}

	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("expected a generated PDF: %v", err)
	}

	// gopdf embeds 8-bit PNGs natively; the file must be untouched.
	if !bytesEqual(readTestFile(t, path), original) {
		t.Error("expected the 8-bit PNG file to remain unchanged")
	}
}

func TestImagesToPDFJPEG(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "001.jpg")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}

	if err := jpeg.Encode(f, image.NewRGBA(image.Rect(0, 0, 20, 10)), nil); err != nil {
		t.Fatalf("failed to encode JPEG: %v", err)
	}

	closeErr := f.Close()
	if closeErr != nil {
		t.Fatalf("failed to close JPEG: %v", closeErr)
	}

	pdfPath := filepath.Join(dir, "out.pdf")

	if err := ImagesToPDF(dir, pdfPath, discardLogger()); err != nil {
		t.Fatalf("ImagesToPDF() error: %v", err)
	}

	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("expected a generated PDF: %v", err)
	}
}

func TestImagesToPDFEmptyDir(t *testing.T) {
	dir := t.TempDir()

	err := ImagesToPDF(dir, filepath.Join(dir, "out.pdf"), discardLogger())
	if err == nil {
		t.Fatal("expected an error when the directory has no images")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
