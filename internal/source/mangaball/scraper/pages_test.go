package mangaballscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// chapterPageHTML reproduces the real chapter page's data shape: an
// escaped "pages" JSON array embedded inside a script payload.
func chapterPageHTML(urls []string) string {
	escaped := make([]string, 0, len(urls))

	for _, u := range urls {
		escaped = append(
			escaped,
			fmt.Sprintf(`\"https://cdn.example/%s\"`, u),
		)
	}

	return `<html><head><title>MangaBall</title></head><body>` +
		`<img src="https://cdn.example/" alt="placeholder">` +
		`<script>self.__next_f.push([1,"$L12:[1,\"$\",\"$L19\",null,` +
		`{\"chapter\":{\"id\":\"685d80de\"},` +
		`\"pages\":[` + strings.Join(escaped, ",") + `]}])</script>` +
		`</body></html>`
}

func serveChapterPage(
	t *testing.T,
	bodies map[string]string,
) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc(
		"/chapter-detail/",
		func(w http.ResponseWriter, r *http.Request) {
			name := strings.TrimPrefix(r.URL.Path, "/chapter-detail/")

			body, ok := bodies[name]
			if !ok {
				http.NotFound(w, r)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, body)
		},
	)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func TestChapterIDsFromURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    []string
		wantErr bool
	}{
		{
			name: "canonical single-source chapter",
			url:  "https://mangaball.com/chapter-detail/685d80ded41ee75a90bbadd1",
			want: []string{"685d80ded41ee75a90bbadd1"},
		},
		{
			name: "multi-source entry with joined ids",
			url: "https://mangaball.com/chapter-detail/" +
				"685d80ded41ee75a90bbadd1,6a2cb5f4b617755de80cde3b",
			want: []string{
				"685d80ded41ee75a90bbadd1",
				"6a2cb5f4b617755de80cde3b",
			},
		},
		{
			name: "trailing slash",
			url:  "https://mangaball.com/chapter-detail/685d80ded41ee75a90bbadd1/",
			want: []string{"685d80ded41ee75a90bbadd1"},
		},
		{
			name:    "wrong first segment",
			url:     "https://mangaball.com/title-detail/685d80ded41ee75a90bbadd1",
			wantErr: true,
		},
		{
			name:    "no ids",
			url:     "https://mangaball.com/chapter-detail/,",
			wantErr: true,
		},
		{
			name:    "garbage url",
			url:     "://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ChapterIDsFromURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("ChapterIDsFromURL(%q) = %v, want error", tt.url, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("ChapterIDsFromURL(%q): %v", tt.url, err)
			}

			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("ChapterIDsFromURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestGetPageImageURLsKeepsPageOrder(
	t *testing.T,
) {
	server := serveChapterPage(t, map[string]string{
		"chapter-1": chapterPageHTML([]string{
			"685d/001.webp",
			"685d/002.webp",
			"685d/003.webp",
		}),
	})

	urls, err := GetPageImageURLs(
		server.URL+"/chapter-detail/chapter-1",
		nil,
	)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("urls = %d, want 3", len(urls))
	}

	// First page first, absolute URLs preserved.
	if urls[0] != "https://cdn.example/685d/001.webp" {
		t.Errorf("first page = %q", urls[0])
	}
}

func TestGetPageImageURLsPicksTheFullestCopy(
	t *testing.T,
) {
	// The comick copy is the first source but incomplete (2
	// pages); the mangadex copy is fuller (3 pages).
	server := serveChapterPage(t, map[string]string{
		"copy-comick": chapterPageHTML([]string{
			"a/001.webp",
			"a/002.webp",
		}),
		"copy-mangadex": chapterPageHTML([]string{
			"b/001.webp",
			"b/002.webp",
			"b/003.webp",
		}),
	})

	urls, err := GetPageImageURLs(
		server.URL+"/chapter-detail/copy-comick,copy-mangadex",
		nil,
	)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("urls = %d, want the 3-page copy", len(urls))
	}

	if strings.Contains(urls[0], "/a/") {
		t.Errorf("first page = %q, want a page of the fullest copy", urls[0])
	}
}

func TestGetPageImageURLsTieKeepsFirstCopy(
	t *testing.T,
) {
	// Equal page counts keep the first copy (the site's own
	// ordering) instead of gambling on later sources.
	server := serveChapterPage(t, map[string]string{
		"first": chapterPageHTML([]string{
			"first/001.webp",
			"first/002.webp",
		}),
		"second": chapterPageHTML([]string{
			"second/001.webp",
			"second/002.webp",
		}),
	})

	urls, err := GetPageImageURLs(
		server.URL+"/chapter-detail/first,second",
		nil,
	)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}

	if len(urls) != 2 {
		t.Fatalf("urls = %d, want 2", len(urls))
	}

	if !strings.Contains(urls[0], "/first/") {
		t.Errorf("first page = %q, want a page of the first copy", urls[0])
	}
}

func TestGetPageImageURLsToleratesBrokenCopies(
	t *testing.T,
) {
	// One source may be gone (removed/404); the entry still
	// resolves through the remaining ones.
	server := serveChapterPage(t, map[string]string{
		"gone":    "",
		"healthy": chapterPageHTML([]string{"h/001.webp"}),
	})

	urls, err := GetPageImageURLs(
		server.URL+"/chapter-detail/gone,healthy",
		nil,
	)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}

	if len(urls) != 1 {
		t.Fatalf("urls = %d, want 1", len(urls))
	}
}

func TestGetPageImageURLsAllCopiesBroken(
	t *testing.T,
) {
	server := serveChapterPage(t, map[string]string{})

	_, err := GetPageImageURLs(
		server.URL+"/chapter-detail/all-gone",
		nil,
	)

	if err == nil {
		t.Fatal("expected an error when no copy is usable")
	}
}

func TestGetPageImageURLsSkipsForeignSchemes(
	t *testing.T,
) {
	server := serveChapterPage(t, map[string]string{
		"weird": `self.__next_f.push([1,"{\"pages\":[\"https://cdn.example/001.webp\",\"javascript:evil()\"]}"])`,
	})

	urls, err := GetPageImageURLs(
		server.URL+"/chapter-detail/weird",
		nil,
	)
	if err != nil {
		t.Fatalf("pages: %v", err)
	}

	if len(urls) != 1 || !strings.HasPrefix(urls[0], "http") {
		t.Fatalf("urls = %v, want only the http page", urls)
	}
}

func TestGetPageImageURLsNoPageList(
	t *testing.T,
) {
	server := serveChapterPage(t, map[string]string{
		"empty": "200 <html><body>no payload</body></html>",
	})

	_, err := GetPageImageURLs(
		server.URL+"/chapter-detail/empty",
		nil,
	)

	if err == nil || !strings.Contains(err.Error(), "no page list") {
		t.Fatalf("error = %v, want the missing page-list failure", err)
	}
}
