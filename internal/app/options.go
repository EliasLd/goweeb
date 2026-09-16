package app

import (
	"flag"
	"fmt"
	"github.com/EliasLd/goweeb/internal/source"
	"log"
	"os"
	"sort"
	"strings"
)

type RangeMode int

const (
	RangeNone = iota
	RangeNormal
	RangeOpenEnded
	RangeLastN
)

func isValidProvider(name string) bool {
	return source.IsSupported(name)
}

func supportedProvidersList() string {
	providers := source.AvailableProviders()

	names := make([]string, 0, len(providers))

	for _, provider := range providers {
		names = append(names, provider.ID)
	}

	sort.Strings(names)

	return strings.Join(names, ", ")
}

// Holds parsed CLI arguments
type Options struct {
	Slug          string
	All           bool
	Range         [2]int // [0] = start, [1] = end (0 means open-ended)
	RangeMode     RangeMode
	ScanDir       string
	Cleanup       bool
	CustomDomain  string // custom domain override
	Debug         bool
	EbookFriendly bool
	Source        string

	MangaURL string
	ScanPath string
}

func ParseFlags() Options {
	// Define flags
	allFlag := flag.Bool("all", false, "Download all available chapters")
	allShort := flag.Bool("a", false, "Shortand for --all)")

	sourceFlag := flag.String("source", "", "Source provider...")

	rangeFlag := flag.String("range", "", "Range of chapters to download, e.g., 10-77, 14-")
	rangeShort := flag.String("r", "", "Shorthand for --range")

	var scanDir string
	flag.StringVar(&scanDir, "scan-dir", "scan", "Directory to save the generated PDF files")
	flag.StringVar(&scanDir, "d", "pdf", "Shorthand for --scan-dir")

	ebookFlag := flag.Bool("ebook-friendly", false, "Save chapters as image folders, compatible with Kindle Comic Converter (no pdf output)")

	keepImagesFlag := flag.Bool("keep-images", false, "Keep images after PDF creation")
	keepImagesShort := flag.Bool("k", false, "Shorthand for --keep-images")

	var customDomain string
	flag.StringVar(&customDomain, "domain", "", "Override the provider's default domain")
	flag.StringVar(&customDomain, "u", "", "Shorthand for --domain")

	debugFlag := flag.Bool("debug", false, "Enable verbose debug logging")

	flag.Parse()

	// Expecting slug (manga title) as a positional argument
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: scan-scraper [options] <manga-slug>")
		flag.PrintDefaults()
		os.Exit(1)
	}
	slug := args[0]

	// Resolve final values
	all := *allFlag || *allShort

	sourceName := strings.ToLower(strings.TrimSpace(*sourceFlag))

	if sourceName == "" {
		fmt.Println("Missing required option: --source")
		fmt.Printf("Supported providers: %s\n", supportedProvidersList())
		os.Exit(1)
	}

	if !isValidProvider(sourceName) {
		fmt.Printf("Unsupported source provider: %q\n", sourceName)
		fmt.Printf("Supported providers: %s\n", supportedProvidersList())
		os.Exit(1)
	}

	dir := scanDir
	keepImages := *keepImagesFlag || *keepImagesShort
	domain := customDomain
	debug := *debugFlag
	ebook := *ebookFlag

	// Normalize domain (remove trailing slash, ensure https://)
	if domain != "" {
		domain = strings.TrimSuffix(domain, "/")
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
	}

	// Parse chapters range
	rangeStr := *rangeFlag
	if *rangeShort != "" {
		rangeStr = *rangeShort
	}

	var chapterRange [2]int
	var rangeMode RangeMode = RangeNone

	if rangeStr != "" {
		// Handle last N chapters
		if strings.HasPrefix(rangeStr, "-") && len(rangeStr) > 1 {
			var nLast int
			n, err := fmt.Sscanf(rangeStr, "-%d", &nLast)
			if err != nil || n != 1 || nLast <= 0 {
				fmt.Printf("Invalid range format: %s. Use format <start-end>, <start>- or -<N>\n", rangeStr)
				os.Exit(1)
			}
			chapterRange[0] = 0 // useles for LastN format, filled for compatiblity
			chapterRange[1] = nLast
			rangeMode = RangeLastN
			// Handle open-ended range like "10-"
		} else if strings.HasSuffix(rangeStr, "-") {
			var start int
			trimmed := strings.TrimSuffix(rangeStr, "-")
			n, err := fmt.Sscanf(trimmed, "%d", &start)
			if err != nil || n != 1 {
				fmt.Printf("Invalid range format: %s. Use format: <start>-<end> or <start>-\n", rangeStr)
				os.Exit(1)
			}
			chapterRange[0] = start
			chapterRange[1] = 0 // 0 means open-ended
			rangeMode = RangeOpenEnded
			// Normal range like "10-20"
		} else {
			var start, end int
			n, err := fmt.Sscanf(rangeStr, "%d-%d", &start, &end)
			if err == nil && n == 2 && start <= end {
				chapterRange[0] = start
				chapterRange[1] = end
				rangeMode = RangeNormal
			} else {
				// If only one chapter provided
				var solo int
				n, err := fmt.Sscanf(rangeStr, "%d", &solo)
				if err == nil && n == 1 {
					chapterRange[0] = solo
					chapterRange[1] = solo
					rangeMode = RangeNormal
				} else {
					fmt.Printf("Invalid range format: %s. Use format: <start>-<end>, <start>-, -<N> or <chapter>\n", rangeStr)
					os.Exit(1)
				}
			}
		}
	}

	// Create scanDir if it doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		const defaultDirPerm = 0755
		err := os.MkdirAll(dir, defaultDirPerm)
		if err != nil {
			log.Fatalf("Failed to create scan-dir (%s): %v", dir, err)
		}
	}

	return Options{
		Slug:          slug,
		All:           all,
		Source:        sourceName,
		Range:         chapterRange,
		ScanDir:       dir,
		RangeMode:     rangeMode,
		Cleanup:       !keepImages,
		CustomDomain:  domain,
		Debug:         debug,
		EbookFriendly: ebook,
	}
}
