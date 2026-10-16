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

// MangaDot exposes one chapter list per upload group. Group filter
// options are either numeric upload ids or "scanlator:" pseudo-ids and
// are forwarded verbatim to the chapter listing step.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	mangaID, err := mangadotnetscraper.MangaIDFromURL(workURL)
	if err != nil {
		return nil, err
	}

	scanPaths, err := mangadotnetscraper.ListScanPaths(
		p.domain,
		mangaID,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get mangadot.net scan paths: %w",
			err,
		)
	}

	items := make([]common.SelectableItem, 0, len(scanPaths))

	for _, scanPath := range scanPaths {
		items = append(items, common.SelectableItem{
			Label: scanPath.Label,
			Value: scanPath.Value,
		})
	}

	return items, nil
}

// The remaining Provider methods are temporary stubs. Chapter listing
// and page retrieval are added in the following implementation steps.

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
