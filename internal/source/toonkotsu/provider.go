package toonkotsu

import (
	"fmt"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	toonkotsuscraper "github.com/EliasLd/goweeb/internal/source/toonkotsu/scraper"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

const defaultDomain = "https://toonkotsu.com"

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
	return "toonkotsu"
}

// Search walks the catalogue's browse route: relevance-ordered rows
// carrying the work URL and cover.
func (p *Provider) Search(
	query string,
	log *logger.Logger,
) ([]sourcetypes.SearchResult, error) {
	results, err := toonkotsuscraper.SearchCatalog(
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
			Title:    result.Title,
			URL:      result.URL,
			CoverURL: result.CoverURL,
		})
	}

	return out, nil
}

// Toonkotsu splits a work into its main series and its color
// editions; each listing is one selectable path, the chapter count in
// the label. The value is the edition code, empty for the main
// listing.
func (p *Provider) ListScanPaths(
	workURL string,
	log *logger.Logger,
) ([]common.SelectableItem, error) {
	items, err := toonkotsuscraper.GetScanPaths(
		p.domain,
		workURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get toonkotsu scan paths: %w",
			err,
		)
	}

	return items, nil
}

// Toonkotsu addresses the reader by chapter id; every upload of a
// number (the rows carrying it and their alternates) travels as
// comma-joined candidate ids. The scan path names an edition's
// listing when set.
func (p *Provider) ListEntries(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Entry, error) {
	code, err := toonkotsuscraper.WorkCodeFromURL(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	title, chapters, err := toonkotsuscraper.ListChapters(
		p.domain,
		code,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to list toonkotsu chapters: %w",
			err,
		)
	}

	entries := make([]sourcetypes.Entry, 0, len(chapters))

	for _, chapter := range chapters {
		entries = append(entries, sourcetypes.Entry{
			Number: chapter.Number,
			Label:  chapter.Label,
			URL:    chapter.URL,
		})
	}

	return sourcetypes.Work{
		Title: title,
		Kind:  sourcetypes.ItemChapter,
	}, entries, nil
}

// WorkMetadata reads the series payload the listing flow already
// relies on, deriving the metadata parts it publishes: authors,
// artists, tags, description, cover, year and original language.
func (p *Provider) WorkMetadata(workURL string,
	log *logger.Logger,
) (sourcetypes.Metadata, error) {
	code, err := toonkotsuscraper.WorkCodeFromURL(workURL)
	if err != nil {
		return sourcetypes.Metadata{}, err
	}

	metadata, err := toonkotsuscraper.GetWorkMetadata(
		p.domain,
		code,
		log,
	)
	if err != nil {
		return sourcetypes.Metadata{}, fmt.Errorf(
			"failed to get toonkotsu work metadata: %w",
			err,
		)
	}

	return metadata, nil
}

// Toonkotsu embeds its page list in a reader API robots.txt
// explicitly allows; every candidate upload is read and the fullest
// copy wins. The blobs themselves are AES-CTR ciphertexts whose key
// material rides in the page URL's fragment.
func (p *Provider) GetPageImageURLs(
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	urls, err := toonkotsuscraper.GetPageImageURLs(
		p.domain,
		entryURL,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get toonkotsu page images: %w",
			err,
		)
	}

	return urls, nil
}

// DecodeImageData decrypts the site's page blobs in place; only the
// encrypted blob pages carry key material, everything else passes
// through unchanged.
func (p *Provider) DecodeImageData(
	pageURL string,
	data []byte,
) ([]byte, bool) {
	return toonkotsuscraper.DecodeImageData(pageURL, data)
}

// ImageRequestHeaders carries what the site's hotlink rule demands:
// a bare "Mozilla/5.0" gets 403 on the encrypted page blobs, while a
// full browser user agent passes everywhere. No referer is needed.
func (p *Provider) ImageRequestHeaders() map[string]string {
	return map[string]string{
		"User-Agent": toonkotsuscraper.BrowserUserAgent,
	}
}

// Toonkotsu tags its chapters with the volume they belong to; volume
// mode groups them, untagged chapters staying chapter-mode only. The
// scan path names an edition's listing when set.
func (p *Provider) ListVolumes(
	workURL string,
	scanPath string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Volume, error) {
	code, err := toonkotsuscraper.WorkCodeFromURL(workURL)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	title, groups, err := toonkotsuscraper.ListVolumes(
		p.domain,
		code,
		scanPath,
		log,
	)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to list toonkotsu volumes: %w",
			err,
		)
	}

	volumes := make([]sourcetypes.Volume, 0, len(groups))

	for _, group := range groups {
		entries := make([]sourcetypes.Entry, 0, len(group.Chapters))

		for _, chapter := range group.Chapters {
			entries = append(entries, sourcetypes.Entry{
				Number: chapter.Number,
				Label:  chapter.Label,
				URL:    chapter.URL,
			})
		}

		volumes = append(volumes, sourcetypes.Volume{
			Number:  group.Number,
			Entries: entries,
		})
	}

	return sourcetypes.Work{
		Title: title,
		Kind:  sourcetypes.ItemVolume,
	}, volumes, nil
}

// Compile-time: the volume capability is wired.
var _ sourcetypes.VolumeLister = New("")
