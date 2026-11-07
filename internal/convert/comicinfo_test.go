package convert

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"testing"
)

// writeTestZipEntryAsComicInfo marshals info with the production
// writer into a fresh zip held in memory.
func writeTestComicInfoZip(
	t *testing.T,
	info *ComicInfo,
) *zip.Reader {
	t.Helper()

	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	if err := addComicInfoToZip(zipWriter, info); err != nil {
		t.Fatalf("addComicInfoToZip() error: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zipWriter.Close() error: %v", err)
	}

	reader, err := zip.NewReader(
		bytes.NewReader(buf.Bytes()),
		int64(buf.Len()),
	)
	if err != nil {
		t.Fatalf("zip.NewReader() error: %v", err)
	}

	return reader
}

func TestComicInfoMarshalsExpectedDocument(t *testing.T) {
	info := ComicInfo{
		Series: "One Piece",
		Number: "6.5",
		Count:  42,
		Title:  "Chapter 6.5 - Extra",
		Manga:  MangaYes,
	}

	// Fields appear in schema order, optional fields stay absent
	// when zero, and the file starts with the standard header.
	want := xml.Header + `<ComicInfo>
  <Series>One Piece</Series>
  <Number>6.5</Number>
  <Count>42</Count>
  <Title>Chapter 6.5 - Extra</Title>
  <Manga>Yes</Manga>
</ComicInfo>
`

	got := string(readTestZipEntry(
		t,
		writeTestComicInfoZip(t, &info).File[0],
	))

	if got != want {
		t.Errorf(
			"ComicInfo.xml =\n%q\nwant\n%q",
			got,
			want,
		)
	}
}

func TestComicInfoVolumeField(t *testing.T) {
	info := ComicInfo{
		Series: "One Piece",
		Number: "12",
		Volume: 12,
		Manga:  MangaYes,
	}

	// A set Volume appears right after Count's position, before
	// the omitted fields above it collapse.
	want := xml.Header + `<ComicInfo>
  <Series>One Piece</Series>
  <Number>12</Number>
  <Volume>12</Volume>
  <Manga>Yes</Manga>
</ComicInfo>
`

	got := string(readTestZipEntry(
		t,
		writeTestComicInfoZip(t, &info).File[0],
	))

	if got != want {
		t.Errorf(
			"ComicInfo.xml =\n%q\nwant\n%q",
			got,
			want,
		)
	}
}

func TestComicInfoEscapesValues(t *testing.T) {
	info := ComicInfo{
		Series: `One & <Fruit>`,
		Number: "6.5",
		Manga:  MangaYes,
	}

	entry := readTestZipEntry(
		t,
		writeTestComicInfoZip(t, &info).File[0],
	)

	var decoded ComicInfo

	if err := xml.Unmarshal(entry, &decoded); err != nil {
		t.Fatalf(
			"failed to parse ComicInfo.xml: %v",
			err,
		)
	}

	if decoded.Series != info.Series {
		t.Errorf(
			"Series = %q, want %q",
			decoded.Series,
			info.Series,
		)
	}
}

func TestAddComicInfoToZipEntry(t *testing.T) {
	info := ComicInfo{
		Series: "One Piece",
		Number: "6.5",
		Manga:  MangaYes,
	}

	reader := writeTestComicInfoZip(t, &info)

	if len(reader.File) != 1 {
		t.Fatalf(
			"archive contains %d entries, want 1",
			len(reader.File),
		)
	}

	file := reader.File[0]

	if file.Name != "ComicInfo.xml" {
		t.Errorf(
			"entry name = %q, want %q",
			file.Name,
			"ComicInfo.xml",
		)
	}

	if file.Method != zip.Deflate {
		t.Errorf(
			"entry %s must use zip.Deflate",
			file.Name,
		)
	}

	data := readTestZipEntry(t, file)

	if !bytes.HasPrefix(
		data,
		[]byte(xml.Header),
	) {
		t.Error(
			"expected the file to start with the XML header",
		)
	}
}
