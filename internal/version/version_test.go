package version

import (
	"runtime/debug"
	"testing"
)

func stubBuildInfo(t *testing.T, version string) {
	original := readBuildInfo

	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: version},
		}, true
	}

	t.Cleanup(func() { readBuildInfo = original })
}

func TestIsDevelopWithoutInjection(t *testing.T) {
	if !IsDevelop() {
		t.Errorf("a build without injection reported itself as %q", String())
	}
}

func TestIsDevelopOnAVCSPseudoVersion(t *testing.T) {
	// Local builds carry a VCS pseudo-version in the build
	// information; those are develop builds, never release
	// versions.
	stubBuildInfo(t, "v1.0.3-0.20261115113207-82708b95f860+dirty")

	if !IsDevelop() {
		t.Errorf("a pseudo-version build reported itself as %q", String())
	}
}

func TestIsDevelopOnAStaleModuleTag(t *testing.T) {
	stubBuildInfo(t, "v1.0.3")

	if IsDevelop() {
		t.Errorf("a go install build stamped %q hid its release version", String())
	}

	if got := String(); got != "v1.0.3" {
		t.Errorf("String() = %q, want v1.0.3", got)
	}
}

func TestStringWithInjectedVersion(t *testing.T) {
	original := Version

	defer func() { Version = original }()

	Version = "v4.7.0"

	if IsDevelop() {
		t.Errorf("an injected build reported itself as develop")
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		candidate, current string
		want               bool
	}{
		{"v4.7.0", "v4.6.3", true},
		{"v5.0.0", "v4.99.99", true},
		{"v4.6.4", "v4.6.3", true},
		{"v4.6.3", "v4.6.3", false},
		{"v4.6.3", "v4.7.0", false},
		{"v4.6.3", "develop", false},
		{"develop", "v4.6.3", false},
		{"", "v4.6.3", false},
		{"v4.7.0", "", false},
		{"vBROKEN", "v4.6.3", false},
	}

	for _, test := range tests {
		got := IsNewer(test.candidate, test.current)

		if got != test.want {
			t.Errorf(
				"IsNewer(%q, %q) = %v, want %v",
				test.candidate,
				test.current,
				got,
				test.want,
			)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		tag                 string
		major, minor, patch int
		expectError         bool
	}{
		{"v4.6.3", 4, 6, 3, false},
		{"4.6.3", 4, 6, 3, false},
		{"v4.6", 4, 6, 0, false},
		{"v5", 5, 0, 0, false},
		{"v2.1.0-beta1", 2, 1, 0, false},
		{" v4.6.3 ", 4, 6, 3, false},
		{"", 0, 0, 0, true},
		{"v4.x.3", 0, 0, 0, true},
		{"v4.6.3.1", 0, 0, 0, true},
		{"-v4.6.3", 0, 0, 0, true},
	}

	for _, test := range tests {
		got, err := parse(test.tag)

		if test.expectError {
			if err == nil {
				t.Errorf("parse(%q) = %v, expected an error", test.tag, got)
			}

			continue
		}

		if err != nil {
			t.Errorf("parse(%q) error: %v", test.tag, err)

			continue
		}

		if got.major != test.major ||
			got.minor != test.minor ||
			got.patch != test.patch {
			t.Errorf(
				"parse(%q) = %d.%d.%d, want %d.%d.%d",
				test.tag,
				got.major, got.minor, got.patch,
				test.major, test.minor, test.patch,
			)
		}
	}
}
