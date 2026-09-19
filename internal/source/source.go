package source

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/source/animesama"
	"github.com/EliasLd/goweeb/internal/source/mangadex"
	"github.com/EliasLd/goweeb/internal/source/mangafreak"
	"github.com/EliasLd/goweeb/internal/source/mangavyvy"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
	"github.com/EliasLd/goweeb/internal/source/weebcentral"
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
	"weebcentral": {
		Label: "WeebCentral (English)",
		New: func(customDomain string) sourcetypes.Provider {
			return weebcentral.New(customDomain)
		},
	},
	"mangadex": {
		Label: "MangaDex (Multilingual)",
		New: func(customDomain string) sourcetypes.Provider {
			return mangadex.New(customDomain)
		},
	},
	"mangavyvy": {
		Label: "Mangavyvy (English)",
		New: func(customDomain string) sourcetypes.Provider {
			return mangavyvy.New(customDomain)
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
