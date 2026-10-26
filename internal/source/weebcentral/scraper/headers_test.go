package weebcentralscraper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

// Records the headers of the last request and serves an HTML body
// shaped after the real WeebCentral responses.
type headerCapture struct {
	server   *httptest.Server
	lastBody func() http.Header
}

func serveWeebcentralPage(
	t *testing.T,
	body string,
) *headerCapture {
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

				w.Write([]byte(body))
			},
		),
	)

	t.Cleanup(server.Close)

	return &headerCapture{
		server: server,
		lastBody: func() http.Header {
			if len(*captured) == 0 {
				t.Fatal("no request reached the server")
			}

			return (*captured)[len(*captured)-1]
		},
	}
}

// assertBrowserProfile verifies a request carries one of the
// browser profiles exactly (the scraper must not degrade back to
// the bare User-Agent that bot protection learns to flag).
func assertBrowserProfile(
	t *testing.T,
	headers http.Header,
	want map[string]string,
) {
	t.Helper()

	for key, value := range want {
		if got := headers.Get(key); got != value {
			t.Errorf(
				"header %s = %q, want %q",
				key,
				got,
				value,
			)
		}
	}
}

func TestSearchCatalogSendsFetchProfile(
	t *testing.T,
) {
	capture := serveWeebcentralPage(
		t,
		"<html><body>"+
			"<article>"+
			"<a href=\"/series/01-test-title\">"+
			"<h2>Test Title</h2>"+
			"</a></article>"+
			"</body></html>",
	)

	items, err := SearchCatalog(
		capture.server.URL,
		"berserk",
		logger.New(io.Discard, logger.LevelInfo),
	)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}

	assertBrowserProfile(
		t,
		capture.lastBody(),
		weebcentralFetchHeaders,
	)
}

func TestGetPageImageURLsSendsNavigationProfile(
	t *testing.T,
) {
	capture := serveWeebcentralPage(
		t,
		"<html><body>"+
			"<section id=\"chapter-images\">"+
			"<img src=\"/img/001.jpg\">"+
			"</section>"+
			"</body></html>",
	)

	imageURLs, err := GetPageImageURLs(
		capture.server.URL+"/chapter/1",
		logger.New(io.Discard, logger.LevelInfo),
	)
	if err != nil {
		t.Fatalf("page image urls: %v", err)
	}

	if len(imageURLs) != 1 {
		t.Fatalf("image urls = %d, want 1", len(imageURLs))
	}

	assertBrowserProfile(
		t,
		capture.lastBody(),
		weebcentralNavigationHeaders,
	)
}
