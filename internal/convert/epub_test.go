package convert

import (
	"archive/zip"
	"encoding/xml"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EliasLd/goweeb/internal/logger"
)

// epubTestOPFMetadata mirrors the Dublin Core fields the writer
// fills, namespace-checked.
type epubTestOPFMetadata struct {
	Identifier  string   `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Title       string   `xml:"http://purl.org/dc/elements/1.1/ title"`
	Language    string   `xml:"http://purl.org/dc/elements/1.1/ language"`
	Creator     string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Contributor string   `xml:"http://purl.org/dc/elements/1.1/ contributor"`
	Subjects    []string `xml:"http://purl.org/dc/elements/1.1/ subject"`
	Description string   `xml:"http://purl.org/dc/elements/1.1/ description"`
	Date        string   `xml:"http://purl.org/dc/elements/1.1/ date"`
}

type epubTestOPFItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type epubTestOPF struct {
	XMLName          xml.Name            `xml:"http://www.idpf.org/2007/opf package"`
	UniqueIdentifier string              `xml:"unique-identifier,attr"`
	Metadata         epubTestOPFMetadata `xml:"metadata"`
	Items            []epubTestOPFItem   `xml:"manifest>item"`
	IDRefs           []struct {
		IDRef string `xml:"idref,attr"`
	} `xml:"spine>itemref"`
}

func epubTestLogger() *logger.Logger {
	return logger.New(io.Discard, logger.LevelError)
}

func findTestZipEntry(
	t *testing.T,
	reader *zip.Reader,
	name string,
) *zip.File {
	t.Helper()

	for _, file := range reader.File {
		if file.Name == name {
			return file
		}
	}

	t.Fatalf("zip entry %s missing", name)

	return nil
}

func parseTestOPF(
	t *testing.T,
	data []byte,
) epubTestOPF {
	t.Helper()

	var parsed epubTestOPF

	if err := xml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse the OPF: %v", err)
	}

	return parsed
}

func writeTestPages(
	t *testing.T,
	dir string,
) {
	t.Helper()

	encodeTestJPEG(
		t,
		filepath.Join(dir, "001.jpg"),
		color.RGBA{R: 0x10, A: 0xFF},
	)
	encodeTestJPEG(
		t,
		filepath.Join(dir, "002.jpg"),
		color.RGBA{R: 0x20, A: 0xFF},
	)
}

// The Open Container contract: mimetype first, stored, byte-exact;
// container.xml pointing at the package; no ComicInfo.xml, whose
// place the OPF metadata takes.
func TestImagesToEPUBContainer(t *testing.T) {
	dir := t.TempDir()
	writeTestPages(t, dir)

	epubPath := filepath.Join(dir, "out.epub")

	if err := ImagesToEPUB(
		dir,
		epubPath,
		&ComicInfo{Series: "Berserk"},
		"",
		epubTestLogger(),
	); err != nil {
		t.Fatalf("ImagesToEPUB() error: %v", err)
	}

	reader := readTestZip(t, epubPath)

	mimetype := findTestZipEntry(t, reader, "mimetype")

	if mimetype.Method != zip.Store {
		t.Error("the mimetype entry must be stored uncompressed")
	}

	if content := readTestZipEntry(t, mimetype); content != nil {
		if string(content) != epubMimeType {
			t.Errorf(
				"mimetype = %q, want %q",
				content,
				epubMimeType,
			)
		}
	}

	container := string(
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, epubContainerEntry),
		),
	)

	if !strings.Contains(
		container,
		`full-path="OEBPS/content.opf"`,
	) {
		t.Errorf(
			"container.xml does not point at the package: %s",
			container,
		)
	}

	for _, file := range reader.File {
		if file.Name == "ComicInfo.xml" {
			t.Error(
				"the ComicInfo.xml must stay a CBZ entry only",
			)
		}
	}
}

// The package document: Dublin Core mapping, manifest shape and the
// fixed-layout spine. Without a dedicated cover the first page image
// carries the cover-image property.
func TestImagesToEPUBPackageMetadata(t *testing.T) {
	dir := t.TempDir()
	writeTestPages(t, dir)

	info := &ComicInfo{
		Series:      "Berserk",
		Number:      "1",
		Title:       "The Black Swordsman",
		Writer:      "Miura Kentarou",
		Penciller:   "Studio Gaga",
		Genre:       "Action, Drama",
		Description: "A mercenary joins <the Band>.",
		LanguageISO: "en",
		Year:        1989,
		Manga:       MangaYes,
	}

	epubPath := filepath.Join(dir, "out.epub")

	if err := ImagesToEPUB(
		dir,
		epubPath,
		info,
		"",
		epubTestLogger(),
	); err != nil {
		t.Fatalf("ImagesToEPUB() error: %v", err)
	}

	reader := readTestZip(t, epubPath)

	opf := parseTestOPF(
		t,
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, epubPackagePath),
		),
	)

	if opf.UniqueIdentifier != "pub-id" {
		t.Errorf(
			"unique-identifier = %q, want \"pub-id\"",
			opf.UniqueIdentifier,
		)
	}

	metadata := opf.Metadata

	if metadata.Identifier != "urn:goweeb:Berserk:1" {
		t.Errorf("identifier = %q", metadata.Identifier)
	}

	if metadata.Title != "Berserk - The Black Swordsman" {
		t.Errorf("title = %q", metadata.Title)
	}

	if metadata.Language != "en" {
		t.Errorf("language = %q", metadata.Language)
	}

	if metadata.Creator != "Miura Kentarou" {
		t.Errorf("creator = %q", metadata.Creator)
	}

	if metadata.Contributor != "Studio Gaga" {
		t.Errorf("contributor = %q", metadata.Contributor)
	}

	if len(metadata.Subjects) != 2 ||
		metadata.Subjects[0] != "Action" ||
		metadata.Subjects[1] != "Drama" {
		t.Errorf("subjects = %v", metadata.Subjects)
	}

	if metadata.Description != "A mercenary joins <the Band>." {
		t.Errorf("description = %q", metadata.Description)
	}

	if metadata.Date != "1989" {
		t.Errorf("date = %q", metadata.Date)
	}

	opfText := string(
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, epubPackagePath),
		),
	)

	if !strings.Contains(
		opfText,
		`<meta property="dcterms:modified">`,
	) {
		t.Error("the modified timestamp is mandatory in EPUB 3")
	}

	if !strings.Contains(
		opfText,
		`<meta property="rendition:layout">pre-paginated</meta>`,
	) {
		t.Error("the fixed layout must be declared pre-paginated")
	}

	items := opf.Items

	if len(items) != 1 /* nav */ +2 /* pages */ +2 /* images */ {
		t.Fatalf("manifest holds %d items", len(items))
	}

	if !strings.Contains(opfText, `properties="nav"`) {
		t.Error("the navigation item must carry the nav property")
	}

	if items[1].ID != "page-001" || items[1].Href != "text/page-001.xhtml" {
		t.Errorf("page item = %+v", items[1])
	}

	// First image doubles as the cover.
	if items[3].Href != "images/001.jpg" ||
		items[3].MediaType != "image/jpeg" ||
		items[3].Properties != "cover-image" {
		t.Errorf("cover image item = %+v", items[3])
	}

	if len(opf.IDRefs) != 2 || opf.IDRefs[0].IDRef != "page-001" {
		t.Errorf("spine = %v", opf.IDRefs)
	}
}

// A recorded cover claims its own item; the first page loses the
// cover-image property.
func TestImagesToEPUBDedicatedCover(t *testing.T) {
	dir := t.TempDir()
	writeTestPages(t, dir)

	coverPath := filepath.Join(dir, "..", "series.cover.jpg")
	encodeTestJPEG(t, coverPath, color.RGBA{G: 0x40, A: 0xFF})

	epubPath := filepath.Join(dir, "out.epub")

	if err := ImagesToEPUB(
		dir,
		epubPath,
		&ComicInfo{Series: "Berserk", Number: "1"},
		coverPath,
		epubTestLogger(),
	); err != nil {
		t.Fatalf("ImagesToEPUB() error: %v", err)
	}

	reader := readTestZip(t, epubPath)
	opf := parseTestOPF(
		t,
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, epubPackagePath),
		),
	)

	if content := readTestZipEntry(
		t,
		findTestZipEntry(t, reader, "OEBPS/images/cover.jpg"),
	); content == nil {
		t.Fatal("the cover image is missing from the container")
	}

	var cover epubTestOPFItem

	for _, item := range opf.Items {
		if item.Properties == "cover-image" {
			if item.ID != "cover-image" || item.Href != "images/cover.jpg" {
				t.Errorf("cover item = %+v", item)
			}

			cover = item

			break
		}
	}

	if cover.ID == "" {
		t.Error("no item carries the cover-image property")
	}
}

// The fixed layout documents: the viewport declares the page's real
// dimensions, the image spans it fully.
func TestImagesToEPUBFixedLayoutPage(t *testing.T) {
	dir := t.TempDir()
	writeTestPages(t, dir)

	epubPath := filepath.Join(dir, "out.epub")

	if err := ImagesToEPUB(
		dir,
		epubPath,
		&ComicInfo{Series: "Berserk", Number: "1"},
		"",
		epubTestLogger(),
	); err != nil {
		t.Fatalf("ImagesToEPUB() error: %v", err)
	}

	reader := readTestZip(t, epubPath)

	page := string(
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, "OEBPS/text/page-001.xhtml"),
		),
	)

	// The test JPEGs are 8x11 pixels (encodeTestJPEG).
	if !strings.Contains(page, `content="width=8, height=11"`) {
		t.Errorf("viewport = %s", page)
	}

	if !strings.Contains(page, `src="../images/001.jpg"`) {
		t.Errorf("image reference = %s", page)
	}

	if !strings.Contains(page, `margin: 0;`) {
		t.Errorf("the page body must bleed freely: %s", page)
	}
}

// WebP pages cannot ride along into e-readers (Kobo renders WebP
// blank): the writer transcodes them to JPEG under the corrected
// extension.
func TestImagesToEPUBTranscodesWebP(t *testing.T) {
	fixture, err := os.ReadFile(
		filepath.Join("testdata", "page.webp"),
	)
	if err != nil {
		t.Fatalf("webp fixture missing: %v", err)
	}

	dir := t.TempDir()

	writeTestFile(t, filepath.Join(dir, "001.webp"), fixture)
	encodeTestJPEG(
		t,
		filepath.Join(dir, "002.jpg"),
		color.RGBA{B: 0x30, A: 0xFF},
	)

	epubPath := filepath.Join(dir, "out.epub")

	if err := ImagesToEPUB(
		dir,
		epubPath,
		&ComicInfo{Series: "Berserk", Number: "1"},
		"",
		epubTestLogger(),
	); err != nil {
		t.Fatalf("ImagesToEPUB() error: %v", err)
	}

	reader := readTestZip(t, epubPath)

	if content := readTestZipEntry(
		t,
		findTestZipEntry(t, reader, "OEBPS/images/001.jpg"),
	); content == nil {
		t.Fatal("the transcoded page is missing")
	}

	opf := parseTestOPF(
		t,
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, epubPackagePath),
		),
	)

	for _, item := range opf.Items {
		if item.Href == "images/001.webp" {
			t.Error("the webp page must not be embedded verbatim")
		}
	}
}

// A page nothing can decode is dropped; the spine stays gapless.
func TestImagesToEPUBSkipsUnreadablePage(t *testing.T) {
	dir := t.TempDir()

	encodeTestJPEG(
		t,
		filepath.Join(dir, "001.jpg"),
		color.RGBA{R: 0x10, A: 0xFF},
	)
	writeTestFile(
		t,
		filepath.Join(dir, "002.jpg"),
		[]byte("neither jpeg nor png nor webp"),
	)

	epubPath := filepath.Join(dir, "out.epub")

	if err := ImagesToEPUB(
		dir,
		epubPath,
		&ComicInfo{Series: "Berserk", Number: "1"},
		"",
		epubTestLogger(),
	); err != nil {
		t.Fatalf("ImagesToEPUB() error: %v", err)
	}

	reader := readTestZip(t, epubPath)
	opf := parseTestOPF(
		t,
		readTestZipEntry(
			t,
			findTestZipEntry(t, reader, epubPackagePath),
		),
	)

	if len(opf.IDRefs) != 1 ||
		opf.IDRefs[0].IDRef != "page-001" {
		t.Errorf("spine = %v, want the single readable page", opf.IDRefs)
	}
}

func TestImagesToEPUBEmptyDir(t *testing.T) {
	dir := t.TempDir()

	err := ImagesToEPUB(
		dir,
		filepath.Join(dir, "out.epub"),
		nil,
		"",
		epubTestLogger(),
	)

	if err == nil {
		t.Error("an empty image dir must fail the archive")
	}
}
