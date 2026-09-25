package mangadexscraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/httpx"
	"github.com/EliasLd/goweeb/internal/logger"
)

type mangaDetailsResponse struct {
	Result string           `json:"result"`
	Data   mangaDetailsData `json:"data"`
}

type mangaDetailsData struct {
	ID         string                 `json:"id"`
	Attributes mangaDetailsAttributes `json:"attributes"`
}

type mangaDetailsAttributes struct {
	AvailableTranslatedLanguages []string `json:"availableTranslatedLanguages"`
}

func GetAvailableLanguages(
	workURL string,
	log *logger.Logger,
) ([]string, error) {
	workURL = strings.TrimSpace(workURL)

	if workURL == "" {
		return nil, fmt.Errorf("MangaDex manga URL is empty")
	}

	log.Debug(
		"Fetching MangaDex manga details: %s\n",
		workURL,
	)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		workURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create MangaDex manga details request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "goweeb")

	result, err := httpx.ReadAll(client, req, log)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch MangaDex manga details: %w",
			err,
		)
	}

	log.Debug(
		"MangaDex manga details response status: %d\n",
		result.StatusCode,
	)

	if result.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"MangaDex manga details returned status: %d",
			result.StatusCode,
		)
	}

	var response mangaDetailsResponse

	if err := json.Unmarshal(result.Body, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode MangaDex manga details: %w",
			err,
		)
	}

	if response.Result != "ok" {
		return nil, fmt.Errorf(
			"MangaDex returned unexpected result: %s",
			response.Result,
		)
	}

	languages := make([]string, 0)
	seen := make(map[string]struct{})

	for _, language := range response.Data.Attributes.AvailableTranslatedLanguages {
		language = strings.TrimSpace(language)

		if language == "" {
			continue
		}

		if _, exists := seen[language]; exists {
			continue
		}

		seen[language] = struct{}{}
		languages = append(languages, language)
	}

	if len(languages) == 0 {
		return nil, fmt.Errorf(
			"no translated languages available for MangaDex manga %s",
			response.Data.ID,
		)
	}

	log.Debug(
		"Found %d translated language(s)\n",
		len(languages),
	)

	return languages, nil
}
