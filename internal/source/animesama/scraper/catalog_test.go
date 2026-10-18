package animesamascraper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

// newTestLogger returns a logger discarding everything; the animesama
// scrapers are not nil-safe on their own logging calls.
func newTestLogger() *logger.Logger {
	return logger.New(io.Discard, logger.LevelInfo)
}

// The redesigned catalogue page serves its results as "as-carte"
// cards, mirroring https://anime-sama.to/catalogue/?search=.
const redesignedCatalogPage = `<!DOCTYPE html>
<html>
<body>
	<form>
		<div id="list_catalog" class="overflow-y-scroll">
			<div class="as-carte as-carte-oeuvre" data-url="/catalogue/bleach/" data-nom="Bleach ">
				<a class="as-carte-lien" href="/catalogue/bleach" draggable="false">
					<div class="as-carte-texte">
						<h3 class="as-carte-titre" title="Bleach ">Bleach </h3>
					</div>
				</a>
			</div>
			<div class="as-carte as-carte-oeuvre" data-url="/catalogue/blue-lock/" data-nom="Blue Lock  ">
				<a class="as-carte-lien" href="https://example.test/catalogue/blue-lock/">
					<div class="as-carte-texte">
						<h3 class="as-carte-titre" title="Blue Lock  ">L&#039;Atelier des Sorcières </h3>
					</div>
				</a>
			</div>
		</div>
	</form>
</body>
</html>`

// Mirrors that lag behind the redesign still serve catalog-card tags.
const legacyCatalogPage = `<!DOCTYPE html>
<html>
<body>
	<div class="catalog-card">
		<a href="/catalogue/one-piece/">
			<div class="card-title">One Piece </div>
		</a>
	</div>
</body>
</html>`

func TestSearchCatalogParsesRedesignedCards(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)

			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			_, _ = w.Write([]byte(redesignedCatalogPage))
		},
	))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "bleach", newTestLogger())
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if got := requests.Load(); got != 1 {
		t.Errorf("sent %d request(s), want 1 (no legacy retry)", got)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d (%v)", len(results), results)
	}

	if results[0].Title != "Bleach" {
		t.Errorf("title = %q, want %q", results[0].Title, "Bleach")
	}

	// Root-relative hrefs resolve against the domain.
	wantURL := server.URL + "/catalogue/bleach"
	if results[0].URL != wantURL {
		t.Errorf("URL = %q, want %q", results[0].URL, wantURL)
	}

	// Absolute hrefs are kept as-is; the title text is unescaped
	// and trimmed.
	if results[1].Title != "L'Atelier des Sorcières" {
		t.Errorf(
			"title = %q, want %q",
			results[1].Title,
			"L'Atelier des Sorcières",
		)
	}

	if results[1].URL != "https://example.test/catalogue/blue-lock/" {
		t.Errorf(
			"URL = %q, want %q",
			results[1].URL,
			"https://example.test/catalogue/blue-lock/",
		)
	}
}

func TestSearchCatalogFallsBackToLegacyCards(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)

			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			_, _ = w.Write([]byte(legacyCatalogPage))
		},
	))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "one piece", newTestLogger())
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if got := requests.Load(); got != 2 {
		t.Fatalf("sent %d request(s), want 2 (legacy retry)", got)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d (%v)", len(results), results)
	}

	if results[0].Title != "One Piece" {
		t.Errorf("title = %q, want %q", results[0].Title, "One Piece")
	}

	wantURL := server.URL + "/catalogue/one-piece/"
	if results[0].URL != wantURL {
		t.Errorf("URL = %q, want %q", results[0].URL, wantURL)
	}
}

func TestSearchCatalogNoResults(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)

			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			_, _ = w.Write([]byte("<html><body></body></html>"))
		},
	))
	defer server.Close()

	results, err := SearchCatalog(server.URL, "nothing matches", newTestLogger())
	if err != nil {
		t.Fatalf("SearchCatalog() error: %v", err)
	}

	if got := requests.Load(); got != 2 {
		t.Errorf("sent %d request(s), want 2 (legacy retry)", got)
	}

	if results != nil {
		t.Errorf("results = %v, want nil", results)
	}
}
