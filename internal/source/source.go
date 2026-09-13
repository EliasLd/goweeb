package source

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EliasLd/scan-scraper/internal/source/animesama"
	"github.com/EliasLd/scan-scraper/internal/source/mangafreak"
	sourcetypes "github.com/EliasLd/scan-scraper/internal/source/types"
)

type ProviderInfo struct {
	ID    string
	Label string
}

type providerRegistration struct {
	Label string
	New   func(customDomain string) sourcetypes.Provider
}

var providers = map[string]providerRegistration{
	"animesama": {
		Label: "Anime-Sama (French)",
		New: func(customDomain string) sourcetypes.Provider {
			return animesama.New(customDomain)
		},
	},
	"mangafreak": {
		Label: "MangaFreak (English)",
		New: func(customDomain string) sourcetypes.Provider {
			return mangafreak.New(customDomain)
		},
	},
}

func NewSource(name, customDomain string) (sourcetypes.Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))

	registration, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	return registration.New(customDomain), nil
}

func AvailableProviders() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(providers))

	for id, provider := range providers {
		result = append(result, ProviderInfo{
			ID:    id,
			Label: provider.Label,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Label < result[j].Label
	})

	return result
}

func IsSupported(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))

	_, ok := providers[name]
	return ok
}
