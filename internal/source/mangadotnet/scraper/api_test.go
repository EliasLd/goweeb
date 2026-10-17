package mangadotnetscraper

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testAPIPath = "/api/test/search"

func TestParseBaseDomain(t *testing.T) {
	cases := []struct {
		name        string
		domain      string
		wantScheme  string
		wantHost    string
		wantPath    string
		wantQuery   string
		wantFrag    string
		wantErrText string
	}{
		{
			name:       "default domain",
			domain:     "https://mangadot.net",
			wantScheme: "https",
			wantHost:   "mangadot.net",
		},
		{
			name:       "custom domain with base path",
			domain:     "https://mirror.example.org/base",
			wantScheme: "https",
			wantHost:   "mirror.example.org",
			wantPath:   "/base",
		},
		{
			name:       "strips query and fragment",
			domain:     "https://mangadot.net/manga?x=1#frag",
			wantScheme: "https",
			wantHost:   "mangadot.net",
			wantPath:   "/manga",
			wantQuery:  "",
			wantFrag:   "",
		},
		{
			name:        "missing scheme",
			domain:      "mangadot.net",
			wantErrText: "invalid mangadot.net domain",
		},
		{
			name:        "empty domain",
			domain:      "  ",
			wantErrText: "invalid mangadot.net domain",
		},
		{
			name:        "invalid URL",
			domain:      "://not-a-url",
			wantErrText: "failed to parse mangadot.net domain",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			baseURL, err := parseBaseDomain(testCase.domain)

			if testCase.wantErrText != "" {
				if err == nil {
					t.Fatalf(
						"expected an error for domain %q",
						testCase.domain,
					)
				}

				if !strings.Contains(
					err.Error(),
					testCase.wantErrText,
				) {
					t.Errorf(
						"expected error to contain %q, got %v",
						testCase.wantErrText,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"parseBaseDomain(%q) error: %v",
					testCase.domain,
					err,
				)
			}

			if baseURL.Scheme != testCase.wantScheme {
				t.Errorf(
					"expected scheme %q, got %q",
					testCase.wantScheme,
					baseURL.Scheme,
				)
			}

			if baseURL.Host != testCase.wantHost {
				t.Errorf(
					"expected host %q, got %q",
					testCase.wantHost,
					baseURL.Host,
				)
			}

			if baseURL.Path != testCase.wantPath {
				t.Errorf(
					"expected path %q, got %q",
					testCase.wantPath,
					baseURL.Path,
				)
			}

			if baseURL.RawQuery != testCase.wantQuery {
				t.Errorf(
					"expected query %q, got %q",
					testCase.wantQuery,
					baseURL.RawQuery,
				)
			}

			if baseURL.Fragment != testCase.wantFrag {
				t.Errorf(
					"expected fragment %q, got %q",
					testCase.wantFrag,
					baseURL.Fragment,
				)
			}
		})
	}
}

func TestAPIGet(t *testing.T) {
	var (
		gotMethod  string
		gotPath    string
		gotQuery   string
		gotHeaders http.Header
	)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotQuery = r.URL.RawQuery
			gotHeaders = r.Header.Clone()

			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"manga_list": []}`)
		},
	))
	defer server.Close()

	query := url.Values{}
	query.Set("search", "blue lock")
	query.Set("page", "2")

	body, err := apiGet(server.URL, "search", testAPIPath, query, nil)
	if err != nil {
		t.Fatalf("apiGet() error: %v", err)
	}

	if body == nil {
		t.Fatal("expected a response body")
	}

	if gotMethod != http.MethodGet {
		t.Errorf("expected method GET, got %s", gotMethod)
	}

	if gotPath != testAPIPath {
		t.Errorf("expected path %q, got %q", testAPIPath, gotPath)
	}

	// url.Values.Encode() alphabeticalizes the parameters.
	if gotQuery != "page=2&search=blue+lock" {
		t.Errorf("unexpected raw query: %q", gotQuery)
	}

	if got := gotHeaders.Get("Sec-Fetch-Site"); got != "same-origin" {
		t.Errorf(
			"expected Sec-Fetch-Site same-origin header, got %q",
			got,
		)
	}

	if got := gotHeaders.Get("Accept"); got != "application/json" {
		t.Errorf("expected Accept header, got %q", got)
	}

	if got := gotHeaders.Get("User-Agent"); got != "Mozilla/5.0" {
		t.Errorf("expected User-Agent header, got %q", got)
	}
}

func TestAPIGetServerStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")

			http.Error(
				w,
				"challenge page",
				http.StatusForbidden,
			)
		},
	))
	defer server.Close()

	_, err := apiGet(server.URL, "search", testAPIPath, nil, nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 403")
	}

	if !strings.Contains(err.Error(), "returned status 403") {
		t.Errorf(
			"expected a branded status error, got %v",
			err,
		)
	}
}

func TestAPIGetResponseTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(
				bytes.Repeat([]byte("a"), maxResponseBytes+1),
			)
		},
	))
	defer server.Close()

	_, err := apiGet(server.URL, "search", testAPIPath, nil, nil)
	if err == nil {
		t.Fatal("expected an error for an oversized response")
	}

	if !strings.Contains(err.Error(), "response too large") {
		t.Errorf(
			"expected an oversized response error, got %v",
			err,
		)
	}
}
