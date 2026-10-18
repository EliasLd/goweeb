package animesamascraper

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mirrors a manga page: the template comments precede the real
// panneauScan calls and use placeholder labels.
const mangaPageWithPanneauScan = `<!DOCTYPE html>
<html>
<body>
	<script>
	/* Cartes des saisons / scans, appelées depuis l'index.php de chaque œuvre :
	panneauAnime("nom", "saison1/vostfr"); panneauScan("nom", "scan/vf"); */
	var OE_BASE = "/catalogue/blue-lock/";
	function panneauScan(nom, url) {
		document.location = OE_BASE + url + "/";
	}
	</script>

	<h2>Manga</h2>
	<div>
		<script>
		/*panneauScan("nom", "url"); -> on en met autant qu'on veut à la ligne*/
		panneauScan("Scans", "scan/vf");
		</script>
	</div>

	<div>
		<script>
		panneauScan("Blue Lock Spin-off", "scan_spin-off/vf");
		</script>
	</div>
</body>
</html>`

const mangaPageWithAnchors = `<!DOCTYPE html>
<html>
<body>
	<a href="/catalogue/blue-lock/scan/vf/">Scans</a>
	<a href="/catalogue/blue-lock/scan/vf/">Scans (duplicate)</a>
	<a href="scan_spin-off/vf/">Spin-off</a>
	<a href="/catalogue/blue-lock">Manga page only</a>
</body>
</html>`

// Mirrors the redesigned manga pages that mix a planning menu anchor
// (same target as a panneauScan call, no trailing slash) with the
// actual scan choice panels.
const mangaPageWithPlanningMenu = `<!DOCTYPE html>
<html>
<body>
	<script>
	/* Cartes des saisons / scans, appelées depuis l'index.php de chaque œuvre :
	panneauAnime("nom", "saison1/vostfr"); panneauScan("nom", "scan/vf"); */
	function panneauScan(nom, url) { }
	</script>

	<div class="oe-planning">
		<a href="/catalogue/blue-lock/scan/vf">
			<b>Scans</b>
		</a>
	</div>

	<div>
		<script>
		/*panneauScan("nom", "url"); -> on en met autant qu'on veut à la ligne*/
		panneauScan("Scans", "scan/vf");
		panneauScan("Blue Lock Spin-off", "scan_spin-off/vf");
		</script>
	</div>
</body>
</html>`

func serveMangaPage(t *testing.T, page string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			_, _ = w.Write([]byte(page))
		},
	))
	t.Cleanup(server.Close)

	return server
}

func TestGetAllScanPathsPrefersRealPanneauScanLabels(
	t *testing.T,
) {
	server := serveMangaPage(t, mangaPageWithPanneauScan)

	paths, err := GetAllScanPaths(
		server.URL+"/catalogue/blue-lock/",
		newTestLogger(),
	)
	if err != nil {
		t.Fatalf("GetAllScanPaths() error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d (%v)", len(paths), paths)
	}

	// The placeholder label from the template comment is replaced
	// by the real label of the later call with the same path.
	if paths[0].Label != "Scans" || paths[0].Path != "scan/vf" {
		t.Errorf(
			"first path = %q / %q, want %q / %q",
			paths[0].Label,
			paths[0].Path,
			"Scans",
			"scan/vf",
		)
	}

	if paths[1].Label != "Blue Lock Spin-off" ||
		paths[1].Path != "scan_spin-off/vf" {
		t.Errorf(
			"second path = %q / %q, want %q / %q",
			paths[1].Label,
			paths[1].Path,
			"Blue Lock Spin-off",
			"scan_spin-off/vf",
		)
	}
}

func TestGetAllScanPathsParsesScanAnchors(t *testing.T) {
	server := serveMangaPage(t, mangaPageWithAnchors)

	paths, err := GetAllScanPaths(
		server.URL+"/catalogue/blue-lock/",
		newTestLogger(),
	)
	if err != nil {
		t.Fatalf("GetAllScanPaths() error: %v", err)
	}

	// Duplicates collapse and non-scan anchors are ignored.
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d (%v)", len(paths), paths)
	}

	if paths[0].Label != "Scans" ||
		paths[0].Path != "/catalogue/blue-lock/scan/vf/" {
		t.Errorf(
			"first path = %q / %q, want %q / %q",
			paths[0].Label,
			paths[0].Path,
			"Scans",
			"/catalogue/blue-lock/scan/vf/",
		)
	}

	if paths[1].Label != "Spin-off" ||
		paths[1].Path != "scan_spin-off/vf/" {
		t.Errorf(
			"second path = %q / %q, want %q / %q",
			paths[1].Label,
			paths[1].Path,
			"Spin-off",
			"scan_spin-off/vf/",
		)
	}
}

func TestGetAllScanPathsMergesPanneauScanAndPlanningAnchors(
	t *testing.T,
) {
	server := serveMangaPage(t, mangaPageWithPlanningMenu)

	paths, err := GetAllScanPaths(
		server.URL+"/catalogue/blue-lock",
		newTestLogger(),
	)
	if err != nil {
		t.Fatalf("GetAllScanPaths() error: %v", err)
	}

	// The planning anchor shares its target with the first
	// panneauScan call, so only the two choice panels surface,
	// with clean labels — and this holds whether the manga URL
	// carries a trailing slash or not.
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d (%v)", len(paths), paths)
	}

	if paths[0].Label != "Scans" || paths[0].Path != "scan/vf" {
		t.Errorf(
			"first path = %q / %q, want %q / %q",
			paths[0].Label,
			paths[0].Path,
			"Scans",
			"scan/vf",
		)
	}

	if paths[1].Label != "Blue Lock Spin-off" ||
		paths[1].Path != "scan_spin-off/vf" {
		t.Errorf(
			"second path = %q / %q, want %q / %q",
			paths[1].Label,
			paths[1].Path,
			"Blue Lock Spin-off",
			"scan_spin-off/vf",
		)
	}
}
