package tui

import (
	"io"

	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
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
