package update

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const digestA = "sha256:915532321e363e3cfb9e2f3e095ce53c68bbc525f59a8a0318b87412ff9bbc42"
const digestB = "sha256:6a724b92fde097aaf19655c376ac1e4b7d3d779946110e98434e969511066b11"

// Mirror of a real releases/latest payload: the assets carry their
// own sha256 digest.
func releaseFixture(tag string) string {
	return `{"tag_name":"` + tag + `","prerelease":false,"assets":[` +
		`{"name":"` + archiveName(FlavorTUI, tag, "linux", "amd64") +
		`","browser_download_url":"https://github.com/EliasLd/goweeb/releases/download/` + tag + `/goweeb-tui-` + tag + `-linux-amd64.tar.gz",` +
		`"digest":"` + digestA + `"},` +
		`{"name":"` + archiveName(FlavorCLI, tag, "linux", "amd64") +
		`","browser_download_url":"https://github.com/EliasLd/goweeb/releases/download/` + tag + `/goweeb-` + tag + `-linux-amd64.tar.gz",` +
		`"digest":"` + digestB + `"}` +
		`]}`
}

// newReleaseServer hosts the payload the way the release API does.
func newReleaseServer(t *testing.T, payload string) {
	original := latestReleaseURL

	mux := http.NewServeMux()
	mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, payload)
	})

	server := httptest.NewServer(mux)

	latestReleaseURL = server.URL + "/latest"

	t.Cleanup(func() {
		latestReleaseURL = original
		server.Close()
	})
}

func TestCheckOffersANewerRelease(t *testing.T) {
	newReleaseServer(t, releaseFixture("v4.7.0"))

	release, err := Check("v4.6.3", FlavorCLI, nil)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if release == nil {
		t.Fatal("expected an update offer")
	}

	if release.Tag != "v4.7.0" {
		t.Errorf("tag = %q", release.Tag)
	}

	expected := archiveName(FlavorCLI, "v4.7.0", "linux", "amd64")

	if release.AssetURL != "https://github.com/EliasLd/goweeb/releases/download/v4.7.0/"+expected {
		t.Errorf("asset URL = %q", release.AssetURL)
	}

	if release.Digest != digestB {
		t.Errorf("digest = %q", release.Digest)
	}
}

func TestCheckSilentWhenUpToDate(t *testing.T) {
	newReleaseServer(t, releaseFixture("v4.6.3"))

	release, err := Check("v4.6.3", FlavorCLI, nil)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if release != nil {
		t.Errorf("expected no offer, got %+v", release)
	}
}

func TestCheckSelectsTheFlavorAsset(t *testing.T) {
	newReleaseServer(t, releaseFixture("v4.7.0"))

	release, err := Check("v1.0.0", FlavorTUI, nil)
	if err != nil {
		t.Fatalf("Check() error: %v", err)
	}

	if release.Digest != digestA {
		t.Errorf("picked the wrong asset, digest = %q", release.Digest)
	}
}

func TestCheckRefusesAMissingAsset(t *testing.T) {
	// Only the TUI artifact published: the CLI finds nothing for
	// itself.
	newReleaseServer(t, `{"tag_name":"v4.7.0","prerelease":false,"assets":[{`+
		`"name":"`+archiveName(FlavorTUI, "v4.7.0", "linux", "amd64")+`",`+
		`"browser_download_url":"https://x",`+
		`"digest":"`+digestA+`"}]}`)

	_, err := Check("v1.0.0", FlavorCLI, nil)
	if err == nil {
		t.Fatal("expected an error for the missing asset")
	}
}

func TestCheckRefusesADigestlessAsset(t *testing.T) {
	newReleaseServer(t, `{"tag_name":"v4.7.0","prerelease":false,"assets":[{`+
		`"name":"`+archiveName(FlavorCLI, "v4.7.0", "linux", "amd64")+`",`+
		`"browser_download_url":"https://x",`+
		`"digest":""}]}`)

	_, err := Check("v1.0.0", FlavorCLI, nil)
	if err == nil {
		t.Fatal("expected an error for the missing digest")
	}
}

func TestCheckMalformedPayload(t *testing.T) {
	newReleaseServer(t, `{"tag_name":`)

	_, err := Check("v4.6.3", FlavorCLI, nil)
	if err == nil {
		t.Fatal("expected a decode error")
	}
}

func TestCheckLookupFailure(t *testing.T) {
	original := latestReleaseURL

	latestReleaseURL = "http://127.0.0.1:1/latest"

	t.Cleanup(func() { latestReleaseURL = original })

	_, err := Check("v4.6.3", FlavorCLI, nil)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestArchiveName(t *testing.T) {
	tests := []struct {
		flavor Flavor
		tag    string
		goos   string
		goarch string
		want   string
	}{
		{FlavorCLI, "v4.6.3", "linux", "amd64", "goweeb-v4.6.3-linux-amd64.tar.gz"},
		{FlavorCLI, "v4.6.3", "darwin", "arm64", "goweeb-v4.6.3-macos-arm64.tar.gz"},
		{FlavorCLI, "v4.6.3", "windows", "amd64", "goweeb-v4.6.3-windows-amd64.zip"},
		{FlavorTUI, "v4.6.3", "linux", "amd64", "goweeb-tui-v4.6.3-linux-amd64.tar.gz"},
		{FlavorTUI, "v4.6.3", "darwin", "amd64", "goweeb-tui-v4.6.3-macos-amd64.tar.gz"},
	}

	for _, test := range tests {
		got := archiveName(test.flavor, test.tag, test.goos, test.goarch)

		if got != test.want {
			t.Errorf(
				"archiveName(%s, %s, %s, %s) = %q, want %q",
				test.flavor, test.tag, test.goos, test.goarch,
				got, test.want,
			)
		}
	}
}

func TestIsSHA256Digest(t *testing.T) {
	if !isSHA256Digest(digestA) {
		t.Errorf("digest %q refused", digestA)
	}

	for _, bad := range []string{
		"",
		"sha256:",
		"md5:915532321e363e3cfb9e2f3e095ce53c68bbc525f59a8a0318b87412ff9bbc42",
		"sha256:zz5532321e363e3cfb9e2f3e095ce53c68bbc525f59a8a0318b87412ff9bbc42",
		"915532321e363e3cfb9e2f3e095ce53c68bbc525f59a8a0318b87412ff9bbc42",
	} {
		if isSHA256Digest(bad) {
			t.Errorf("digest %q accepted", bad)
		}
	}
}
