package animesamascraper

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	"github.com/PuerkitoBio/goquery"
)

type ScanPathResult struct {
	Label string
	Path  string
}

// Fetches the manga page and extracts all available scan paths from
// the panneauScan("Label", "path") calls, completed by scan-related
// DOM anchors (deduped by resolved path).
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

	result, err := httpx.ReadAll(
		client,
		req,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch manga page: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"manga page returned status: %d",
			result.StatusCode,
		)
	}

	html := string(result.Body)
	var results []ScanPathResult

	// Result indexes by resolved path, for dedupe and label upgrades.
	seen := map[string]int{}

	// Both scan-path sources are relative or absolute references to
	// the manga page; resolve them so identical targets (planning
	// menu anchors vs panneauScan calls) dedupe correctly.
	base, err := url.Parse(strings.TrimSuffix(mangaURL, "/") + "/")
	if err != nil {
		return nil, fmt.Errorf("invalid manga URL: %w", err)
	}

	resolve := func(ref string) string {
		parsed, parseErr := url.Parse(ref)
		if parseErr != nil {
			return strings.TrimSuffix(ref, "/")
		}

		return strings.TrimSuffix(
			base.ResolveReference(parsed).String(),
			"/",
		)
	}

	// Primary source: the scan choice panels, declared as
	// panneauScan("Label", "path") calls. Template comments also
	// embed placeholder examples (such as
	// panneauScan("nom", "scan/vf")), so labels are upgraded when
	// the real call for the same path shows up later.
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

		key := resolve(path)

		if idx, exists := seen[key]; exists {
			// Template comments use placeholder labels such as
			// "nom"; replace them with the meaningful label of
			// a later call for the same path.
			if isPlaceholderLabel(results[idx].Label) &&
				!isPlaceholderLabel(label) &&
				label != "" {
				results[idx].Label = label
			}

			continue
		}
		seen[key] = len(results)

		if label == "" {
			label = path
		}

		results = append(results, ScanPathResult{
			Label: label,
			Path:  path,
		})
	}

	// Secondary source: <a href="...scan..."> anchors, for pages
	// that expose scan choices as DOM elements (or declare no
	// panneauScan call at all). Anchors resolving to a path that is
	// already known (planning menus, duplicates) are skipped.
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

			key := resolve(href)
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = len(results)

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

	if len(results) == 0 {
		return nil, fmt.Errorf("no scan paths found")
	}

	log.Info("Found %d scan option(s)\n", len(results))
	return results, nil
}

// isPlaceholderLabel reports whether a panneauScan label is one of
// the placeholder names the site's template comments use (for
// example panneauScan("nom", "scan/vf") inside a JS comment).
func isPlaceholderLabel(label string) bool {
	return label == "nom" || label == "url"
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
