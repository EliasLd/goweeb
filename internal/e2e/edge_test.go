package e2e

// Edge-challenge detection for the live provider suite. This file is
// deliberately ungated: the logic is pure string matching on provider
// errors, so it compiles and is unit-tested in the hermetic gates
// beside its use in the -tags live flow (providers_live_test.go),
// with no live traffic attached.

import (
	"errors"
	"strings"
	"testing"
)

// Edge challenge markers, taken from the managed challenge the
// mangaball edge serves to flagged IP ranges (nightly report
// 2026-11-07): the page title and the challenge script host. A real
// provider outage never carries these markers, and goweeb cannot
// make an edge answer them by accident: any request answered by a
// challenge was blocked for who asked, not by what goweeb requested.
const (
	challengeTitle = "Just a moment"
	challengeHost  = "challenges.cloudflare.com"
)

// edgeChallenge reports whether a live call failed because the site's
// edge served a bot-protection challenge: an answer no HTTP client
// can produce by design, keyed to the requester's IP reputation
// alone. From such a response nothing can be concluded about the
// provider code under test, so the live suite skips instead of
// failing.
func edgeChallenge(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()

	return strings.Contains(message, challengeTitle) ||
		strings.Contains(message, challengeHost)
}

// shortEdgeMessage trims a blocked-response error to the leading
// slice of its message: the challenge page is thousands of bytes of
// scripts and styles no test log needs. The trim is diagnostic only
// and byte-based on purpose.
func shortEdgeMessage(err error) string {
	const limit = 160

	message := err.Error()
	if len(message) <= limit {
		return message
	}

	return message[:limit] + " (truncated)"
}

func TestEdgeChallengeRecognizesTheChallengePage(t *testing.T) {
	// The real body from the nightly job (2026-11-07), reduced to
	// the identifying parts.
	blocked := errors.New(`MangaBall title search returned status 403: ` +
		`<!DOCTYPE html><html lang="en-US"><head>` +
		`<title>Just a moment...</title>` +
		`<script nonce="x">(function(){window._cf_chl_opt={` +
		`cRay:"a574233f5a7fa213",cZone:"mangaball.com"};})()</script>` +
		`<meta http-equiv="content-security-policy" content="` +
		`script-src 'unsafe-inline' https://challenges.cloudflare.com">`)

	if !edgeChallenge(blocked) {
		t.Error("expected the managed challenge to be recognized")
	}

	// The skip reason stays human-sized: the header line of the
	// answer, not the whole challenge page.
	if got := shortEdgeMessage(blocked); len(got) > 172 {
		t.Errorf("shortEdgeMessage() = %d bytes, want the trimmed form", len(got))
	}
}

func TestEdgeChallengeLeavesOtherFailuresAlone(t *testing.T) {
	failures := []error{
		errors.New("Mangadex manga detail returned status: 503"),
		errors.New("failed to decode MangaBall title search: EOF"),
		errors.New("no results to choose from"),
		errors.New("the AniList API answered status 429"),
		nil,
	}

	for _, err := range failures {
		if edgeChallenge(err) {
			t.Errorf("edgeChallenge(%v) = true, want false", err)
		}
	}
}

// A marker in a mid-page position still matches: the title tag
// travels before or after other headers depending on the edge.
func TestEdgeChallengeMatchesRegardlessOfPosition(t *testing.T) {
	suffix := errors.New("MangaBall title detail returned status 503: " +
		strings.Repeat("noise ", 40) + "Just a moment…")

	if !edgeChallenge(suffix) {
		t.Error("expected a trailing marker to match")
	}
}
