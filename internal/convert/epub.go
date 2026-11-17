package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/EliasLd/goweeb/internal/logger"
)

// The Open Container Format identifies an EPUB by a first, stored
// mimetype entry. The OEBPS folder holds the package document, the
// navigation document, the pages and the images.
const (
	epubMimeType       = "application/epub+zip"
	epubMimetypeEntry  = "mimetype"
	epubContainerEntry = "META-INF/container.xml"
	epubPackagePath    = "OEBPS/content.opf"
	epubNavPath        = "OEBPS/nav.xhtml"
	epubFolderPrefix   = "OEBPS/"
	epubLanguageNone   = "und"
)

// epubImage locates one embedded image: its staging path and the two
// names it takes inside the container (the zip path and the
// package-relative href the OPF and the pages reference).
type epubImage struct {
	sourcePath string
	zipName    string
	href       string
	mediaType  string
}

// epubPage couples one content document with its page image, the
// fixed-layout viewport geometry included.
type epubPage struct {
	docZipName string
	docHref    string
	image      epubImage
	width      int
	height     int
}

// ImagesToEPUB packs the staged pages of one chapter into a fixed
// layout EPUB 3 book, mirroring ImagesToCBZ. info carries the
// metadata into the package document - no ComicInfo.xml lands in the
// container. coverPath points at the recorded series cover file when
// one exists: it becomes the cover-image, otherwise the first page
// does.
//
// Unlike CBZ, pages are not kept verbatim. WebP and 16-bit PNG pages
// are transcoded to JPEG on staging copies: e-reader firmware (Kobo
// notably) renders WebP blank. A page whose conversion or decoding
// fails is dropped with a log line, and an all-failed chapter fails
// the archive.
func ImagesToEPUB(
	imagesDir string,
	epubPath string,
	info *ComicInfo,
	coverPath string,
	log *logger.Logger,
) error {
	names, err := listImageFiles(imagesDir)
	if err != nil {
		return fmt.Errorf(
			"failed to read image dir: %w",
			err,
		)
	}

	pages := epubPages(imagesDir, names, log)

	if len(pages) == 0 {
		return fmt.Errorf(
			"no readable images found in %s",
			imagesDir,
		)
	}

	cover := epubCover(coverPath, imagesDir, log)

	out, err := os.Create(epubPath)
	if err != nil {
		return fmt.Errorf(
			"failed to create EPUB file: %w",
			err,
		)
	}

	zipWriter := zip.NewWriter(out)

	// A failed write leaves an unusable archive: close both handles
	// and return, so error paths stay one line each.
	abort := func(err error) error {
		_ = zipWriter.Close()
		_ = out.Close()

		return err
	}

	// The mimetype entry must be the container's first member and
	// stored uncompressed, or readers refuse the file.
	mimetype, err := zipWriter.CreateHeader(
		&zip.FileHeader{
			Name:   epubMimetypeEntry,
			Method: zip.Store,
		},
	)
	if err != nil {
		return abort(
			fmt.Errorf(
				"failed to create the mimetype entry: %w",
				err,
			),
		)
	}

	if _, err := mimetype.Write([]byte(epubMimeType)); err != nil {
		return abort(
			fmt.Errorf(
				"failed to write the mimetype: %w",
				err,
			),
		)
	}

	if err := addEPUBDocument(
		zipWriter,
		epubContainerEntry,
		epubContainer(),
	); err != nil {
		return abort(err)
	}

	if err := addEPUBDocument(
		zipWriter,
		epubPackagePath,
		epubPackage(info, pages, cover),
	); err != nil {
		return abort(err)
	}

	if err := addEPUBDocument(
		zipWriter,
		epubNavPath,
		epubNav(info, pages),
	); err != nil {
		return abort(err)
	}

	for i, page := range pages {
		if err := addEPUBDocument(
			zipWriter,
			page.docZipName,
			epubPageDocument(page, i),
		); err != nil {
			return abort(err)
		}

		if err := addImageToZip(
			zipWriter,
			page.image.sourcePath,
			page.image.zipName,
		); err != nil {
			return abort(
				fmt.Errorf(
					"failed to add %s to the archive: %w",
					page.image.zipName,
					err,
				),
			)
		}
	}

	if cover != nil {
		if err := addImageToZip(
			zipWriter,
			cover.sourcePath,
			cover.zipName,
		); err != nil {
			return abort(
				fmt.Errorf(
					"failed to add %s to the archive: %w",
					cover.zipName,
					err,
				),
			)
		}
	}

	// Closing the zip writer flushes the central directory, so it
	// must happen before the file itself is closed.
	if err := zipWriter.Close(); err != nil {
		_ = out.Close()

		return fmt.Errorf(
			"failed to finalize the EPUB archive: %w",
			err,
		)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf(
			"failed to finalize the EPUB archive: %w",
			err,
		)
	}

	return nil
}

// epubPages normalizes the staged pages and measures them: WebP and
// 16-bit PNG pages become JPEGs like in the PDF writer, and the
// format, dimensions and archive names of every readable page are
// collected for the package document.
func epubPages(
	imagesDir string,
	names []string,
	log *logger.Logger,
) []epubPage {
	pages := make([]epubPage, 0, len(names))

	for _, name := range names {
		path := filepath.Join(imagesDir, name)

		if transcodePage(path, log) {
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			log.Error(
				"Failed to open %s: %v\n",
				name,
				err,
			)

			continue
		}

		config, format, err := image.DecodeConfig(file)
		file.Close()

		if err != nil {
			log.Error(
				"Failed to decode image %s: %v\n",
				name,
				err,
			)

			continue
		}

		mediaType, supported := epubImageFormat(format)
		if !supported {
			log.Error(
				"Unsupported image format %s in %s\n",
				format,
				name,
			)

			continue
		}

		// Page numbers count readable pages only, so the spine
		// stays gapless when one page is dropped.
		index := len(pages) + 1

		base := strings.TrimSuffix(name, filepath.Ext(name))

		imageZip := fmt.Sprintf(
			"OEBPS/images/%s%s",
			base,
			epubImageExtension(mediaType),
		)
		docZip := fmt.Sprintf(
			"OEBPS/text/page-%03d.xhtml",
			index,
		)

		pages = append(pages, epubPage{
			docZipName: docZip,
			docHref: strings.TrimPrefix(
				docZip,
				epubFolderPrefix,
			),
			image: epubImage{
				sourcePath: path,
				zipName:    imageZip,
				href: strings.TrimPrefix(
					imageZip,
					epubFolderPrefix,
				),
				mediaType: mediaType,
			},
			width:  config.Width,
			height: config.Height,
		})
	}

	return pages
}

// transcodePage converts the staged page to JPEG when its format
// would not render on e-readers. It reports whether the page must be
// skipped, like the PDF writer skips a failed conversion.
func transcodePage(path string, log *logger.Logger) bool {
	if webp, _ := IsWebP(path); webp {
		if err := WebPToJPG(path, path); err != nil {
			log.Error(
				"Failed to convert WebP: %v\n",
				err,
			)

			return true
		}

		return false
	}

	if sixteenBit, _ := IsSixteenBitPNG(path); sixteenBit {
		if err := PNGToJPG(path, path); err != nil {
			log.Error(
				"failed to convert PNG: %v\n",
				err,
			)

			return true
		}
	}

	return false
}

// epubCover prepares the recorded series cover for embedding,
// transcoding it into the staging folder when needed so the recorded
// file itself is never touched. A missing, unreadable or unsupported
// cover degrades to no dedicated cover: the first page becomes the
// cover-image instead.
func epubCover(
	coverPath string,
	imagesDir string,
	log *logger.Logger,
) *epubImage {
	if coverPath == "" {
		return nil
	}

	if _, err := os.Stat(coverPath); err != nil {
		log.Debug(
			"Series cover %s is gone, using the first page as cover\n",
			coverPath,
		)

		return nil
	}

	source := coverPath

	// The cover may itself be WebP or a 16-bit PNG under any
	// extension; the conversion result lands inside the staging
	// folder, which the caller removes afterwards.
	transcoded := filepath.Join(imagesDir, "cover-transcode.jpg")

	if webp, _ := IsWebP(source); webp {
		if err := WebPToJPG(source, transcoded); err != nil {
			log.Debug(
				"Cover conversion failed: %v\n",
				err,
			)

			return nil
		}

		source = transcoded
	} else if sixteenBit, _ := IsSixteenBitPNG(source); sixteenBit {
		if err := PNGToJPG(source, transcoded); err != nil {
			log.Debug(
				"Cover conversion failed: %v\n",
				err,
			)

			return nil
		}

		source = transcoded
	}

	file, err := os.Open(source)
	if err != nil {
		log.Debug("Cover unreadable: %v\n", err)

		return nil
	}

	_, format, err := image.DecodeConfig(file)
	file.Close()

	if err != nil {
		log.Debug("Cover unreadable: %v\n", err)

		return nil
	}

	mediaType, supported := epubImageFormat(format)
	if !supported {
		log.Debug("Cover format %s unsupported\n", format)

		return nil
	}

	zipName := "OEBPS/images/cover" + epubImageExtension(mediaType)

	return &epubImage{
		sourcePath: source,
		zipName:    zipName,
		href: strings.TrimPrefix(
			zipName,
			epubFolderPrefix,
		),
		mediaType: mediaType,
	}
}

// epubImageFormat maps a Go image format name to the EPUB media
// type. Reading-system support decides what may ride along: JPEG and
// PNG are the formats every viewer renders.
func epubImageFormat(format string) (string, bool) {
	switch format {
	case "jpeg":
		return "image/jpeg", true
	case "png":
		return "image/png", true
	}

	return "", false
}

func epubImageExtension(mediaType string) string {
	switch mediaType {
	case "image/png":
		return ".png"
	default:
		return ".jpg"
	}
}

// epubContainer points the container document at the package.
func epubContainer() string {
	var document strings.Builder

	document.WriteString(xml.Header)
	document.WriteString(
		`<container version="1.0" ` +
			`xmlns="urn:oasis:names:tc:opendocument:xmlns:container">`,
	)
	document.WriteString(`<rootfiles>`)
	document.WriteString(
		`<rootfile full-path="` + epubPackagePath +
			`" media-type="application/oebps-package+xml"/>`,
	)
	document.WriteString(`</rootfiles>`)
	document.WriteString(`</container>`)

	return document.String()
}

// epubTitle is the title readers display in their library. A bare
// chapter label is ambiguous across a work's chapters, so the series
// travels with it.
func epubTitle(info *ComicInfo) string {
	switch {
	case info.Title != "":
		return info.Series + " - " + info.Title
	case info.Number != "":
		return info.Series + " - " + info.Number
	default:
		return info.Series
	}
}

// epubIdentifier names the publication uniquely. Stable naming per
// (series, entry) needs no UUID scheme; the URN marks goweeb's own.
func epubIdentifier(info *ComicInfo) string {
	number := info.Number

	if number == "" {
		number = info.Title
	}

	if number == "" {
		number = "chapter"
	}

	return "urn:goweeb:" + info.Series + ":" + number
}

// epubLanguage keeps the scan path's language. "und" is the honest
// BCP 47 value when nothing recorded one: the OPF requires a
// language and a lie would be worse.
func epubLanguage(info *ComicInfo) string {
	if info.LanguageISO == "" {
		return epubLanguageNone
	}

	return info.LanguageISO
}

// epubPackage builds the OPF package document: the Dublin Core
// metadata taken from the run's ComicInfo, the manifest and the
// fixed-layout spine. The dcterms and rendition prefixes are
// reserved vocabularies and need no declaration.
func epubPackage(
	info *ComicInfo,
	pages []epubPage,
	cover *epubImage,
) string {
	var opf strings.Builder

	opf.WriteString(xml.Header)
	opf.WriteString(
		`<package xmlns="http://www.idpf.org/2007/opf" ` +
			`version="3.0" unique-identifier="pub-id">`,
	)
	opf.WriteString(
		`<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">`,
	)

	opf.WriteString(
		`<dc:identifier id="pub-id">` +
			epubXMLText(epubIdentifier(info)) + `</dc:identifier>`,
	)
	opf.WriteString(
		`<dc:title>` + epubXMLText(epubTitle(info)) + `</dc:title>`,
	)
	opf.WriteString(
		`<dc:language>` +
			epubXMLText(epubLanguage(info)) + `</dc:language>`,
	)

	// The modified timestamp is mandatory in EPUB 3.
	opf.WriteString(
		`<meta property="dcterms:modified">` +
			time.Now().UTC().Format("2006-01-02T15:04:05Z") +
			`</meta>`,
	)

	// One fixed layout for every page: the book is nothing but
	// full-bleed pages.
	opf.WriteString(
		`<meta property="rendition:layout">pre-paginated</meta>`,
	)

	if info.Writer != "" {
		opf.WriteString(
			`<dc:creator>` + epubXMLText(info.Writer) + `</dc:creator>`,
		)
	}

	if info.Penciller != "" {
		opf.WriteString(
			`<dc:contributor>` +
				epubXMLText(info.Penciller) + `</dc:contributor>`,
		)
	}

	// ComicInfo joins the genre terms into "A, B, C"; OPF carries
	// one subject per term.
	for _, genre := range strings.Split(info.Genre, ", ") {
		if genre == "" {
			continue
		}

		opf.WriteString(
			`<dc:subject>` + epubXMLText(genre) + `</dc:subject>`,
		)
	}

	if info.Description != "" {
		opf.WriteString(
			`<dc:description>` +
				epubXMLText(info.Description) + `</dc:description>`,
		)
	}

	if info.Year != 0 {
		opf.WriteString(
			`<dc:date>` + strconv.Itoa(info.Year) + `</dc:date>`,
		)
	}

	opf.WriteString(`</metadata>`)
	opf.WriteString(`<manifest>`)

	opf.WriteString(
		`<item id="nav" href="nav.xhtml" ` +
			`media-type="application/xhtml+xml" properties="nav"/>`,
	)

	for i, page := range pages {
		opf.WriteString(
			`<item id="` + epubPageID(i) + `" href="` + page.docHref +
				`" media-type="application/xhtml+xml"/>`,
		)
	}

	for i, page := range pages {
		// Without a dedicated series cover, the first page doubles
		// as the cover-image.
		coverProperties := ""

		if i == 0 && cover == nil {
			coverProperties = ` properties="cover-image"`
		}

		opf.WriteString(
			`<item id="` + epubImageID(i) +
				`" href="` + page.image.href +
				`" media-type="` + page.image.mediaType + `"` +
				coverProperties + `/>`,
		)
	}

	if cover != nil {
		opf.WriteString(
			`<item id="cover-image" href="` + cover.href +
				`" media-type="` + cover.mediaType +
				`" properties="cover-image"/>`,
		)
	}

	opf.WriteString(`</manifest>`)
	opf.WriteString(`<spine>`)

	for i := range pages {
		opf.WriteString(`<itemref idref="` + epubPageID(i) + `"/>`)
	}

	opf.WriteString(`</spine>`)
	opf.WriteString(`</package>`)

	return opf.String()
}

// epubNav builds the mandatory navigation document. A chapter holds
// no table of contents beyond its pages, so the single entry stays
// minimal and valid.
func epubNav(
	info *ComicInfo,
	pages []epubPage,
) string {
	title := epubXMLText(epubTitle(info))

	var nav strings.Builder

	nav.WriteString(xml.Header)
	nav.WriteString(
		`<html xmlns="http://www.w3.org/1999/xhtml" ` +
			`xmlns:epub="http://www.idpf.org/2007/ops">`,
	)
	nav.WriteString(`<head><title>` + title + `</title></head>`)
	nav.WriteString(`<body>`)
	nav.WriteString(`<nav epub:type="toc" id="toc">`)
	nav.WriteString(`<h1>` + title + `</h1>`)
	nav.WriteString(`<ol>`)
	nav.WriteString(
		`<li><a href="` + pages[0].docHref + `">` +
			title + `</a></li>`,
	)
	nav.WriteString(`</ol>`)
	nav.WriteString(`</nav>`)
	nav.WriteString(`</body>`)
	nav.WriteString(`</html>`)

	return nav.String()
}

// epubPageDocument wraps one page image in the fixed layout: the
// viewport declares the page's real dimensions, so readers render it
// as a full-bleed page instead of scaling it.
func epubPageDocument(
	page epubPage,
	index int,
) string {
	number := strconv.Itoa(index + 1)

	var document strings.Builder

	document.WriteString(xml.Header)
	document.WriteString(
		`<html xmlns="http://www.w3.org/1999/xhtml">`,
	)
	document.WriteString(`<head>`)
	document.WriteString(`<title>Page ` + number + `</title>`)
	document.WriteString(
		`<meta name="viewport" content="width=` +
			strconv.Itoa(page.width) +
			`, height=` + strconv.Itoa(page.height) + `"/>`,
	)
	document.WriteString(`</head>`)
	document.WriteString(`<body style="margin: 0;">`)
	document.WriteString(
		`<img src="../` + page.image.href + `" alt="Page ` +
			number + `"/>`,
	)
	document.WriteString(`</body>`)
	document.WriteString(`</html>`)

	return document.String()
}

func epubPageID(index int) string {
	return fmt.Sprintf("page-%03d", index+1)
}

func epubImageID(index int) string {
	return fmt.Sprintf("img-%03d", index+1)
}

// epubXMLText makes an arbitrary metadata string safe inside one XML
// element or attribute.
func epubXMLText(s string) string {
	var escaped strings.Builder

	if err := xml.EscapeText(&escaped, []byte(s)); err != nil {
		return ""
	}

	return escaped.String()
}

// addEPUBDocument writes one XML or plain text member into the
// container. Unlike already compressed page images, these shrink
// well under the DEFLATE method.
func addEPUBDocument(
	zipWriter *zip.Writer,
	name string,
	document string,
) error {
	entry, err := zipWriter.Create(name)
	if err != nil {
		return fmt.Errorf(
			"failed to create zip entry %s: %w",
			name,
			err,
		)
	}

	if _, err := entry.Write([]byte(document)); err != nil {
		return fmt.Errorf(
			"failed to write zip entry %s: %w",
			name,
			err,
		)
	}

	return nil
}
