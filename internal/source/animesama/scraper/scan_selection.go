package animesamascraper

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	"github.com/PuerkitoBio/goquery"
)

type ScanPathResult struct {
	Label string
	Path  string
}

// Fetches the manga page, extracts all available scan paths
// from panneauScan() function calls.
// Returns the path(s) (e.g., "scan/vf", "scan/va", "Scans (couleur)", etc...)
func GetAllScanPaths(mangaURL string, log *logger.Logger) ([]ScanPathResult, error) {
	log.Debug("Fetching scan path from: %s\n", mangaURL)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", mangaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manga page returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manga page: %w", err)
	}
	html := string(body)

	var results []ScanPathResult
	seen := map[string]struct{}{}

	// DOM parsing: <a href="...scan/...">
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		doc.Find("a[href]").Each(func(i int, link *goquery.Selection) {
			href, ok := link.Attr("href")
			if !ok {
				return
			}
			h := strings.TrimSpace(strings.ToLower(href))
			if !(strings.Contains(h, "/scan/") || strings.HasPrefix(h, "scan")) {
				return
			}
			if _, exists := seen[href]; exists {
				return
			}
			seen[href] = struct{}{}

			label := strings.TrimSpace(link.Text())
			if label == "" {
				label = href
			}

			results = append(results, ScanPathResult{
				Label: label,
				Path:  href,
			})
		})
	}

	// Fallback: parse panneauScan("Label","path")
	if len(results) == 0 {
		re := regexp.MustCompile(`panneauScan\(\s*["']([^"']+)["']\s*,\s*["']([^"']+)["']\s*\)`)
		matches := re.FindAllStringSubmatch(html, -1)

		for _, m := range matches {
			if len(m) < 3 {
				continue
			}

			label := strings.TrimSpace(m[1])
			path := strings.TrimSpace(m[2])

			p := strings.ToLower(path)
			if !(strings.Contains(p, "/scan/") || strings.HasPrefix(p, "scan")) {
				continue
			}

			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}

			if label == "" {
				label = path
			}

			results = append(results, ScanPathResult{
				Label: label,
				Path:  path,
			})
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no scan paths found")
	}

	log.Info("Found %d scan option(s)\n", len(results))
	return results, nil
}

// Displays scan options and asks user to choose
func PromptUserToSelectScanPath(scanPaths []ScanPathResult, log *logger.Logger) (string, error) {
	items := make([]common.SelectableItem, 0, len(scanPaths))
	for _, sp := range scanPaths {
		items = append(items, common.SelectableItem{Label: sp.Label, Value: sp.Path})
	}

	return common.PromptUserToSelect(
		items,
		"Multiple scan versions found:",
		"Select a scan version (1-%d) or 0 to cancel: ",
		log,
	)
}
