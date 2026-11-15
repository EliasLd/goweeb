// Package update checks GitHub for a newer goweeb release and
// replaces the running binary on request.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/version"
)

// latestReleaseURL is tested with a local server instead.
var latestReleaseURL = "https://api.github.com/repos/EliasLd/goweeb/releases/latest"

// Flavor selects the artifact group the release carries: the CLI and
// TUI binaries are published side by side.
type Flavor string

const (
	FlavorCLI Flavor = "cli"
	FlavorTUI Flavor = "tui"
)

// Release is the newest published release as far as the check saw,
// with the archive of this build's platform and its published
// checksum ready for the download step.
type Release struct {
	Tag      string
	AssetURL string
	Digest   string
}

// Check queries the latest GitHub release and reports it when it is
// newer than the running version; nil means there is nothing to
// update. Any failure — network, rate limit, unreadable payload,
// missing artifact — is an error and callers silently skip the
// update: the check never stands in the way of running goweeb.
func Check(currentVersion string, flavor Flavor, log *logger.Logger) (*Release, error) {
	client := &http.Client{
		Timeout: 4 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	result, err := httpx.ReadAll(
		client,
		req,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch the latest release: %w",
			err,
		)
	}

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"latest release lookup returned status %d",
			result.StatusCode,
		)
	}

	published, err := decodeRelease(result.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to decode the latest release: %w",
			err,
		)
	}

	if !version.IsNewer(published.TagName, currentVersion) {
		if log != nil {
			log.Debug(
				"No update: %s against running %s\n",
				published.TagName,
				currentVersion,
			)
		}

		return nil, nil
	}

	asset, err := findAsset(published.Assets, published.TagName, flavor)
	if err != nil {
		return nil, fmt.Errorf(
			"no update asset for %s/%s: %w",
			runtime.GOOS,
			runtime.GOARCH,
			err,
		)
	}

	if log != nil {
		log.Debug(
			"Update available: %s (%s)\n",
			published.TagName,
			asset.Name,
		)
	}

	return &Release{
		Tag:      published.TagName,
		AssetURL: asset.BrowserDownloadURL,
		Digest:   asset.Digest,
	}, nil
}

// latestResponse decodes only the fields the checker needs of
// https://api.github.com's latest release payload.
type latestResponse struct {
	TagName    string          `json:"tag_name"`
	Prerelease bool            `json:"prerelease"`
	Assets     []assetResponse `json:"assets"`
}

// assetResponse is one published artifact and the checksum the
// release API carries for it.
type assetResponse struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

// decodeRelease keeps its failures as errors: a malformed lookup
// never becomes an offer to update.
func decodeRelease(body []byte) (latestResponse, error) {
	var published latestResponse

	if err := json.Unmarshal(body, &published); err != nil {
		return latestResponse{}, err
	}

	if published.TagName == "" {
		return latestResponse{}, fmt.Errorf("the release carries no tag")
	}

	return published, nil
}

// findAsset picks this build's artifact by its exact published name
// and hands back the checksum the release API carries for it; a
// digest is mandatory, an unchecked replacement is no replacement.
func findAsset(
	assets []assetResponse,
	tag string,
	flavor Flavor,
) (assetResponse, error) {
	wanted := archiveName(flavor, tag, runtime.GOOS, runtime.GOARCH)

	for _, asset := range assets {
		if asset.Name != wanted {
			continue
		}

		if !isSHA256Digest(asset.Digest) {
			return assetResponse{}, fmt.Errorf(
				"asset %s carries no sha256 digest (%q)",
				asset.Name,
				asset.Digest,
			)
		}

		return asset, nil
	}

	return assetResponse{}, fmt.Errorf(
		"the release lists no %s artifact",
		wanted,
	)
}

// archiveName mirrors the release workflow's naming: the flavor
// prefix, the tag, the platform, and a zip only for Windows.
func archiveName(
	flavor Flavor,
	tag string,
	goos string,
	goarch string,
) string {
	prefix := "goweeb"

	if flavor == FlavorTUI {
		prefix += "-tui"
	}

	osName := goos
	if goos == "darwin" {
		osName = "macos"
	}

	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}

	return prefix + "-" + tag + "-" + osName + "-" + goarch + extension
}

// isSHA256Digest reports whether the text is a GitHub release digest
// of the "sha256:<hex>" shape.
func isSHA256Digest(text string) bool {
	const prefix = "sha256:"

	if len(text) != len(prefix)+64 ||
		!strings.HasPrefix(text, prefix) {
		return false
	}

	for _, r := range text[len(prefix):] {
		hex := r >= '0' && r <= '9' ||
			r >= 'a' && r <= 'f' ||
			r >= 'A' && r <= 'F'

		if !hex {
			return false
		}
	}

	return true
}
