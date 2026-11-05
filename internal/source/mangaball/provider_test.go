package mangaball

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	cases := []struct {
		name         string
		customDomain string
		expected     string
	}{
		{
			name:     "default domain",
			expected: "https://mangaball.com",
		},
		{
			name:         "whitespace falls back to default",
			customDomain: "   ",
			expected:     "https://mangaball.com",
		},
		{
			name:         "custom domain is trimmed",
			customDomain: "  https://mirror.example.org  ",
			expected:     "https://mirror.example.org",
		},
		{
			name:         "trailing slash is trimmed",
			customDomain: "https://mirror.example.org/",
			expected:     "https://mirror.example.org",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := New(testCase.customDomain)

			if provider.domain != testCase.expected {
				t.Errorf(
					"expected domain %q, got %q",
					testCase.expected,
					provider.domain,
				)
			}
		})
	}
}

func TestName(t *testing.T) {
	provider := New("")

	if provider.Name() != "mangaball" {
		t.Errorf(
			"expected name mangaball, got %q",
			provider.Name(),
		)
	}
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			_, _ = fmt.Fprint(w, `{"code":200,"status":"success","data":[`+
				`{"id":"6851544a702284f8341782f2","name":"Berserk","slug":"berserk"},`+
				`{"id":"685270ea6908629fc2c82502","name":"Otonari Complex","slug":"otomari-complex"}]}`)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	results, err := provider.Search("berserk", nil)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Berserk" {
		t.Errorf("unexpected title: %q", results[0].Title)
	}

	expectedURL := server.URL +
		"/title-detail/6851544a702284f8341782f2-berserk"

	if results[0].URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, results[0].URL)
	}
}

func TestSearchErrorPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			http.Error(
				w,
				"service unavailable",
				http.StatusServiceUnavailable,
			)
		},
	))
	defer server.Close()

	provider := New(server.URL)

	_, err := provider.Search("berserk", nil)
	if err == nil {
		t.Fatal("expected an error from the scraper")
	}
}
