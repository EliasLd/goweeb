package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/EliasLd/scan-scraper/internal/logger"
)

// Asks user to choose chapters after entries are discovered.
func PromptChapterRange(total int, log *logger.Logger) ([2]int, RangeMode, bool, error) {
	var chapterRange [2]int
	var rangeMode RangeMode = RangeNone
	all := false

	reader := bufio.NewReader(os.Stdin)

	log.Info("Found %d chapters.\n", total)
	log.Info("Select chapters to download:\n")
	log.Info("  - all  : all chapters\n")
	log.Info("  - 10   : only chapter 10\n")
	log.Info("  - 1-10 : chapters 1 to 10\n")
	log.Info("  - 10-  : chapter 10 to last\n")
	log.Info("  - -10  : last 10 chapters\n")

	for {
		log.Info("Enter range: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return chapterRange, rangeMode, all, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			log.Warn("Empty input. Please enter a valid range.\n")
			continue
		}

		if strings.EqualFold(input, "all") {
			all = true
			return chapterRange, rangeMode, all, nil
		}

		parsedRange, parsedMode, err := ParseRangeString(input)
		if err != nil {
			log.Warn("%v\n", err)
			continue
		}

		return parsedRange, parsedMode, all, nil
	}
}
