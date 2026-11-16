package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChallengeResponseFailsFast drives the header detection through
// ReadAll. The challenge case rides a retryable status on purpose: it
// proves the short-circuit sits before the retry ladder.
func TestChallengeResponseFailsFast(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		status   int
		sentinel bool
	}{
		{"managed challenge", "challenge", http.StatusTooManyRequests, true},
		{"explicit block", "block", http.StatusForbidden, true},
		{"plain status", "", http.StatusForbidden, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hits := 0

			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, _ *http.Request) {
						hits++

						if test.header != "" {
							w.Header().Set("cf-mitigated", test.header)
						}

						w.WriteHeader(test.status)
					},
				),
			)

			defer server.Close()

			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ReadAll(&http.Client{}, req, nil)

			if got := errors.Is(err, ErrEdgeChallenge); got != test.sentinel {
				t.Fatalf(
					"errors.Is(err, ErrEdgeChallenge) = %v, want %v (err: %v)",
					got,
					test.sentinel,
					err,
				)
			}

			if hits != 1 {
				t.Errorf("server hit %d times, want exactly one", hits)
			}

			if test.header != "" &&
				!strings.Contains(err.Error(), "(cf-mitigated: "+test.header+")") {
				t.Errorf("error missing the header value: %v", err)
			}
		})
	}
}
