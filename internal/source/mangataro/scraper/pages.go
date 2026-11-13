package mangataroscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
)

const chapterContentPath = "/auth/chapter-content"

// ChapterIDFromEntryURL extracts the chapter id an entry URL carries.
//
// Expected shape: {domain}/read/{slug}/ch{number}-{chapterID}. Manga
// numbers carry their own dots and dashes when written out (0.1 lists
// as ch0-1-32150), so the id is the token after the last dash.
func ChapterIDFromEntryURL(entryURL string) (string, error) {
	entryURL = strings.TrimSpace(entryURL)

	parsed, err := url.Parse(entryURL)
	if err != nil {
		return "", fmt.Errorf(
			"invalid MangaTaro chapter URL %q: %w",
			entryURL,
			err,
		)
	}

	path := strings.TrimSuffix(parsed.Path, "/")

	// Expected: ["", "read", "{slug}", "ch{number}-{id}"]
	segments := strings.Split(path, "/")

	if len(segments) != 4 || segments[1] != "read" {
		return "", fmt.Errorf(
			"unexpected MangaTaro chapter URL: %q",
			entryURL,
		)
	}

	id := segments[3]

	dash := strings.LastIndex(id, "-")
	if dash < 0 || dash == len(id)-1 {
		return "", fmt.Errorf(
			"unexpected MangaTaro chapter URL: %q",
			entryURL,
		)
	}

	token, err := url.PathUnescape(id[dash+1:])
	if err != nil {
		return "", fmt.Errorf(
			"unexpected MangaTaro chapter URL: %q",
			entryURL,
		)
	}

	token = strings.TrimSpace(token)

	if token == "" {
		return "", fmt.Errorf(
			"unexpected MangaTaro chapter URL: %q",
			entryURL,
		)
	}

	return token, nil
}

// chapterContentResponse mirrors the reader-content payload: page
// images in reading order, or the HTML body of a text chapter — the
// site also hosts novels, whose chapters carry no pages.
type chapterContentResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	ChapterType string   `json:"chapter_type"`
	Images      []string `json:"images"`
}

// GetPageImageURLs resolves the ordered page image URLs of a chapter
// entry through the reader's content route. Absolute http(s) URLs only
// are kept; a text chapter is an explicit, provider-branded error so
// a download can never mistake it for an empty one.
func GetPageImageURLs(
	domain string,
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	chapterID, err := ChapterIDFromEntryURL(entryURL)
	if err != nil {
		return nil, err
	}

	body, err := callAPI(
		domain,
		"chapter content",
		http.MethodGet,
		chapterContentPath,
		url.Values{"chapter_id": {chapterID}},
		nil,
		log,
	)
	if err != nil {
		return nil, err
	}

	var response chapterContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode MangaTaro chapter content: %w",
			err,
		)
	}

	if !response.Success {
		return nil, fmt.Errorf(
			"MangaTaro chapter content failed: %s",
			strings.TrimSpace(response.Message),
		)
	}

	if response.ChapterType == "text" {
		return nil, fmt.Errorf(
			"MangaTaro chapter %s is a text chapter and carries no page images",
			chapterID,
		)
	}

	imageURLs := make([]string, 0, len(response.Images))

	for _, raw := range response.Images {
		imageURL := strings.TrimSpace(raw)

		keep := strings.HasPrefix(imageURL, "http://") ||
			strings.HasPrefix(imageURL, "https://")

		if !keep {
			if log != nil {
				log.Debug(
					"Skipping non-http MangaTaro page image %q\n",
					imageURL,
				)
			}

			continue
		}

		imageURLs = append(imageURLs, imageURL)
	}

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf(
			"MangaTaro chapter %s has no page images",
			chapterID,
		)
	}

	if log != nil {
		log.Debug(
			"Found %d MangaTaro page images\n",
			len(imageURLs),
		)
	}

	return imageURLs, nil
}
