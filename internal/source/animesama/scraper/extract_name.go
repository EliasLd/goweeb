package animesamascraper

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

// ExtractMangaName fetches the scan/v* page and extracts the real manga name
// Returns the exact name (e.g., "One Piece Couleur")
func ExtractMangaName(scanPageURL string, log *logger.Logger) (string, error) {
	// Add trailing slash if missing (site requires it)
	if !strings.HasSuffix(scanPageURL, "/") {
		scanPageURL += "/"
	}

	log.Debug("Extracting manga name from: %s\n", scanPageURL)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", scanPageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	result, err := httpx.ReadAll(
		client,
		req,
		log,
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to fetch scan page: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"scan page returned status %d",
			result.StatusCode,
		)
	}

	html := string(result.Body)

	// Method 1: Extract from the work title heading, currently
	// `<h1 id="titreOeuvre" class="ct-titre">Naruto  </h1>` (was an
	// h3). The text is returned verbatim on purpose: the chapter-list
	// API matches the heading text byte-for-byte, trailing whitespace
	// included — trimming breaks works whose key carries it (Naruto,
	// Bleach).
	titleRegex := regexp.MustCompile(`<h[1-6][^>]+id="titreOeuvre"[^>]*>([^<]+)</h[1-6]>`)
	match := titleRegex.FindStringSubmatch(html)

	if len(match) >= 2 {
		mangaName := match[1]
		log.Debug("Extracted manga name from title: %s\n", mangaName)
		return mangaName, nil
	}

	// Method 2: Extract from HTML comment <!-- Remplacer -> One Piece Couleur -->
	commentRegex := regexp.MustCompile(`<!--\s*Remplacer\s*->\s*([^-]+?)\s*-->`)
	match = commentRegex.FindStringSubmatch(html)

	if len(match) >= 2 {
		mangaName := strings.TrimSpace(match[1])
		log.Debug("Extracted manga name from comment: %s\n", mangaName)
		return mangaName, nil
	}

	// Method 3: Extract from JavaScript variable (if exists)
	jsRegex := regexp.MustCompile(`var\s+\w+\s*=\s*["']https?://[^"']+/s2/scans/([^"'/]+)`)
	match = jsRegex.FindStringSubmatch(html)

	if len(match) >= 2 {
		mangaName := match[1]
		log.Debug("Extracted manga name from JS: %s\n", mangaName)
		return mangaName, nil
	}

	return "", fmt.Errorf("could not extract manga name from scan page")
}
