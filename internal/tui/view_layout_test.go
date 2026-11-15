package tui

import (
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/version"
)

func TestRenderFooterRowRidesTheVersionInTheLeftMargin(t *testing.T) {
	original := version.Version

	defer func() { version.Version = original }()

	version.Version = "v4.6.3"

	row := renderFooterRow(40, "helper text")

	if !strings.Contains(row, "v4.6.3") {
		t.Errorf("footer row missing the version: %q", row)
	}

	if !strings.Contains(row, "helper text") {
		t.Errorf("footer row missing the helper: %q", row)
	}

	// The helper must stay centered rather than shifted by the
	// label: it begins after the version and the leading blank
	// columns between them.
	helperStart := strings.Index(row, "helper text")

	if helperStart != 14 {
		t.Errorf("helper landed at column %d, want 14", helperStart)
	}
}

func TestRenderFooterRowSkipsTheLabelOnNarrowTerminals(t *testing.T) {
	row := renderFooterRow(12, "this helper text is long")

	// The label is dropped, the helper still shown (wrapped).
	if strings.Contains(row, version.String()) {
		t.Errorf("narrow terminal grew the footer: %q", row)
	}

	if !strings.Contains(row, "text is long") {
		t.Errorf("helper lost: %q", row)
	}
}
