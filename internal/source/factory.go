package source

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/source/animesama"
	"github.com/EliasLd/goweeb/internal/source/mangafreak"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
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
