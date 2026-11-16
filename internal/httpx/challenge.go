package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrEdgeChallenge marks a response answered by the site's edge
// instead of the application: Cloudflare serves a managed challenge
// or an explicit block for requests it flagged, keyed to the
// requester's reputation alone. No HTTP client can turn such an
// answer into the resource, so these responses are never retried.
var ErrEdgeChallenge = errors.New("edge answered with a bot-protection challenge")

// challengeResponse reports whether the edge itself answered the
// request, wrapping the sentinel with its header value.
func challengeResponse(resp *http.Response) error {
	value := resp.Header.Get("cf-mitigated")

	switch value {
	case "challenge", "block":
		return fmt.Errorf(
			"%w (cf-mitigated: %s)",
			ErrEdgeChallenge,
			value,
		)
	}

	return nil
}
