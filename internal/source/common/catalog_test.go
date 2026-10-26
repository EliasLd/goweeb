package common

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

// serveCatalogPage records the request headers of each hit and
// replies with a one-card catalog page, mirroring the real search
// endpoints shape.
func serveCatalogPage(
	t *testing.T,
) (*httptest.Server, func() http.Header) {
	t.Helper()

	captured := &[]http.Header{}

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				*captured = append(
					*captured,
					r.Header.Clone(),
				)

				w.Header().Set(
					"Content-Type",
					"text/html",
				)

				w.Write([]byte(
					"<html><body>" +
						"<div class=\"catalog-card\">" +
						"<a href=\"/series/test\">" +
						"<h2>Test Title</h2>" +
						"</a></div>" +
						"</body></html>",
				))
			},
		),
	)

	t.Cleanup(server.Close)

	return server, func() http.Header {
		if len(*captured) == 0 {
			t.Fatal("the catalog was never requested")
		}

		return (*captured)[len(*captured)-1]
	}
}

func searchTestConfig(server *httptest.Server) CatalogSearchConfig {
	return CatalogSearchConfig{
		Domain:        server.URL,
		EndpointPath:  "/search/data",
		QueryParam:    "text",
		CardSelector:  "div.catalog-card",
		LinkSelector:  "a",
		TitleSelector: "h2",
	}
}

func TestSearchHTMLCatalogDefaultUserAgent(
	t *testing.T,
) {
	server, lastHeaders := serveCatalogPage(t)

	items, err := SearchHTMLCatalog(
		searchTestConfig(server),
		"query",
		logger.New(io.Discard, logger.LevelInfo),
	)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	if got := lastHeaders().Get("User-Agent"); got != "Mozilla/5.0" {
		t.Errorf("user agent = %q, want the default browser one", got)
	}
}

func TestSearchHTMLCatalogSendsConfigHeaders(
	t *testing.T,
) {
	server, lastHeaders := serveCatalogPage(t)

	cfg := searchTestConfig(server)
	cfg.Headers = map[string]string{
		"User-Agent":     "goweeb-profile",
		"X-Goweeb":       "sentinel",
		"Sec-Fetch-Mode": "cors",
	}

	items, err := SearchHTMLCatalog(
		cfg,
		"query",
		logger.New(io.Discard, logger.LevelInfo),
	)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	headers := lastHeaders()

	if got := headers.Get("User-Agent"); got != "goweeb-profile" {
		t.Errorf(
			"user agent = %q, want the config one (config overrides the default)",
			got,
		)
	}

	if got := headers.Get("X-Goweeb"); got != "sentinel" {
		t.Errorf("header X-Goweeb = %q, want sentinel", got)
	}

	if got := headers.Get("Sec-Fetch-Mode"); got != "cors" {
		t.Errorf("header Sec-Fetch-Mode = %q, want cors", got)
	}
}
