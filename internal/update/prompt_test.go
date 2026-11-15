package update

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/source/common"
	"github.com/EliasLd/goweeb/internal/version"
)

func stubVersion(t *testing.T, tag string) {
	t.Helper()

	original := version.Version

	version.Version = tag

	t.Cleanup(func() { version.Version = original })
}

func stubStdin(t *testing.T, answer string) {
	t.Helper()

	original := common.StdinReader

	common.StdinReader = bufio.NewReader(strings.NewReader(answer))

	t.Cleanup(func() { common.StdinReader = original })
}

func TestWantsUpdate(t *testing.T) {
	tests := []struct {
		answer string
		want   bool
	}{
		{"y", true},
		{"Y", true},
		{"yes\n", true},
		{" YES \n", true},
		{"n", false},
		{"no", false},
		{"", false},
		{"\n", false},
		{"  ", false},
		{"update", false},
	}

	for _, test := range tests {
		if got := wantsUpdate(test.answer); got != test.want {
			t.Errorf(
				"wantsUpdate(%q) = %v, want %v",
				test.answer,
				got,
				test.want,
			)
		}
	}
}

func TestOfferSkipsDevelopBuilds(t *testing.T) {
	// No version injected and no server reachable: a develop
	// build must not even try.
	stubVersion(t, "develop")

	original := latestReleaseURL

	latestReleaseURL = "http://127.0.0.1:1/latest"

	t.Cleanup(func() { latestReleaseURL = original })

	if err := Offer("develop", FlavorCLI, nil); err != nil {
		t.Errorf("Offer() error: %v", err)
	}
}

func TestOfferDeclinesThePrompt(t *testing.T) {
	dir := t.TempDir()

	target := stubExecutable(t, dir)

	stubVersion(t, "v4.6.3")
	newReleaseServer(t, releaseFixture("v4.7.0"))
	stubStdin(t, "n\n")

	if err := Offer("v4.6.3", FlavorCLI, nil); err != nil {
		t.Fatalf("Offer() error: %v", err)
	}

	// Declining leaves the running binary untouched.
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("the running binary was touched: %v", err)
	}
}

func TestOfferInstallsWhenConfirmed(t *testing.T) {
	dir := t.TempDir()

	target := stubExecutable(t, dir)

	if err := os.WriteFile(target, []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	archive, digest := newBinaryArchive(t, []struct{ name, body string }{
		{"goweeb", "NEW BINARY"},
	}, false)

	url := serve(t, archive)

	original := latestReleaseURL

	latestReleaseURL = url

	t.Cleanup(func() { latestReleaseURL = original })

	// The release lookup only needs the tag: the URL the offer
	// applies carries the archive.
	newReleaseServer(t, `{"tag_name":"v4.7.0","prerelease":false,"assets":[{"name":"`+
		archiveName(FlavorCLI, "v4.7.0", "linux", "amd64")+`","browser_download_url":"`+url+`","digest":"`+digest+`"}]}`)

	stubVersion(t, "v4.6.3")
	stubStdin(t, "y\n")

	if err := Offer("v4.6.3", FlavorCLI, nil); err != nil {
		t.Fatalf("Offer() error: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "NEW BINARY" {
		t.Errorf("the confirmed update did not install: %q", content)
	}
}
