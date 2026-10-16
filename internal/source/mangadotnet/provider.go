package mangadotnet

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	mangadotnetscraper "github.com/EliasLd/goweeb/internal/source/mangadotnet/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://mangadot.net"

type Provider struct {
	domain string
}

func New(customDomain string) *Provider {
	domain := strings.TrimSpace(customDomain)

	if domain == "" {
		domain = defaultDomain
	}

	return &Provider{
		domain: strings.TrimSuffix(domain, "/"),
	}
}

func (p *Provider) Name() string {
	return "mangadotnet"
}

func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := mangadotnetscraper.SearchCatalog(
		p.domain,
		query,
		log,
	)
	if err != nil {
		return nil, err
	}

	out := make([]sourcetypes.SearchResult, 0, len(results))

	for _, result := range results {
		out = append(out, sourcetypes.SearchResult{
			Title: result.Title,
			URL:   p.mangaURL(result.ID),
		})
	}

	return out, nil
}

// mangaURL builds the canonical manga page URL used as work URL by the
// following provider steps: {domain}/manga/{mangaID}.
func (p *Provider) mangaURL(mangaID string) string {
	return p.domain + "/manga/" + mangaID
}

// The remaining Provider methods are temporary stubs. Scan selection,
// chapter listing and page retrieval are added in the following
// implementation steps.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	return nil, fmt.Errorf(
		"mangadot.net scan selection is not implemented yet",
	)
}

func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	return sourcetypes.Work{}, nil, fmt.Errorf(
		"mangadot.net chapter listing is not implemented yet",
	)
}

func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	return nil, fmt.Errorf(
		"mangadot.net page image retrieval is not implemented yet",
	)
}
