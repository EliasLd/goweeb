package fetch

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// Downloads chapter images using an exact base URL.
func DownloadChapterFromBaseURL(
	baseURL, chapter, destDir string,
	log *logger.Logger,
) error {
	chapterURL := fmt.Sprintf("%s/%s", baseURL, chapter)

	urls, err := CollectSequentialJPGURLs(chapterURL, log)
	if err != nil {
		return err
	}

	return DownloadImages(urls, destDir, log)
}

// Probes sequential JPG pages and returns valid URLs.
func CollectSequentialJPGURLs(
	chapterURL string,
	log *logger.Logger,
) ([]string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var urls []string

	for page := 1; ; page++ {
		imgURL := fmt.Sprintf(
			"%s/%d.jpg",
			chapterURL,
			page,
		)

		req, err := http.NewRequest(
			http.MethodGet,
			imgURL,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create request: %w",
				err,
			)
		}

		req.Header.Set(
			"User-Agent",
			"Mozilla/5.0",
		)

		// Retry transient network errors and HTTP failures.
		resp, err := httpx.Do(client, req, log)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to probe page %d: %w",
				page,
				err,
			)
		}

		statusCode := resp.StatusCode
		resp.Body.Close()

		switch statusCode {
		case http.StatusOK:
			urls = append(urls, imgURL)

		case http.StatusNotFound, http.StatusGone:
			// These statuses indicate the end of the
			// sequential chapter image list.
			log.Debug(
				"No more pages (status %d at page %d)\n",
				statusCode,
				page,
			)

			if len(urls) == 0 {
				return nil, fmt.Errorf(
					"no images found at %s",
					chapterURL,
				)
			}

			return urls, nil

		default:
			// A persistent 503, 429 or other unexpected
			// status must not silently truncate the chapter.
			return nil, fmt.Errorf(
				"unexpected status %d while probing page %d: %s",
				statusCode,
				page,
				imgURL,
			)
		}
	}
}

// Downloads a list of image URLs using the default request headers.
func DownloadImages(
	imageURLs []string,
	destDir string,
	log *logger.Logger,
) error {
	return DownloadImagesWithHeaders(
		imageURLs,
		destDir,
		nil,
		nil,
		log,
	)
}

// PageTransform rewrites one downloaded page's bytes before it lands
// on disk. The false return keeps the original bytes.
type PageTransform func(pageURL string, data []byte) ([]byte, bool)

// Downloads a list of image URLs with optional custom request headers
// and an optional per-page byte transform.
func DownloadImagesWithHeaders(
	imageURLs []string,
	destDir string,
	headers map[string]string,
	transform PageTransform,
	log *logger.Logger,
) error {
	if len(imageURLs) == 0 {
		return fmt.Errorf("empty image URL list")
	}

	const defaultDirPerm = 0755

	if err := os.MkdirAll(
		destDir,
		defaultDirPerm,
	); err != nil {
		return fmt.Errorf(
			"failed to create destDir: %w",
			err,
		)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for i, imgURL := range imageURLs {
		req, err := http.NewRequest(
			http.MethodGet,
			imgURL,
			nil,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to create request for page %d: %w",
				i+1,
				err,
			)
		}

		// Default behavior used by existing providers.
		req.Header.Set(
			"User-Agent",
			"Mozilla/5.0",
		)

		// Provider-specific headers override defaults.
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		imgPath := filepath.Join(
			destDir,
			fmt.Sprintf("%03d.jpg", i+1),
		)

		// Retry transient HTTP errors and interrupted
		// downloads. DownloadToFile handles temporary
		// files so incomplete images are not retained.
		if err := httpx.DownloadToFile(
			client,
			req,
			imgPath,
			log,
		); err != nil {
			return fmt.Errorf(
				"failed to download page %d: %w",
				i+1,
				err,
			)
		}

		if transform != nil {
			if err := transformStagedImage(
				transform,
				imgURL,
				imgPath,
			); err != nil {
				return fmt.Errorf(
					"failed to transform page %d: %w",
					i+1,
					err,
				)
			}
		}

		log.Debug(
			"Downloaded page %d\n",
			i+1,
		)
	}

	return nil
}

// transformStagedImage passes a freshly downloaded page through the
// transform and replaces the staged file when the bytes change.
// Sources with scrambled page bytes stage ciphertext images, so the
// rewrite runs after the download's own retry loop, on the complete
// file.
func transformStagedImage(
	transform PageTransform,
	pageURL string,
	path string,
) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read staged page: %w", err)
	}

	decoded, ok := transform(pageURL, data)
	if !ok {
		return nil
	}

	// DownloadToFile's temporary file carried the same permission.
	if err := os.WriteFile(path, decoded, 0600); err != nil {
		return fmt.Errorf("failed to replace staged page: %w", err)
	}

	return nil
}
