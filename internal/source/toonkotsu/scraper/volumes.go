package toonkotsuscraper

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
)

// VolumeGroup is one site-tagged volume and its chapters, reading
// order.
type VolumeGroup struct {
	Number   string
	Chapters []Chapter
}

// ListVolumes groups the listing's chapters into volumes from the
// site's own per-chapter volume tags: each volume lists its chapters
// in reading order, volumes ascending. Untagged chapters and
// unnumbered rows (webtoon extras, digitized volume entries) stay
// chapter-mode only — volume mode owns the numbered chapters, exactly
// as the site tags them.
//
// A chapter number the listing tags under several volumes refuses
// every bundle it touches: a mis-bundled volume is worse than no
// volume mode. The listing walk is shared with the chapter flow, and
// the edition scan path picks the edition's listing like everywhere
// else.
func ListVolumes(
	domain string,
	code string,
	scanPath string,
	log *logger.Logger,
) (string, []VolumeGroup, error) {
	seriesCode := strings.TrimSpace(code)

	if edition := strings.TrimSpace(scanPath); edition != "" {
		seriesCode = edition
	}

	series, err := GetSeries(domain, seriesCode, log)
	if err != nil {
		return "", nil, err
	}

	uploads := buildChapterUploads(series, log)

	// Chapters the site tags under several volumes refuse every
	// bundle they touch, wherever their other rows sit.
	conflicted := make(map[string]string)

	for _, upload := range uploads {
		if len(upload.vols) <= 1 {
			continue
		}

		reason := fmt.Sprintf(
			"chapter %s is tagged in several volumes",
			upload.number.String(),
		)

		for _, key := range upload.vols {
			if _, seen := conflicted[key]; !seen {
				conflicted[key] = reason
			}
		}
	}

	// The unambiguous volumes, in the tags' canonical order.
	grouped := make(map[string][]chapterUpload)

	var numbers []common.ChapterNumber

	for _, upload := range uploads {
		if len(upload.vols) != 1 {
			continue
		}

		key := upload.vols[0]

		if _, refused := conflicted[key]; refused {
			continue
		}

		if _, seen := grouped[key]; !seen {
			volume, err := common.ParseChapterNumber(key)
			if err != nil {
				continue
			}

			numbers = append(numbers, volume)
		}

		grouped[key] = append(grouped[key], upload)
	}

	sort.Slice(numbers, func(i, j int) bool {
		return numbers[i].Compare(numbers[j]) < 0
	})

	volumes := make([]VolumeGroup, 0, len(numbers))

	for _, number := range numbers {
		key := number.String()

		if reason, refused := conflicted[key]; refused {
			if log != nil {
				log.Warn(
					"Skipping toonkotsu volume %s: %s\n",
					key,
					reason,
				)
			}

			continue
		}

		uploads := grouped[key]

		chapters := make([]Chapter, 0, len(uploads))

		for _, upload := range uploads {
			chapters = append(chapters, Chapter{
				Number: upload.number,
				Label:  upload.label,
				URL:    chapterEntryURL(domain, upload.ids),
			})
		}

		volumes = append(volumes, VolumeGroup{
			Number:   key,
			Chapters: chapters,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d toonkotsu volume(s) (listing %q)\n",
			len(volumes),
			seriesCode,
		)
	}

	return seriesTitle(series), volumes, nil
}
