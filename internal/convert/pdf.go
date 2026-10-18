package convert

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/signintech/gopdf"
	"golang.org/x/image/webp"

	"github.com/EliasLd/goweeb/internal/logger"
)

var pngMagicNumber = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

var pngIHDRChunk = []byte{0x49, 0x48, 0x44, 0x52}

// Opens the file at path and checks
// if it is encoded in WEBP format
func IsWebP(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	header := make([]byte, 12)
	_, err = f.Read(header)
	if err != nil {
		return false, err
	}

	if string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return true, nil
	}
	return false, nil
}

// Converts a WebP image in the jpg format
func WebPToJPG(srcPath, destPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("Failed to open WebP file: %w", err)
	}
	defer f.Close()

	img, err := webp.Decode(f)
	if err != nil {
		return fmt.Errorf("Failed to decode WebP image: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("Failed to create JPEG file: %w", err)
	}
	defer out.Close()

	// Encode image into JPEG format
	opt := jpeg.Options{Quality: 90}
	if err := jpeg.Encode(out, img, &opt); err != nil {
		return fmt.Errorf("Failed to encode JPEG: %w", err)
	}

	return nil
}

// IsSixteenBitPNG checks whether the file at path contains a PNG with
// a bit depth greater than 8. gopdf rejects such files with
// "16-bit depth not supported", so they must be converted before
// being embedded into a PDF.
//
// This happens when a website serves pages whose content does not
// match their name, for example Anime-Sama serving 16-bit PNGs under a
// .jpg name on some manga such as Bleach.
//
// PNG layout check: 8 signature bytes, a 4-byte chunk length, the
// "IHDR" chunk type, then the IHDR data whose 5th byte is the bit
// depth (width [16:20], height [20:24], bit depth [24]).
func IsSixteenBitPNG(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	header := make([]byte, 25)

	if _, err := io.ReadFull(f, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// Too short to be a 16-bit PNG.
			return false, nil
		}

		return false, err
	}

	if !bytes.Equal(header[0:8], pngMagicNumber) ||
		!bytes.Equal(header[12:16], pngIHDRChunk) {
		return false, nil
	}

	return header[24] > 8, nil
}

// PNGToJPG converts the PNG image at srcPath into a JPEG encoded at
// destPath.
//
// The source handle is closed before creating the destination file so
// in-place conversions never truncate a file that is still open.
func PNGToJPG(srcPath, destPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open PNG file: %w", err)
	}

	img, err := png.Decode(f)
	closeErr := f.Close()
	if err != nil {
		return fmt.Errorf("failed to decode PNG image: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("failed to close PNG file: %w", closeErr)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create JPEG file: %w", err)
	}
	defer out.Close()

	opt := jpeg.Options{Quality: 90}
	if err := jpeg.Encode(out, img, &opt); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return nil
}

// listImageFiles returns the sorted image file basenames of
// imagesDir, ignoring directories and unknown extensions. Callers
// decide how to treat an empty list.
func listImageFiles(imagesDir string) ([]string, error) {
	files, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, err
	}

	var imageFiles []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".webp" {
			imageFiles = append(imageFiles, f.Name())
		}
	}

	sort.Strings(imageFiles)

	return imageFiles, nil
}

func ImagesToPDF(imagesDir, outputPDFPath string, log *logger.Logger) error {
	imageFiles, err := listImageFiles(imagesDir)
	if err != nil {
		log.Error("Failed to read image dir: %v", err)
		return fmt.Errorf("Failed to read image dir: %v", err)
	}

	if len(imageFiles) == 0 {
		return fmt.Errorf("No JPG files found in %s", imagesDir)
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{})

	for _, imageName := range imageFiles {
		imagePath := filepath.Join(imagesDir, imageName)
		lower := strings.ToLower(imageName)

		// Handle .webp
		if strings.HasSuffix(lower, ".webp") || strings.HasSuffix(lower, ".jpg") {
			isWebP, err := IsWebP(imagePath)
			if err != nil {
				return fmt.Errorf("Failed to check whether %s is WebP: %v", imagePath, err)
			}
			if isWebP {
				err := WebPToJPG(imagePath, imagePath)
				if err != nil {
					log.Error("Failed to convert WebP: %v\n", err)
					continue
				}
			}
		}

		// Handle 16-bit PNGs that gopdf cannot embed directly, such as
		// Anime-Sama pages served as 16-bit PNGs named .jpg.
		isSixteenBitPNG, err := IsSixteenBitPNG(imagePath)
		if err != nil {
			return fmt.Errorf(
				"failed to check whether %s is a 16-bit PNG: %w",
				imagePath,
				err,
			)
		}

		if isSixteenBitPNG {
			err := PNGToJPG(imagePath, imagePath)
			if err != nil {
				log.Error("failed to convert PNG: %v\n", err)
				continue
			}
		}

		imageFile, err := os.Open(imagePath)
		if err != nil {
			return fmt.Errorf("Failed to open image %s: %w", imageName, err)
		}

		imageConfig, _, err := image.DecodeConfig(imageFile)
		imageFile.Close()
		if err != nil {
			return fmt.Errorf("Failed to decode image %s: %w", imageName, err)
		}

		width := float64(imageConfig.Width)
		height := float64(imageConfig.Height)

		// Start a new page with exact dimensions
		pdf.AddPageWithOption(gopdf.PageOption{
			PageSize: &gopdf.Rect{W: width, H: height},
		})

		err = pdf.Image(imagePath, 0, 0, &gopdf.Rect{W: width, H: height})
		if err != nil {
			return fmt.Errorf("Error adding image %s: %w", imageName, err)
		}
	}

	// Save newly created PDF file
	if err := pdf.WritePdf(outputPDFPath); err != nil {
		log.Error("Failed to write PDF: %v", err)
		return fmt.Errorf("Failed to write PDF: %w", err)
	}

	log.Debug("PDF generated at: %s\n", outputPDFPath)

	return nil
}
