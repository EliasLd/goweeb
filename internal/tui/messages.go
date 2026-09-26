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

// Results returned by asynchronous provider operations.
type catalogSearchResultMsg struct {
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
