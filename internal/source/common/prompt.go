package common

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

// Displays a list and asks user to pick one item.
// Returns selected Value, empty string if cancelled.
func PromptUserToSelect(items []SelectableItem, title string, prompt string, log *logger.Logger) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no results to choose from")
	}

	if len(items) == 1 {
		log.Debug("Auto-selecting: %s\n", items[0].Value)
		return items[0].Value, nil
	}

	log.Info("\n%s\n", title)
	for i, item := range items {
		log.Info("[%d] - %s\n", i+1, item.Label)
	}
	log.Info("")

	reader := bufio.NewReader(os.Stdin)
	for {
		log.Info(prompt, len(items))

		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err != nil {
			log.Warn("Invalid input. Please enter a number.\n")
			continue
		}

		if choice == 0 {
			log.Debug("Selection cancelled by user\n")
			return "", nil
		}

		if choice < 1 || choice > len(items) {
			log.Warn("Invalid choice. Please enter a number between 1 and %d.\n", len(items))
			continue
		}

		selected := items[choice-1]
		log.Info("Selected: %s\n", selected.Label)
		log.Debug("Value: %s\n", selected.Value)
		return selected.Value, nil
	}
}
