package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func PromptChapterRange(
	entries []sourcetypes.Entry,
	log *logger.Logger,
) (RangeSelection, error) {
	var selection RangeSelection

	reader := bufio.NewReader(os.Stdin)

	log.Info(
		"Found %d chapter(s).\n",
		len(entries),
	)

	log.Info(
		"Available chapters: %s\n",
		FormatAvailableRanges(entries),
	)

	log.Info("Select chapters to download:\n")
	log.Info("  - all           : all available chapters\n")
	log.Info("  - 10            : only chapter 10\n")
	log.Info("  - 1-10          : chapters 1 to 10\n")
	log.Info("  - 10-           : chapter 10 onwards\n")
	log.Info("  - -10           : last 10 available chapters\n")
	log.Info("  - 1-10,20-30    : multiple ranges\n")

	for {
		log.Info("Enter range: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return selection, fmt.Errorf(
				"failed to read input: %w",
				err,
			)
		}

		input = strings.TrimSpace(input)

		if input == "" {
			log.Warn(
				"Empty input. Please enter a valid range.\n",
			)
			continue
		}

		parsed, err := ParseRangeExpression(input)
		if err != nil {
			log.Warn("%v\n", err)
			continue
		}

		matched := FilterEntriesBySelection(
			entries,
			parsed,
		)

		if len(matched) == 0 {
			log.Warn(
				"No available chapters match this selection.\n",
			)

			continue
		}

		return parsed, nil
	}
}
