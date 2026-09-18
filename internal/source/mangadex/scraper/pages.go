package mangadexscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
)

const mangaDexImageQuality = "data"

type atHomeResponse struct {
	Result  string        `json:"result"`
	BaseURL string        `json:"baseUrl"`
	Chapter atHomeChapter `json:"chapter"`
}

type atHomeChapter struct {
	Hash      string   `json:"hash"`
	Data      []string `json:"data"`
	DataSaver []string `json:"dataSaver"`
}

func GetPageImageURLs(
	domain string,
	entryURL string,
	log *logger.Logger,
) ([]string, error) {
	chapterID, err := extractChapterID(entryURL)
	if err != nil {
		return nil, err
	}

	baseAPIURL, err := url.Parse(
		strings.TrimSuffix(domain, "/"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid MangaDex domain: %w",
			err,
		)
	}

	atHomeRef, err := url.Parse(
		fmt.Sprintf(
			"/at-home/server/%s",
			url.PathEscape(chapterID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to build MangaDex At-Home URL: %w",
			err,
		)
	}

	atHomeURL := baseAPIURL.ResolveReference(
		atHomeRef,
	).String()

	log.Debug(
		"Fetching MangaDex At-Home server: %s\n",
		atHomeURL,
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		atHomeURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaDex At-Home request: %w",
			err,
		)
	}

	req.Header.Set(
		"Accept",
		"application/json",
	)

	req.Header.Set(
		"User-Agent",
		"goweeb",
	)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaDex At-Home server: %w",
			err,
		)
	}
	defer resp.Body.Close()

	log.Debug(
		"MangaDex At-Home response status: %d\n",
		resp.StatusCode,
	)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"MangaDex At-Home returned status: %d",
			resp.StatusCode,
		)
	}

	var response atHomeResponse

	if err := json.NewDecoder(resp.Body).Decode(
		&response,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode MangaDex At-Home response: %w",
			err,
		)
	}

	if response.Result != "ok" {
		return nil, fmt.Errorf(
			"MangaDex At-Home returned unexpected result: %s",
			response.Result,
		)
	}

	baseURL := strings.TrimSpace(
		response.BaseURL,
	)

	chapterHash := strings.TrimSpace(
		response.Chapter.Hash,
	)

	if baseURL == "" {
		return nil, fmt.Errorf(
			"MangaDex At-Home returned an empty base URL",
		)
	}

	if chapterHash == "" {
		return nil, fmt.Errorf(
			"MangaDex At-Home returned an empty chapter hash",
		)
	}

	files := response.Chapter.Data

	if len(files) == 0 {
		return nil, fmt.Errorf(
			"MangaDex At-Home returned no pages for chapter %s",
			chapterID,
		)
	}

	// MangaDex explicitly documents baseUrl as an opaque string.
	// Do not parse it or make assumptions about its host/path.
	baseURL = strings.TrimRight(
		baseURL,
		"/",
	)

	imageURLs := make(
		[]string,
		0,
		len(files),
	)

	for _, filename := range files {
		filename = strings.TrimSpace(filename)

		if filename == "" {
			continue
		}

		imageURL := fmt.Sprintf(
			"%s/%s/%s/%s",
			baseURL,
			mangaDexImageQuality,
			url.PathEscape(chapterHash),
			url.PathEscape(filename),
		)

		imageURLs = append(
			imageURLs,
			imageURL,
		)
	}

	if len(imageURLs) == 0 {
		return nil, fmt.Errorf(
			"no valid MangaDex page URLs found for chapter %s",
			chapterID,
		)
	}

	log.Debug(
		"Found %d MangaDex page image(s)\n",
		len(imageURLs),
	)

	return imageURLs, nil
}

func extractChapterID(
	entryURL string,
) (string, error) {
	u, err := url.Parse(
		strings.TrimSpace(entryURL),
	)
	if err != nil {
		return "", fmt.Errorf(
			"invalid MangaDex chapter URL: %w",
			err,
		)
	}

	parts := strings.Split(
		strings.Trim(u.Path, "/"),
		"/",
	)

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "chapter" {
			continue
		}

		chapterID := strings.TrimSpace(
			parts[i+1],
		)

		if chapterID != "" {
			return chapterID, nil
		}
	}

	return "", fmt.Errorf(
		"unable to extract MangaDex chapter ID from URL: %s",
		entryURL,
	)
}
