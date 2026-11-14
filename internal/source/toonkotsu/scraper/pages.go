package toonkotsuscraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

// readerPath is the reader route goweeb uses; robots.txt allows it
// explicitly (Allow: /api/manga/chapter/ beside a blanket /api/ ban).
const readerPath = "/api/manga/chapter/"

// pageRow is one page of a reader payload: the blob path plus the
// AES-CTR key and nonce, base64url without padding.
type pageRow struct {
	Src string `json:"src"`
	K   string `json:"k"`
	N   string `json:"n"`
}

type readerResponse struct {
	Chapter struct {
		Pages []pageRow `json:"pages"`
	} `json:"chapter"`
}

// ChapterIDsFromURL extracts the chapter ids a chapter entry URL
// carries.
//
// Expected shape: {domain}/manga/chapter/{id}[,{id}…] — a single id
// when the site lists one upload, comma-joined ids when a number
// merged several uploads (see chapterEntryURL).
func ChapterIDsFromURL(entryURL string) ([]string, error) {
	entryURL = strings.TrimSpace(entryURL)

	parsed, err := url.Parse(entryURL)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid toonkotsu chapter URL %q: %w",
			entryURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")

	// Expected: ["", "manga", "chapter", "{ids}"]
	segments := strings.Split(path, "/")

	if len(segments) != 4 ||
		segments[1] != "manga" ||
		segments[2] != "chapter" {
		return nil, fmt.Errorf(
			"unexpected toonkotsu chapter URL: %q",
			entryURL,
		)
	}

	ids := make([]string, 0)

	for _, token := range strings.Split(segments[3], ",") {
		id := strings.TrimSpace(token)

		if id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf(
			"unexpected toonkotsu chapter URL: %q",
			entryURL,
		)
	}

	return ids, nil
}

// GetPageImageURLs resolves the ordered, decrypt-ready page URLs of
// one chapter entry.
//
// Every candidate id names one upload of the chapter; each is read
// through the reader API and the upload with the most pages wins —
// the site re-hosts crawls of the same chapter that differ in
// completeness, and a listing's page counts can go stale. Copies
// tying on count keep the first, mirroring the site's own ordering.
// A candidate failing to load or to parse simply loses; the error
// only surfaces when no candidate is usable at all.
func GetPageImageURLs(
	domain string,
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	ids, err := ChapterIDsFromURL(entryURL)
	if err != nil {
		return nil, err
	}

	var (
		bestPages []pageRow
		lastError error
	)

	for _, id := range ids {
		pages, err := fetchReaderPages(domain, id, log)
		if err != nil {
			lastError = err

			continue
		}

		if len(pages) > len(bestPages) {
			bestPages = pages
		}
	}

	if len(bestPages) == 0 {
		if lastError != nil {
			return nil, lastError
		}

		return nil, fmt.Errorf(
			"toonkotsu chapter %s has no page images",
			strings.Join(ids, ","),
		)
	}

	urls := make([]string, 0, len(bestPages))

	for _, page := range bestPages {
		src := strings.TrimSpace(page.Src)

		// A page without key material could only stage as
		// ciphertext, so it never enters the download list.
		if src == "" || page.K == "" || page.N == "" {
			if log != nil {
				log.Debug(
					"Dropping toonkotsu page without src or key material in chapter %s\n",
					strings.Join(ids, ","),
				)
			}

			continue
		}

		// The key and nonce travel in the fragment: the
		// downloader never sends fragments on the wire, and
		// DecodeImageData reads them back after the download.
		urls = append(
			urls,
			absoluteURL(domain, src)+
				"#k="+page.K+
				"&n="+page.N,
		)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf(
			"toonkotsu chapter %s has no decryptable page images",
			strings.Join(ids, ","),
		)
	}

	if log != nil {
		log.Debug(
			"Found %d toonkotsu page images\n",
			len(urls),
		)
	}

	return urls, nil
}

// fetchReaderPages loads one candidate upload's reader payload and
// returns its pages in order.
func fetchReaderPages(
	domain string,
	id string,
	log *logger.Logger,
) ([]pageRow, error) {
	body, err := callAPI(
		domain,
		"reader",
		readerPath+id,
		nil,
		log,
	)
	if err != nil {
		return nil, err
	}

	var response readerResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode toonkotsu chapter %s: %w",
			id,
			err,
		)
	}

	return response.Chapter.Pages, nil
}
