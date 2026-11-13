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
