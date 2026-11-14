package fetch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

func TestDownloadImagesWithHeadersPageTransform(t *testing.T) {
	const encryptedPrefix = "cipher:"

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/encrypted/1" {
				_, _ = w.Write([]byte(encryptedPrefix + "jpeg-bytes"))

				return
			}

			_, _ = w.Write([]byte("jpeg-bytes"))
		},
	))
	defer server.Close()

	transform := func(pageURL string, data []byte) ([]byte, bool) {
		text := string(data)

		if !strings.HasPrefix(text, encryptedPrefix) {
			return nil, false
		}

		return []byte(strings.TrimPrefix(text, encryptedPrefix)), true
	}

	dest := t.TempDir()

	err := DownloadImagesWithHeaders(
		[]string{server.URL + "/encrypted/1", server.URL + "/plain/2"},
		dest,
		nil,
		transform,
		logger.New(io.Discard, logger.LevelInfo),
	)
	if err != nil {
		t.Fatalf("DownloadImagesWithHeaders() error: %v", err)
	}

	first, err := os.ReadFile(filepath.Join(dest, "001.jpg"))
	if err != nil {
		t.Fatalf("read first staged page: %v", err)
	}

	if string(first) != "jpeg-bytes" {
		t.Errorf(
			"first page = %q, want the transformed bytes",
			first,
		)
	}

	second, err := os.ReadFile(filepath.Join(dest, "002.jpg"))
	if err != nil {
		t.Fatalf("read second staged page: %v", err)
	}

	if string(second) != "jpeg-bytes" {
		t.Errorf(
			"second page = %q, want the original bytes",
			second,
		)
	}
}

func TestTransformStagedImageMissingFile(t *testing.T) {
	err := transformStagedImage(
		func(_ string, _ []byte) ([]byte, bool) {
			return nil, true
		},
		"https://site/p",
		filepath.Join(t.TempDir(), "missing.jpg"),
	)
	if err == nil {
		t.Fatal("expected the missing staged page to fail the download")
	}

	if !strings.Contains(err.Error(), "failed to read staged page") {
		t.Errorf("expected the staged-read error, got %v", err)
	}
}

func TestDownloadImagesEmptyURLList(t *testing.T) {
	err := DownloadImages(nil, t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected the empty URL list to be refused")
	}
}
