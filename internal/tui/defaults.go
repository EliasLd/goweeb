package tui

import (
	"os"
	"path/filepath"
)

// Returns the default directory for downloaded manga.
func getDefaultScanDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, "Documents")
}
