package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
)

// Manga reading-direction value written into ComicInfo.xml. Content
// comes from manga providers and pages already carry their final
// order, so no stricter direction claim is made.
const MangaYes = "Yes"

// ComicInfo is the subset of the ComicRack ComicInfo 2.0 schema that
// goweeb fills. Komga, Kavita and Mihon read this file when a CBZ
// lands in their library, grouping it into the named series.
type ComicInfo struct {
	XMLName   xml.Name `xml:"ComicInfo"`
	Series    string   `xml:"Series"`
	Number    string   `xml:"Number"`
	Count     int      `xml:"Count,omitempty"`
	Volume    int      `xml:"Volume,omitempty"`
	Title     string   `xml:"Title,omitempty"`
	PageCount int      `xml:"PageCount,omitempty"`
	Manga     string   `xml:"Manga,omitempty"`
}

// addComicInfoToZip embeds the metadata as ComicInfo.xml at the
// archive root. The XML is DEFLATE-compressed: unlike already
// compressed page images, plain text shrinks well.
func addComicInfoToZip(
	zipWriter *zip.Writer,
	info *ComicInfo,
) error {
	data, err := xml.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"failed to encode ComicInfo: %w",
			err,
		)
	}

	// Append a trailing newline so the file ends like any plain
	// text file when opened in an editor.
	document := append(data, '\n')

	header := &zip.FileHeader{
		Name:   "ComicInfo.xml",
		Method: zip.Deflate,
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf(
			"failed to create zip entry: %w",
			err,
		)
	}

	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf(
			"failed to write ComicInfo header: %w",
			err,
		)
	}

	if _, err := writer.Write(document); err != nil {
		return fmt.Errorf(
			"failed to write ComicInfo: %w",
			err,
		)
	}

	return nil
}
