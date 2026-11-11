package mangaballscraper

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/EliasLd/goweeb/internal/logger"
	"github.com/EliasLd/goweeb/internal/source/common"
	sourcetypes "github.com/EliasLd/goweeb/internal/source/types"
)

// ListVolumes groups a title's chapter rows into volumes from the
// site's own per-chapter volume tags. Only some mirrors (comick,
// mangadex) carry one, so untagged chapters stay chapter-mode only.
// The listing walk is shared with the chapter flow: nothing beyond
// the title name is an extra API call.
//
// A row that would leave its volume's bundle incomplete (no id,
// unreadable number) or a chapter the site tags in several volumes
// at once refuses that volume: a mis-bundled volume is worse than no
// volume mode.
func ListVolumes(
	domain string,
	titleID string,
	language string,
	log *logger.Logger,
) (sourcetypes.Work, []sourcetypes.Volume, error) {
	baseURL, err := parseBaseDomain(domain)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	title, err := GetTitleName(domain, titleID, log)
	if err != nil {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"failed to get MangaBall title: %w",
			err,
		)
	}

	listing, err := fetchChapterRows(domain, titleID, language, log)
	if err != nil {
		return sourcetypes.Work{}, nil, err
	}

	if !listing.complete {
		return sourcetypes.Work{}, nil, fmt.Errorf(
			"MangaBall chapter listing is incomplete: %d of %d rows after %d pages",
			len(listing.rows),
			listing.declaredTotal,
			maxListingPages,
		)
	}

	type chapterSources struct {
		number common.ChapterNumber
		ids    []string
		first  chapterRow
	}

	// Rows grouped per volume, then per distinct chapter number,
	// keeping the API's own row order inside each group.
	grouped := make(map[int]map[string]*chapterSources)
	volumeOfNumber := make(map[string]int)
	refused := make(map[int]string)

	for _, row := range listing.rows {
		volume, tagged := rowVolume(row)
		if !tagged {
			// Untagged row: volume mode does not own it.
			continue
		}

		number, err := common.ParseChapterNumber(row.Number.String())

		if row.ID == "" || err != nil {
			if _, seen := refused[volume]; !seen {
				if row.ID == "" {
					refused[volume] = "chapter without an id"
				} else {
					refused[volume] = fmt.Sprintf(
						"unreadable chapter number %q",
						row.Number.String(),
					)
				}
			}

			if log != nil {
				log.Debug(
					"Skipping MangaBall volume %d row without a usable entry: %q\n",
					volume,
					row.Name,
				)
			}

			continue
		}

		key := number.String()

		if previous, exists := volumeOfNumber[key]; exists {
			if previous != volume {
				reason := fmt.Sprintf(
					"chapter %s is tagged in several volumes",
					key,
				)

				if _, seen := refused[previous]; !seen {
					refused[previous] = reason
				}

				if _, seen := refused[volume]; !seen {
					refused[volume] = reason
				}
			} else {
				grouped[volume][key].ids = append(
					grouped[volume][key].ids,
					row.ID,
				)
			}

			continue
		}

		volumeOfNumber[key] = volume

		rows := grouped[volume]
		if rows == nil {
			rows = make(map[string]*chapterSources)
			grouped[volume] = rows
		}

		rows[key] = &chapterSources{
			number: number,
			ids:    []string{row.ID},
			first:  row,
		}
	}

	volumeNumbers := make([]int, 0, len(grouped))
	for volume := range grouped {
		volumeNumbers = append(volumeNumbers, volume)
	}
	for volume := range refused {
		if _, listed := grouped[volume]; !listed {
			volumeNumbers = append(volumeNumbers, volume)
		}
	}
	sort.Ints(volumeNumbers)

	volumes := make([]sourcetypes.Volume, 0, len(volumeNumbers))

	for _, volume := range volumeNumbers {
		if reason, ok := refused[volume]; ok {
			if log != nil {
				log.Warn(
					"Skipping MangaBall volume %d: %s\n",
					volume,
					reason,
				)
			}

			continue
		}

		entries := make([]sourcetypes.Entry, 0, len(grouped[volume]))

		numbers := make([]*chapterSources, 0, len(grouped[volume]))
		for _, sources := range grouped[volume] {
			numbers = append(numbers, sources)
		}

		sort.Slice(numbers, func(i, j int) bool {
			return numbers[i].number.Compare(numbers[j].number) < 0
		})

		for _, sources := range numbers {
			entries = append(entries, sourcetypes.Entry{
				Number: sources.number,
				Label:  chapterLabel(sources.first),
				URL:    chapterEntryURL(baseURL, sources.ids),
			})
		}

		volumes = append(volumes, sourcetypes.Volume{
			Number:  strconv.Itoa(volume),
			Entries: entries,
		})
	}

	if log != nil {
		log.Debug(
			"Found %d MangaBall volume(s) (language %q)\n",
			len(volumes),
			language,
		)
	}

	return sourcetypes.Work{
		Title: title,
		Kind:  sourcetypes.ItemVolume,
	}, volumes, nil
}
