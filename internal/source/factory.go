package source

import (
	"fmt"
	"strings"

	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

func New(name, customDomain string) (sourcetypes.Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))

	registration, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	return registration.New(customDomain), nil
}
