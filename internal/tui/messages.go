package tui

import (
	"io"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/EliasLd/goweeb/internal/update"
)

// A line received from the download logger.
type logMsg string

// Initializes the log pipe used during downloads.
type setupLogPipeMsg struct {
	reader *io.PipeReader
}

type interactiveSearchDebounceMsg struct {
	query      string
	generation uint64
}

type interactiveSearchResultMsg struct {
	query      string
	generation uint64

	// Provider that produced these results; the fan-out search
	// sends one message per provider, even in single-provider
	// mode, so results carry their origin.
	provider      string
	providerLabel string

	results []sourcetypes.SearchResult
	err     error
}

type scanPathResultMsg struct {
	paths []common.SelectableItem
	err   error
}

type entriesResultMsg struct {
	entries []sourcetypes.Entry
	err     error
}

// The volume listing a provider carries for the current work; one
// message per scan-path pick, next to the entries listing.
type volumesResultMsg struct {
	volumes []sourcetypes.Volume
	err     error
}

// The startup update check outcome: an offer, nothing, or the
// failure of the check itself (silently dropped).
type updateCheckResultMsg struct {
	release *update.Release
	err     error
}

// The update installation has finished: success or failure for the
// alert layer to show.
type updateApplyDoneMsg struct {
	err error
}
