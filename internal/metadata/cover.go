package metadata

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// knownCoverExtensions are the image types cover CDNs serve; a
// recognizable extension is preserved, anything else falls back to
// jpg (the overwhelmingly common case).
var knownCoverExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// CoverExtension derives the file extension of a cover address: the
// URL path's own extension when recognizable, .jpg otherwise (query
// strings and fragment do not leak into the result).
func CoverExtension(coverURL string) string {
	parsed, err := url.Parse(coverURL)
	if err != nil {
		return ".jpg"
	}

	extension := strings.ToLower(path.Ext(parsed.Path))

	if !knownCoverExtensions[extension] {
		return ".jpg"
	}

	return extension
}

// downloadCoverTo fetches the cover into the destination base name,
// choosing the extension from the URL, and returns the recorded file
// name. The download is atomic (temporary file renamed into place on
// success), so a half-written cover can never be recorded.
func downloadCoverTo(
	coverURL string,
	basePath string,
	headers map[string]string,
	log *logger.Logger,
) (string, error) {
	destination := basePath + CoverExtension(coverURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	request, err := http.NewRequest(http.MethodGet, coverURL, nil)
	if err != nil {
		return "", fmt.Errorf(
			"failed to create the cover request: %w",
			err,
		)
	}

	// Shared polite defaults; per-provider headers apply on top.
	request.Header.Set(
		"User-Agent",
		"goweeb (+https://github.com/EliasLd/goweeb)",
	)
	request.Header.Set("Accept", "image/*,*/*")

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	if err := httpx.DownloadToFile(
		client,
		request,
		destination,
		log,
	); err != nil {
		return "", err
	}

	// A well-formed cover that is not an image (an error page
	// answering 200, for example) must not be recorded.
	if err := verifyCoverImage(destination); err != nil {
		_ = os.Remove(destination)

		return "", err
	}

	return filepath.Base(destination), nil
}

// verifyCoverImage checks the recorded cover starts with the magic
// bytes of the image types cover CDNs serve.
func verifyCoverImage(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open the cover: %w", err)
	}
	defer file.Close()

	const headBytes = 512

	head := make([]byte, headBytes)

	read, ioErr := io.ReadFull(file, head)
	if read == 0 && ioErr != nil && !errors.Is(ioErr, io.EOF) &&
		!errors.Is(ioErr, io.ErrUnexpectedEOF) {
		return fmt.Errorf("failed to read the cover: %w", ioErr)
	}

	head = head[:read]

	if len(head) == 0 {
		return errors.New("the cover file is empty")
	}

	if !looksLikeImage(head) {
		return errors.New("the cover file is not an image")
	}

	return nil
}

// looksLikeImage matches the magic bytes of the image types cover
// CDNs serve.
func looksLikeImage(head []byte) bool {
	switch {
	case len(head) >= 3 &&
		head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return true // JPEG

	case len(head) >= 8 &&
		string(head[:8]) == "\x89PNG\r\n\x1a\n":
		return true // PNG

	case len(head) >= 12 &&
		string(head[:4]) == "RIFF" &&
		string(head[8:12]) == "WEBP":
		return true // WEBP
	}

	return false
}
