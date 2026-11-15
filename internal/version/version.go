// Package version reports the version the running goweeb was built
// as, and compares release tags. The release build injects the tag
// with -ldflags "-X github.com/EliasLd/goweeb/internal/version.Version";
// every other build falls back to the build information or reports
// itself as a develop build.
package version

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

// Version is the running goweeb version, "develop" when the build
// carried no injection.

// readBuildInfo mirrors debug.ReadBuildInfo and is swappable in
// tests.
var readBuildInfo = debug.ReadBuildInfo
var Version = "develop"

// String returns the version of the running goweeb: the build-time
// injected tag, else the version recorded in the build information —
// `go install module@vX.Y.Z` stamps a clean release version there —
// else "develop".
func String() string {
	if Version != "develop" {
		return Version
	}

	info, ok := readBuildInfo()
	if ok && isCleanRelease(info.Main.Version) {
		return info.Main.Version
	}

	return "develop"
}

// isCleanRelease reports whether the text is a plain semantic
// version. The build information of a locally built binary carries a
// VCS pseudo-version (commit, timestamp, dirty suffixes): that is
// not the version of any release, and updating over such a build
// would replace local work with an older release.
func isCleanRelease(text string) bool {
	body := strings.TrimPrefix(strings.TrimSpace(text), "v")

	if body == "" {
		return false
	}

	for _, r := range body {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}

	_, err := parse(body)

	return err == nil
}

// IsDevelop reports whether this build carries no release version.
// Those builds skip the update check: updating them would replace a
// locally built binary with a release binary.
func IsDevelop() bool {
	return String() == "develop"
}

// IsNewer reports whether the candidate release tag is strictly
// newer than the current version tag. A version either side that
// cannot be parsed reports false: nothing is offered based on an
// unreadable comparison.
func IsNewer(candidate, current string) bool {
	upstream, err := parse(candidate)
	if err != nil {
		return false
	}

	running, err := parse(current)
	if err != nil {
		return false
	}

	return upstream.MoreThan(running)
}

// semver is the numeric parts of a release tag.
type semver struct {
	major, minor, patch int
}

// parse reads a "vMAJOR.MINOR.PATCH" tag; an optional build suffix
// ("-rc1") is tolerated and ignored, a two-part version is read as
// ".0".
func parse(tag string) (semver, error) {
	tag = strings.TrimPrefix(strings.TrimSpace(tag), "v")

	if cut := strings.IndexAny(tag, "-+"); cut >= 0 {
		tag = tag[:cut]
	}

	if tag == "" {
		return semver{}, fmt.Errorf("empty version")
	}

	parts := strings.Split(tag, ".")
	if len(parts) > 3 {
		return semver{}, fmt.Errorf("too many version parts in %q", tag)
	}

	out := semver{}
	fields := [...]*int{&out.major, &out.minor, &out.patch}

	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return semver{}, fmt.Errorf("unreadable version part %q in %q", part, tag)
		}

		*fields[i] = number
	}

	return out, nil
}

// MoreThan compares on major, then minor, then patch parts.
func (s semver) MoreThan(other semver) bool {
	switch {
	case s.major != other.major:
		return s.major > other.major

	case s.minor != other.minor:
		return s.minor > other.minor

	default:
		return s.patch > other.patch
	}
}
