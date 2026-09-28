package app

import (
	"os"
	"path/filepath"
	"strings"
)

const outputDirEnv = "GOWEEB_OUTPUT_DIR"

// DefaultTUIScanDir returns the default output directory used by the TUI.
func DefaultTUIScanDir() string {
	if forcedDir, ok := ForcedOutputDir(); ok {
		return forcedDir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, "Documents")
}

// DefaultCLIScanDir returns the default output directory used by the CLI.
func DefaultCLIScanDir() string {
	if forcedDir, ok := ForcedOutputDir(); ok {
		return forcedDir
	}

	return "scan"
}

// ForcedOutputDir returns an output directory imposed by the environment.
//
// This is primarily used by the official container image so generated files
// are always written to the mounted persistent directory.
func ForcedOutputDir() (string, bool) {
	value := strings.TrimSpace(
		os.Getenv(outputDirEnv),
	)

	if value == "" {
		return "", false
	}

	return value, true
}

// ResolveOutputDir applies an environment-enforced output directory when one
// exists. Otherwise, the requested directory is preserved.
func ResolveOutputDir(requested string) string {
	if forcedDir, ok := ForcedOutputDir(); ok {
		return forcedDir
	}

	return requested
}
