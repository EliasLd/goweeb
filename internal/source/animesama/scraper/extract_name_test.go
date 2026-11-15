package animesamascraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mirrors the current scan page: the work title moved to an h1 and
// carries the site's own trailing spaces, which are part of the key
// the chapter-list API matches on.
const scanPageWithH1 = `<!DOCTYPE html>
<html>
<body>
	<div class="ct-bloc">
		<h1 id="titreOeuvre" class="ct-titre">Naruto  </h1>
		<h2 id="avOeuvre" class="ct-saison"></h2>
	</div>
</body>
</html>`

const scanPageWithLegacyH3 = `<!DOCTYPE html>
<html>
<body>
	<h3 id="titreOeuvre">One Piece Couleur</h3>
</body>
</html>`

func TestExtractMangaNameKeepsTheHeadingVerbatim(t *testing.T) {
	// The trailing spaces are the site's own: the oœuvre API only
	// matches the heading text byte-for-byte, padding included.
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			fmt.Fprint(w, scanPageWithH1)
		}),
	)
	defer server.Close()

	// No trailing slash: the flow resolves the scan path into the
	// work URL, which the extractor must normalize itself.
	name, err := ExtractMangaName(
		server.URL+"/catalogue/naruto/scan/vf",
		newTestLogger(),
	)
	if err != nil {
		t.Fatalf("ExtractMangaName() error: %v", err)
	}

	if name != "Naruto  " {
		t.Errorf("name = %q, want %q", name, "Naruto  ")
	}
}

func TestExtractMangaNameMatchesTheLegacyMarkup(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			fmt.Fprint(w, scanPageWithLegacyH3)
		}),
	)
	defer server.Close()

	name, err := ExtractMangaName(
		server.URL+"/scan/vf/",
		newTestLogger(),
	)
	if err != nil {
		t.Fatalf("ExtractMangaName() error: %v", err)
	}

	if name != "One Piece Couleur" {
		t.Errorf("name = %q", name)
	}
}

func TestExtractMangaNameMissing(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			_ *http.Request,
		) {
			fmt.Fprint(w, "<html><body><h1>Nothing to see</h1></body></html>")
		}),
	)
	defer server.Close()

	_, err := ExtractMangaName(
		server.URL+"/scan/vf/",
		newTestLogger(),
	)
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.Contains(err.Error(), "could not extract manga name") {
		t.Errorf("expected the extraction error, got %v", err)
	}
}
