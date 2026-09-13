package fetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
)

// Downloads chapter images using an exact base URL.
func DownloadChapterFromBaseURL(baseURL, chapter, destDir string, log *logger.Logger) error {
	chapterURL := fmt.Sprintf("%s/%s", baseURL, chapter)
	urls, err := CollectSequentialJPGURLs(chapterURL, log)
	if err != nil {
		return err
	}
	return DownloadImages(urls, destDir, log)
}

// Probes sequential JPG pages and returns valid URLs.
func CollectSequentialJPGURLs(chapterURL string, log *logger.Logger) ([]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var urls []string
	for page := 1; ; page++ {
		imgURL := fmt.Sprintf("%s/%d.jpg", chapterURL, page)

		req, err := http.NewRequest("GET", imgURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP GET failed: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Debug("No more pages (status %d at page %d)\n", resp.StatusCode, page)
			break
		}

		urls = append(urls, imgURL)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no images found at %s", chapterURL)
	}

	return urls, nil
}

// Downloads a list of image URLs into destDir as 001.jpg, 002.jpg...
func DownloadImages(imageURLs []string, destDir string, log *logger.Logger) error {
	if len(imageURLs) == 0 {
		return fmt.Errorf("empty image URL list")
	}

	const defaultDirPerm = 0755
	if err := os.MkdirAll(destDir, defaultDirPerm); err != nil {
		return fmt.Errorf("failed to create destDir: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for i, imgURL := range imageURLs {
		req, err := http.NewRequest("GET", imgURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP GET failed: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, imgURL)
		}

		imgPath := filepath.Join(destDir, fmt.Sprintf("%03d.jpg", i+1))
		outFile, err := os.Create(imgPath)
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to create image file: %w", err)
		}

		_, err = io.Copy(outFile, resp.Body)
		outFile.Close()
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}

		log.Debug("Downloaded page %d\n", i+1)
	}

	return nil
}
