package mangataroscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetTitleDetail(t *testing.T) {
	var fetchedPath string

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fetchedPath = r.URL.Path

			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(
				w,
				`<html><head><title>Fallback</title></head>`,
				`<body class="bg-neutral-900" data-manga-id="32148">`,
				`<h1 class="text-base xs:text-lg">Berserk</h1>`,
				`<p>ベルセルク</p>`,
				`</body></html>`,
			)
		},
	))
	defer server.Close()

	detail, err := GetTitleDetail(
		server.URL,
		"https://mangataro.org/manga/berserk",
		nil,
	)
	if err != nil {
		t.Fatalf("GetTitleDetail() error: %v", err)
	}

	if !strings.HasSuffix(fetchedPath, "/manga/berserk") {
		t.Errorf("title page fetched at %q, want the /manga/{slug} path", fetchedPath)
	}

	if detail.ID != "32148" {
		t.Errorf("id = %q, want 32148", detail.ID)
	}

	if detail.Name != "Berserk" {
		t.Errorf("name = %q, want Berserk", detail.Name)
	}
}

func TestGetTitleDetailWithoutMangaID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(
				w,
				`<html><body>`+
					`<h1>Mystery Manga</h1>`+
					`</body></html>`,
			)
		},
	))
	defer server.Close()

	_, err := GetTitleDetail(
		server.URL,
		"https://mangataro.org/manga/mystery",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for a page without the manga id")
	}
}

func TestGetTitleDetailWithoutName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(
				w,
				`<html><body data-manga-id="32148">`+
					`<p>only text</p>`+
					`</body></html>`,
			)
		},
	))
	defer server.Close()

	_, err := GetTitleDetail(
		server.URL,
		"https://mangataro.org/manga/no-title",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for a page without a name")
	}
}

func TestGetTitleDetailNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")

			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, "<html><title>Page not found</title></html>")
		},
	))
	defer server.Close()

	_, err := GetTitleDetail(
		server.URL,
		"https://mangataro.org/manga/does-not-exist",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for the 404")
	}

	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}

func TestGetTitleDetailInvalidWorkURL(t *testing.T) {
	_, err := GetTitleDetail(
		"https://mangataro.org",
		"https://mangataro.org/not-a-manga/berserk",
		nil,
	)
	if err == nil {
		t.Fatal("expected an error for an unexpected work URL")
	}
}

func TestGetTitleDetailMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")

			// The markers the page embeds, mirroring the real
			// Berserk page: three ld+json documents (only the
			// ComicSeries one carries the author), a full
			// description in paragraphs, one cover in the
			// OpenGraph metadata, and tag links rendered once
			// per layout.
			_, _ = fmt.Fprint(w,
				`<html><body data-manga-id="32148"><h1>Berserk</h1>`+
					`<script type="application/ld+json">{"@context":"https://schema.org","@type":"ComicSeries","name":"Berserk Manga | Read Online Free at MangaTaro","author":{"@type":"Person","name":"Miura, Kentarou, Studio Gaga"},"genre":"Manga","status":"Ongoing"}</script>`+
					`<script type="application/ld+json">{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[]}</script>`+
					`<script type="application/ld+json">{"@context":"https://schema.org","@type":"WebSite","url":"https://mangataro.org"}</script>`+
					`<meta property='og:image' content='https://mangataro.org/content/media/berserk.jpg'>`+
					`<div id="description-content-tab">`+
					`<p>First paragraph of the story.</p>`+
					`<p>Second paragraph, with an entity: &#039;quote&#039;.</p>`+
					`</div>`+
					`<a href="https://mangataro.org/tag/action"> Action </a>`+
					`<a href="https://mangataro.org/tag/drama">Drama</a>`+
					`<a href="https://mangataro.org/tag/action">Action</a>`+
					`</body></html>`)
		},
	))
	defer server.Close()

	detail, err := GetTitleDetail(
		server.URL,
		"https://mangataro.org/manga/berserk",
		nil,
	)
	if err != nil {
		t.Fatalf("GetTitleDetail() error: %v", err)
	}

	if detail.Metadata.Author != "Miura, Kentarou, Studio Gaga" {
		t.Errorf("author = %q, want the ComicSeries author line", detail.Metadata.Author)
	}

	if detail.Metadata.CoverURL != "https://mangataro.org/content/media/berserk.jpg" {
		t.Errorf("cover = %q, want the og:image content", detail.Metadata.CoverURL)
	}

	expectedDescription := "First paragraph of the story.\n" +
		"Second paragraph, with an entity: 'quote'."

	if detail.Metadata.Description != expectedDescription {
		t.Errorf("description = %q, want the joined paragraphs", detail.Metadata.Description)
	}

	// Duplicated layout renders deduplicate; the first order wins.
	if len(detail.Metadata.Tags) != 2 ||
		detail.Metadata.Tags[0] != "Action" ||
		detail.Metadata.Tags[1] != "Drama" {
		t.Errorf("tags = %v, want the deduplicated page tags", detail.Metadata.Tags)
	}

	if !detail.Metadata.Known() {
		t.Error("expected the metadata to be known")
	}
}

func TestGetTitleDetailMetadataDegradesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")

			// No ld+json, no description container, no tags.
			_, _ = fmt.Fprint(w,
				`<html><body data-manga-id="32148"><h1>Anonymous Manga</h1></body></html>`)
		},
	))
	defer server.Close()

	detail, err := GetTitleDetail(
		server.URL,
		"https://mangataro.org/manga/anonymous",
		nil,
	)
	if err != nil {
		t.Fatalf("GetTitleDetail() error: %v", err)
	}

	if detail.Metadata.Known() {
		t.Errorf("expected unknown metadata, got %+v", detail.Metadata)
	}
}
