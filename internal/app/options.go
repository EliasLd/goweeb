package app

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/EliasLd/goweeb/internal/source"
)

type RangeMode int

const (
	RangeNone RangeMode = iota
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

// Holds parsed CLI arguments.
type Options struct {
	Slug          string
	Selection     RangeSelection
	ScanDir       string
	Cleanup       bool
	CustomDomain  string
	Debug         bool
	EbookFriendly bool
	Source        string

	MangaURL string
	ScanPath string
}

func configureUsage() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()

		fmt.Fprintln(out, "goweeb - Fast and lightweight manga downloader")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "Usage:")
		fmt.Fprintln(out, `  goweeb [options] "<manga title>"`)
		fmt.Fprintln(out)

		fmt.Fprintln(out, "Required:")
		fmt.Fprintln(out, "  --source <provider>")
		fmt.Fprintln(out, "      Source provider")
		fmt.Fprintf(
			out,
			"      Supported: %s\n",
			supportedProvidersList(),
		)
		fmt.Fprintln(out)

		fmt.Fprintln(out, "Chapter selection:")
		fmt.Fprintln(out, "  --all")
		fmt.Fprintln(out, "  -a")
		fmt.Fprintln(out, "      Download all available chapters")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "  --range <selection>")
		fmt.Fprintln(out, "  -r <selection>")
		fmt.Fprintln(out, "      Select chapters to download")
		fmt.Fprintln(out, "      Examples:")
		fmt.Fprintln(out, "        10              chapter 10")
		fmt.Fprintln(out, "        1-10            chapters 1 through 10")
		fmt.Fprintln(out, "        10-             chapter 10 onwards")
		fmt.Fprintln(out, "        -10             last 10 available chapters")
		fmt.Fprintln(out, "        1-10,20-30      multiple ranges")
		fmt.Fprintln(out, "        1,5,10          multiple individual chapters")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "Output:")
		fmt.Fprintln(out, "  --scan-dir <directory>")
		fmt.Fprintln(out, "  -d <directory>")
		fmt.Fprintln(out, "      Output directory (default: scan)")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "  --ebook-friendly")
		fmt.Fprintln(out, "      Save chapters as image folders instead of PDF files")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "  --keep-images")
		fmt.Fprintln(out, "  -k")
		fmt.Fprintln(out, "      Keep downloaded images after PDF creation")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "Provider options:")
		fmt.Fprintln(out, "  --domain <domain>")
		fmt.Fprintln(out, "  -u <domain>")
		fmt.Fprintln(out, "      Override the provider's default domain")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "Other:")
		fmt.Fprintln(out, "  --debug")
		fmt.Fprintln(out, "      Enable verbose debug logging")
		fmt.Fprintln(out)

		fmt.Fprintln(out, "  --help")
		fmt.Fprintln(out, "  -h")
		fmt.Fprintln(out, "      Show this help message")
	}
}

func ParseFlags() Options {
	configureUsage()

	var (
		all           bool
		sourceName    string
		rangeStr      string
		scanDir       string
		ebookFriendly bool
		keepImages    bool
		customDomain  string
		debug         bool
	)

	// Chapter selection.
	flag.BoolVar(
		&all,
		"all",
		false,
		"Download all available chapters",
	)
	flag.BoolVar(
		&all,
		"a",
		false,
		"Shorthand for --all",
	)

	flag.StringVar(
		&rangeStr,
		"range",
		"",
		"Chapter selection",
	)
	flag.StringVar(
		&rangeStr,
		"r",
		"",
		"Shorthand for --range",
	)

	// Provider.
	flag.StringVar(
		&sourceName,
		"source",
		"",
		"Source provider",
	)

	flag.StringVar(
		&customDomain,
		"domain",
		"",
		"Override the provider's default domain",
	)
	flag.StringVar(
		&customDomain,
		"u",
		"",
		"Shorthand for --domain",
	)

	// Output.
	flag.StringVar(
		&scanDir,
		"scan-dir",
		"scan",
		"Directory to save generated files",
	)
	flag.StringVar(
		&scanDir,
		"d",
		"scan",
		"Shorthand for --scan-dir",
	)

	flag.BoolVar(
		&ebookFriendly,
		"ebook-friendly",
		false,
		"Save chapters as image folders instead of PDF files",
	)

	flag.BoolVar(
		&keepImages,
		"keep-images",
		false,
		"Keep images after PDF creation",
	)
	flag.BoolVar(
		&keepImages,
		"k",
		false,
		"Shorthand for --keep-images",
	)

	// Misc.
	flag.BoolVar(
		&debug,
		"debug",
		false,
		"Enable verbose debug logging",
	)

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		fmt.Fprintln(
			os.Stderr,
			"Missing manga title.",
		)
		fmt.Fprintln(os.Stderr)

		flag.Usage()
		os.Exit(2)
	}

	// Allows both:
	//
	//   goweeb ... "one piece"
	//
	// and:
	//
	//   goweeb ... one piece
	//
	// as long as the positional arguments come after the flags.
	slug := strings.TrimSpace(
		strings.Join(args, " "),
	)

	sourceName = strings.ToLower(
		strings.TrimSpace(sourceName),
	)

	if sourceName == "" {
		fmt.Fprintln(
			os.Stderr,
			"Missing required option: --source",
		)

		fmt.Fprintf(
			os.Stderr,
			"Supported providers: %s\n",
			supportedProvidersList(),
		)

		os.Exit(2)
	}

	if !isValidProvider(sourceName) {
		fmt.Fprintf(
			os.Stderr,
			"Unsupported source provider: %q\n",
			sourceName,
		)

		fmt.Fprintf(
			os.Stderr,
			"Supported providers: %s\n",
			supportedProvidersList(),
		)

		os.Exit(2)
	}

	rangeStr = strings.TrimSpace(rangeStr)

	if all && rangeStr != "" {
		fmt.Fprintln(
			os.Stderr,
			"--all and --range cannot be used together",
		)

		os.Exit(2)
	}

	var selection RangeSelection

	if all {
		selection.All = true
	} else if rangeStr != "" {
		parsedSelection, err := ParseRangeExpression(
			rangeStr,
		)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Invalid chapter selection: %v\n",
				err,
			)

			os.Exit(2)
		}

		selection = parsedSelection
	}

	customDomain = strings.TrimSpace(
		customDomain,
	)

	if customDomain != "" {
		customDomain = strings.TrimSuffix(
			customDomain,
			"/",
		)

		if !strings.HasPrefix(customDomain, "http://") &&
			!strings.HasPrefix(customDomain, "https://") {
			customDomain = "https://" + customDomain
		}
	}

	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		const defaultDirPerm = 0755

		if err := os.MkdirAll(
			scanDir,
			defaultDirPerm,
		); err != nil {
			log.Fatalf(
				"Failed to create scan-dir (%s): %v",
				scanDir,
				err,
			)
		}
	}

	return Options{
		Slug:          slug,
		Selection:     selection,
		Source:        sourceName,
		ScanDir:       scanDir,
		Cleanup:       !keepImages,
		CustomDomain:  customDomain,
		Debug:         debug,
		EbookFriendly: ebookFriendly,
	}
}
