package source

import (
	"fmt"
	"strings"

	"github.com/EliasLd/scan-scraper/internal/source/animesama"
	"github.com/EliasLd/scan-scraper/internal/source/mangafreak"
	sourcetypes "github.com/EliasLd/scan-scraper/internal/source/types"
)

func New(name, customDomain string) (sourcetypes.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "animesama":
		return animesama.New(customDomain), nil
	case "mangafreak":
		return mangafreak.New(customDomain), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
