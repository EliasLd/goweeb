package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/EliasLd/goweeb/internal/source/common"
)

// Layout is the naming scheme of downloads inside the output
// directory.
type Layout string

const (
	// LayoutSeries groups every entry of a work under a folder named
	// after the (sanitized) series, with reader-friendly padded
	// filenames parsed by Komga and Kavita:
	//
	//   {output}/{Series}/{Series} - c010.cbz
	LayoutSeries Layout = "series"

	// LayoutFlat writes files next to each other in the output
	// directory, keeping the historical naming verbatim:
	//
	//   {output}/{Series}_{number}.cbz
	LayoutFlat Layout = "flat"
)

// DefaultLayout is used when --layout is not provided (CLI) or not
// settable (TUI).
const DefaultLayout = LayoutSeries

// LayoutChoices renders the supported layouts for usage and errors.
func LayoutChoices() string {
	return string(LayoutSeries) + ", " + string(LayoutFlat)
}

// ParseLayout validates a --layout value.
func ParseLayout(raw string) (Layout, error) {
	layout := Layout(strings.ToLower(strings.TrimSpace(raw)))

	switch layout {
	case LayoutSeries, LayoutFlat:
		return layout, nil
	default:
		return "", fmt.Errorf(
			"invalid layout %q (supported: %s)",
			raw,
			LayoutChoices(),
		)
	}
}

// filenameIllegalChars lists characters that are path separators or
// illegal on common filesystems (Windows notably).
const filenameIllegalChars = `/\:*?"<>|`

// sanitizeSegment makes a series title safe as a path segment.
// Illegal characters become spaces, runs of whitespace are collapsed
// and the result is trimmed of dots and dashes; a title stripped down
// to nothing falls back to a fixed name so paths stay usable.
func sanitizeSegment(segment string) string {
	sanitized := strings.Map(
		func(r rune) rune {
			if strings.ContainsRune(filenameIllegalChars, r) {
				return ' '
			}

			return r
		},
		segment,
	)

	sanitized = strings.Join(strings.Fields(sanitized), " ")
	sanitized = strings.Trim(sanitized, "-. ")

	if sanitized == "" {
		return "download"
	}

	return sanitized
}

// seriesBaseName returns the shared component of series-layout paths,
// one folder per series holding reader-friendly entry names:
//
//	Fate Zero/Fate Zero - c010
func seriesBaseName(
	series string,
	number common.ChapterNumber,
	digits int,
) string {
	segment := sanitizeSegment(series)

	return filepath.Join(
		segment,
		fmt.Sprintf(
			"%s - c%s",
			segment,
			number.Padded(digits),
		),
	)
}

// ledgerSuffix names the manifest file of a work.
const ledgerSuffix = ".goweeb.json"

// ledgerPath returns the path of a work's manifest. It lives inside
// the work's folder in the series layout and at the output root in
// the flat layout — always next to the artifacts it describes, so
// the ledger can rely on plain file names during verification.
func ledgerPath(layout Layout, scanDir, series string) string {
	segment := sanitizeSegment(series)
	name := segment + ledgerSuffix

	if layout == LayoutFlat {
		return filepath.Join(scanDir, name)
	}

	return filepath.Join(scanDir, segment, name)
}

// coverBasePath returns the destination base name of a work's cover
// image: the manifest's name without its suffix, followed by
// ".cover" — e.g. "{out}/{Series}/One Piece.cover" — with the
// extension chosen at download time. Covers thus sit next to the
// manifest describing them, in both layouts.
func coverBasePath(layout Layout, scanDir, series string) string {
	base := ledgerPath(layout, scanDir, series)

	return strings.TrimSuffix(base, ledgerSuffix) + ".cover"
}

// layoutEntryPath returns the output file path of one entry.
//
// The series layout zeroes pads the number so files sort in reading
// order; the flat layout keeps the historical naming verbatim,
// including its lack of sanitization, so existing consumers keep
// working.
func layoutEntryPath(
	layout Layout,
	scanDir string,
	series string,
	number common.ChapterNumber,
	digits int,
	ext string,
) string {
	if layout == LayoutFlat {
		return filepath.Join(
			scanDir,
			fmt.Sprintf(
				"%s_%s%s",
				series,
				number.String(),
				ext,
			),
		)
	}

	return filepath.Join(
		scanDir,
		seriesBaseName(series, number, digits)+ext,
	)
}

// layoutEntryImageDir returns the per-entry image folder used by the
// ebook-friendly mode. The series layout reuses the same
// "{series} - c{number}" name as archives; the flat layout keeps the
// historical "{prefix} {number}" folders next to each other.
func layoutEntryImageDir(
	layout Layout,
	scanDir string,
	series string,
	prefix string,
	number common.ChapterNumber,
	digits int,
) string {
	if layout == LayoutFlat {
		return filepath.Join(
			scanDir,
			fmt.Sprintf(
				"%s %s",
				prefix,
				number.Padded(digits),
			),
		)
	}

	return filepath.Join(
		scanDir,
		seriesBaseName(series, number, digits),
	)
}

// volumeLabel renders a bundle's volume number the way the volume
// mode names every artifact it writes, zero-padded for ordering:
// "Vol. 01".
func volumeLabel(number string, digits int) string {
	number = strings.TrimSpace(number)

	if parsed, err := strconv.Atoi(number); err == nil {
		return fmt.Sprintf("Vol. %0*d", digits, parsed)
	}

	return "Vol. " + number
}

// layoutVolumePath returns the output file path of one volume bundle:
//
//	series: {output}/{Series}/{Series} Vol. 01.cbz
//	flat:   {output}/{Series}_Vol. 01.cbz
func layoutVolumePath(
	layout Layout,
	scanDir string,
	series string,
	number string,
	digits int,
	ext string,
) string {
	volume := volumeLabel(number, digits)

	if layout == LayoutFlat {
		return filepath.Join(
			scanDir,
			fmt.Sprintf(
				"%s_%s%s",
				series,
				volume,
				ext,
			),
		)
	}

	segment := sanitizeSegment(series)

	return filepath.Join(
		scanDir,
		segment,
		fmt.Sprintf(
			"%s %s%s",
			segment,
			volume,
			ext,
		),
	)
}

// layoutVolumeDir returns the per-volume directory grouping the
// per-chapter artifacts (PDF and ebook-friendly volume mode):
//
//	series: {output}/{Series}/Vol. 01
//	flat:   {output}/{Series} Vol. 01
func layoutVolumeDir(
	layout Layout,
	scanDir string,
	series string,
	number string,
	digits int,
) string {
	volume := volumeLabel(number, digits)

	if layout == LayoutFlat {
		return filepath.Join(
			scanDir,
			fmt.Sprintf(
				"%s %s",
				series,
				volume,
			),
		)
	}

	return filepath.Join(
		scanDir,
		sanitizeSegment(series),
		volume,
	)
}

// volumeChapterBaseName names one chapter artifact inside a volume
// directory. Both layouts use the reader-friendly padded form, since
// the surrounding "Vol. NN" directory already carries the layout
// distinction.
func volumeChapterBaseName(
	series string,
	number common.ChapterNumber,
	digits int,
) string {
	return fmt.Sprintf(
		"%s - c%s",
		sanitizeSegment(series),
		number.Padded(digits),
	)
}

// volumeEntryImageDir names one chapter image folder inside a volume
// directory. The series layout keeps the "{series} - c{number}" name
// of its archives; the flat layout keeps its historical
// "{prefix} {number}" folders.
func volumeEntryImageDir(
	volumeDir string,
	layout Layout,
	series string,
	number common.ChapterNumber,
	digits int,
) string {
	if layout == LayoutFlat {
		return filepath.Join(
			volumeDir,
			fmt.Sprintf("Chapter %s", number.Padded(digits)),
		)
	}

	return filepath.Join(
		volumeDir,
		volumeChapterBaseName(series, number, digits),
	)
}
