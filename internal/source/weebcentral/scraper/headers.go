package weebcentralscraper

import "net/http"

// WeebCentral rides behind Cloudflare bot protection, which also
// scores datacenter IPs at the edge before the site's own
// application runs. These profiles mirror what a real browser
// sends for the two request classes the provider makes: scripts
// fetching the search endpoint, and full page navigations. Whether
// they convince the edge from CI runners is settled by the
// scheduled Provider checks workflow (the User-Agent also stays
// more browser-faithful from residential IPs).
const weebcentralUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0"

// weebcentralFetchHeaders matches the site's own JavaScript
// fetching /search/data (Accept: */*, same-origin fetch).
var weebcentralFetchHeaders = map[string]string{
	"User-Agent":     weebcentralUserAgent,
	"Accept":         "*/*",
	"Sec-Fetch-Dest": "empty",
	"Sec-Fetch-Mode": "cors",
	"Sec-Fetch-Site": "same-origin",
}

// weebcentralNavigationHeaders matches a full page visit.
var weebcentralNavigationHeaders = map[string]string{
	"User-Agent":     weebcentralUserAgent,
	"Accept":         "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Sec-Fetch-Dest": "document",
	"Sec-Fetch-Mode": "navigate",
	"Sec-Fetch-Site": "none",
}

// applyNavigationHeaders copies the page-navigation profile onto a
// request, replacing the previous bare User-Agent.
func applyNavigationHeaders(req *http.Request) {
	for key, value := range weebcentralNavigationHeaders {
		req.Header.Set(key, value)
	}
}
