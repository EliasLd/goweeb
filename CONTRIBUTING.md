**English** | [Français](CONTRIBUTING_FR.md)

## Contributing

If you know a website that could be useful to other users, implementing a new provider is one of the best ways to contribute to the project.

### Adding a new provider

Each website-specific implementation lives under:

```text
internal/source/
```

For example:

```text
internal/source/
├── animesama/
├── mangafreak/
└── yourprovider/
```

A provider must implement the common provider interface:

```go
type Provider interface {
	Name() string

	Search(
		query string,
		log *logger.Logger,
	) ([]SearchResult, error)

	ListScanPaths(
		workURL string,
		log *logger.Logger,
	) ([]common.SelectableItem, error)

	ListEntries(
		workURL string,
		scanPath string,
		log *logger.Logger,
	) (Work, []Entry, error)

	GetPageImageURLs(
		entryURL string,
		log *logger.Logger,
	) ([]string, error)
}
```

The usual implementation workflow is:

```text
Search
  ↓
Select manga
  ↓
ListScanPaths
  ↓
Select scan/version
  ↓
ListEntries
  ↓
Select chapters
  ↓
GetPageImageURLs
  ↓
Download images
  ↓
PDF / ebook-friendly output
```

### 1. Implement manga search

`Search()` should search the provider's catalog and return results containing at least:

```go
SearchResult{
	Title: "...",
	URL:   "...",
}
```

The returned URL must identify the manga and be usable by the next provider methods.

If the website exposes a regular HTML search page, existing helpers under `internal/source/common` may already cover part of the parsing logic.

### 2. Implement scan/version discovery

`ListScanPaths()` should return the available versions of the selected manga.

Examples include:

```text
Scans
Colored
Black & White
English
French
```

If the provider only exposes one version, simply return one automatically selectable item.

For example:

```go
[]common.SelectableItem{
	{
		Label: "Your Provider",
		Value: workURL,
	},
}
```

The application automatically skips the selection screen when only one option exists.

### 3. Implement chapter discovery

`ListEntries()` must return information about the selected work and its downloadable entries.

For manga chapters, the returned `Work` should generally use:

```go
Work{
	Title: "Manga Name",
	Kind:  ItemChapter,
}
```

Each chapter should then be converted to an `Entry`:

```go
Entry{
	Number: 100,
	Label:  "Chapter 100",
	URL:    "https://example.com/chapter/100",
}
```

The generic application code handles chapter range filtering, so provider implementations should normally return the complete chapter list.

### 4. Implement page image discovery

`GetPageImageURLs()` receives an entry URL and must return the ordered list of images belonging to that chapter.

For example:

```go
[]string{
	"https://example.com/chapter/100/001.jpg",
	"https://example.com/chapter/100/002.jpg",
	"https://example.com/chapter/100/003.jpg",
}
```

How these URLs are discovered depends entirely on the website.

Some providers may expose sequential image URLs, while others may require parsing the chapter HTML.

The generic downloader handles the actual image download once the list has been returned.

### 5. Register the provider

Once implemented, add the provider to the central provider registry under `internal/source`.

The registry is also used by the CLI and TUI to determine which providers are available, so providers should not be hardcoded separately in the user interfaces.

A registration generally includes:

```go
"yourprovider": {
	Label: "Your Provider",
	New: func(customDomain string) sourcetypes.Provider {
		return yourprovider.New(customDomain)
	},
},
```

After registration, the provider should automatically become available to both:

```text
goweeb --source yourprovider ...
```

and the provider selection screen in the TUI.

### 6. Test the complete workflow

Before opening a pull request, verify at least:

```text
Search
→ manga selection
→ scan/version selection
→ chapter discovery
→ range filtering
→ page discovery
→ image download
→ final output
```

Also test situations where:

* the search returns no results;
* the search returns only one result;
* the search returns multiple results;
* only one scan/version exists;
* the website returns an HTTP error;
* a chapter contains no images.

Finally, run:

```bash
gofmt -w .
go test ./...
```

and make sure the project builds successfully.

### Provider implementation recommendations

When adding a scraper:

* Keep website-specific parsing inside its provider package.
* Reuse helpers from `internal/source/common` and `internal/fetch` whenever possible.
* Avoid duplicating generic download or selection logic.
* Do not hardcode providers inside the CLI or TUI.
* Prefer URLs extracted directly from the website over reconstructing them when possible.
* Return clear errors when the website structure no longer matches what the scraper expects.
* Keep custom domain support whenever the provider can reasonably work with mirrors or domain changes.

Website structures change frequently, so keeping each scraper isolated makes maintenance much easier.
