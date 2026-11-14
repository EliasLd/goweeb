package toonkotsuscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func readerRoute(pages string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[`+pages+`]}}`)
	}
}

func pageJSON(i int) string {
	return fmt.Sprintf(
		`{"src":"/cdn/e/p%d.bin","k":"k%d","n":"n%d","t":"image/jpeg","w":1067,"h":1600}`,
		i,
		i,
		i,
	)
}

func TestChapterIDsFromURL(t *testing.T) {
	cases := []struct {
		name     string
		entryURL string
		ids      []string
		wantErr  bool
	}{
		{
			name:     "single upload",
			entryURL: "https://toonkotsu.com/manga/chapter/abc",
			ids:      []string{"abc"},
		},
		{
			name:     "merged uploads",
			entryURL: "https://toonkotsu.com/manga/chapter/abc,def,ghi",
			ids:      []string{"abc", "def", "ghi"},
		},
		{
			name:     "trailing slash tolerated",
			entryURL: "https://toonkotsu.com/manga/chapter/abc/",
			ids:      []string{"abc"},
		},
		{
			name:     "fragment rides along",
			entryURL: "https://toonkotsu.com/manga/chapter/abc#k=x",
			ids:      []string{"abc"},
		},
		{
			name:     "wrong path",
			entryURL: "https://toonkotsu.com/chapter/abc",
			wantErr:  true,
		},
		{
			name:     "empty ids",
			entryURL: "https://toonkotsu.com/manga/chapter/ , ,",
			wantErr:  true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ids, err := ChapterIDsFromURL(testCase.entryURL)

			if testCase.wantErr {
				if err == nil {
					t.Fatalf(
						"ChapterIDsFromURL(%q) = %v, want error",
						testCase.entryURL,
						ids,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"ChapterIDsFromURL(%q) error: %v",
					testCase.entryURL,
					err,
				)
			}

			if len(ids) != len(testCase.ids) {
				t.Fatalf(
					"ids = %v, want %v",
					ids,
					testCase.ids,
				)
			}

			for i := range ids {
				if ids[i] != testCase.ids[i] {
					t.Errorf("ids[%d] = %q", i, ids[i])
				}
			}
		})
	}
}

func TestGetPageImageURLsFullestCopyWins(t *testing.T) {
	var requested []string

	mux := http.NewServeMux()

	mux.HandleFunc(readerPath+"thin", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		requested = append(requested, "thin")

		page := pageJSON(1)
		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[`+page+`]}}`)
	})
	mux.HandleFunc(readerPath+"full", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		requested = append(requested, "full")

		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[`+
			pageJSON(1)+","+pageJSON(2)+","+pageJSON(3)+`]}}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	urls, err := GetPageImageURLs(
		server.URL,
		server.URL+"/manga/chapter/thin,full",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	// Both candidates are read before the fullest copy wins.
	if len(requested) != 2 {
		t.Errorf("expected every candidate to be read, got %v", requested)
	}

	if len(urls) != 3 {
		t.Fatalf("expected the 3-page upload, got %d pages", len(urls))
	}

	// Blob URLs carry the key material in their fragment.
	if urls[0] != server.URL+"/cdn/e/p1.bin#k=k1&n=n1" {
		t.Errorf("unexpected page URL: %q", urls[0])
	}

	if !strings.HasSuffix(urls[2], "#k=k3&n=n3") {
		t.Errorf("unexpected last page URL: %q", urls[2])
	}
}

func TestGetPageImageURLsTieKeepsTheFirst(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc(readerPath+"first", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[`+pageJSON(1)+`,`+pageJSON(5)+`]}}`)
	})
	mux.HandleFunc(readerPath+"second", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[`+pageJSON(7)+`,`+pageJSON(8)+`]}}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	urls, err := GetPageImageURLs(
		server.URL,
		server.URL+"/manga/chapter/first,second",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	if len(urls) != 2 || !strings.Contains(urls[0], "p1.bin") {
		t.Errorf("expected the first upload to survive the tie, got %v", urls)
	}
}

func TestGetPageImageURLsDropsPagesWithoutKeys(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc(readerPath+"partial", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[`+
			`{"src":"/cdn/e/good.bin","k":"gk","n":"gn"},`+
			`{"src":"/cdn/e/nokey.bin"},`+
			`{"src":"","k":"k","n":"n"}]}}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	urls, err := GetPageImageURLs(
		server.URL,
		server.URL+"/manga/chapter/partial",
		testDiscardLogger(),
	)
	if err != nil {
		t.Fatalf("GetPageImageURLs() error: %v", err)
	}

	// A page without key material could only stage as ciphertext.
	if len(urls) != 1 || !strings.Contains(urls[0], "good.bin") {
		t.Errorf("expected only the keyed page, got %v", urls)
	}
}

func TestGetPageImageURLsNoDecryptablePages(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc(readerPath+"blank", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = fmt.Fprint(w, `{"chapter":{"pages":[{"src":"/cdn/e/x.bin"}]}}`)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	_, err := GetPageImageURLs(
		server.URL,
		server.URL+"/manga/chapter/blank",
		testDiscardLogger(),
	)
	if err == nil {
		t.Fatal("expected the keyless listing to be refused")
	}

	if !strings.Contains(err.Error(), "no decryptable page images") {
		t.Errorf("expected the decrypt refusal, got %v", err)
	}
}

func TestGetPageImageURLsNoUsableCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusGone)
		},
	))
	defer server.Close()

	_, err := GetPageImageURLs(
		server.URL,
		server.URL+"/manga/chapter/a,b",
		testDiscardLogger(),
	)
	if err == nil {
		t.Fatal("expected the candidates' failure to surface")
	}

	if !strings.Contains(err.Error(), "status 410") {
		t.Errorf("expected the status in the error, got %v", err)
	}
}
