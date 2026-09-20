
**English** | [Français](README_FR.md)

# goweeb

![goweeb TUI demo](./assets/demo.gif)

**goweeb** is a fast manga downloader that can download scans from multiple websites and languages.

It's available both as an interactive terminal interface (TUI) and as a command-line tool (CLI).

Currently, it can download manga from the following websites:

| Provider   | Website                   |
| ---------- | ------------------------- |
| Weebcentral (English) | https://weebcentral.com |
| Mangadex (Multilingual) | https://mangadex.org |
| MangaVyvy (English) | https://mangavyvy.com |
| MangaFreak (English)| https://ww3.mangafreak.me |
| Anime-Sama (French only) | https://anime-sama.pw |

> [!NOTE]
> For **Mangadex**, goweeb currently only handles manga directly hosted at *mangadex.org*.
> External links are not handled for the moment, as it requires a dedicated scraping logic (I'm working on that).

> [!TIP]
> Support for more websites will be added over time.

## Installation

Installation is simple. Go to the **Releases** section of the GitHub repository and download the latest version of the tool for your operating system.

### Windows

1. Download the Windows archive.
2. Extract it to a folder of your choice.
3. Open a terminal (PowerShell or CMD) in that folder and run the tool.

### Linux / macOS

1. Download the archive corresponding to your operating system (`darwin` for macOS).
2. Make the binary executable:

```bash
chmod +x goweeb
```

3. Optionally, move it to `/usr/local/bin` to use it from anywhere:

```bash
sudo mv goweeb /usr/local/bin/
```

## Usage

goweeb is available in two versions.

### Graphical Interface (TUI) - Recommended for beginners

Simply launch the TUI executable:

```bash
# Linux / macOS
./goweeb

# Windows
goweeb.exe
```

Then let the interactive interface guide you through the process.

* Enter the manga title you are looking for.
* Select the provider you want to use.
* Optionally configure a custom domain for the selected provider.
* Configure your download options.
* Choose the manga from the search results if multiple results are found.
* Select a scan version when multiple versions are available.
* Choose which chapters you want to download.
* Start the download and follow its progress in real time.

### Command Line (CLI) - For advanced users

### Manga name format

The manga name should be entered **in quotes**, especially when it contains spaces.

You can use any search term supported by the selected provider, such as an English, French or Japanese title.

Examples:

```text
"one piece"
"jujutsu kaisen"
"chainsaw man"
"attack on titan"
```

If multiple results match your search, goweeb will let you choose the correct manga from an interactive list.

### Basic syntax

```bash
goweeb --source <provider> [options] "manga-name"
```

The `--source` option is required when using the CLI.

### Available options

| Option                | Shortcut | Description                                                                                                                                                                             |
| --------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--source <provider>` | none     | Select the website provider used to search for and download the manga. This option is required in CLI mode.                                                                             |
| `--all`               | `-a`     | Download all available chapters.                                                                                                                                                        |
| `--range <range>`     | `-r`     | Download a specific chapter or range without using the interactive chapter selection prompt. Optional and mainly useful when you already know which chapters you want.                  |
| `--scan-dir <folder>` | `-d`     | Folder where downloaded files will be saved. Default: `pdf`.                                                                                                                            |
| `--ebook-friendly`    | none     | Save chapters as images inside separate folders such as `Chapter 001/`, `Chapter 002/`, etc. Compatible with tools such as [Kindle Comic Converter](https://github.com/ciromattia/kcc). |
| `--domain <url>`      | `-u`     | Override the selected provider's default domain.                                                                                                                                        |
| `--debug`             | none     | Enable verbose debug logging. CLI only.                                                                                                                                                 |
| `--keep-images`       | `-k`     | Keep downloaded images after PDF creation.                                                                                                                                              |

Supported `--range` formats:

```text
10       # Chapter 10 only
1-10     # Chapters 1 through 10
10-      # Chapter 10 through the latest available chapter
-10      # Last 10 chapters
```

If neither `--all` nor `--range` is provided, goweeb will ask you interactively which chapters you want to download.

## Usage examples

### Download all chapters of a manga

Using Anime-Sama:

```bash
goweeb --source animesama --all "one piece"
```

Using MangaFreak:

```bash
goweeb --source mangafreak --all "jujutsu kaisen"
```

### Download a specific chapter

```bash
goweeb --source mangafreak --range 100 "jujutsu kaisen"
```

### Download a range of chapters

```bash
goweeb --source mangafreak --range 100-120 "jujutsu kaisen"
```

You can also use the shorthand:

```bash
goweeb --source mangafreak -r 100-120 "jujutsu kaisen"
```

If you do not specify `--range`, goweeb will ask you which chapters you want after the manga has been selected.

### Download the latest chapters

Download the last 10 chapters:

```bash
goweeb --source animesama --range -10 "one piece"
```

Download everything starting from chapter 100:

```bash
goweeb --source mangafreak --range 100- "jujutsu kaisen"
```

### Specify a destination folder

```bash
# Files will be saved in the "my-mangas" folder
goweeb --source animesama -d my-mangas --all "naruto"
```

Or with a full path:

```bash
goweeb --source mangafreak --scan-dir ~/Documents/Mangas "one piece"
```

On Windows:

```powershell
goweeb --source mangafreak -d "C:\Users\<your-username>\Documents\Mangas" "one piece"
```

### Specify a custom provider domain

The `--domain` option can be useful when a provider changes its domain or when you want to use a compatible mirror.

```bash
goweeb --source animesama \
  --domain https://example.com \
  --all \
  "one piece"
```

Or using the shorthand:

```bash
goweeb --source animesama \
  -u https://example.com \
  --all \
  "one piece"
```

### Download using an e-reader-friendly structure

> [!WARNING]
> Ebook-friendly mode does not generate EPUB files.
>
> It downloads manga pages into a folder structure that can then be processed by third-party software designed for e-readers.

For example, this mode works well with [Kindle Comic Converter](https://github.com/ciromattia/kcc), which can convert the downloaded chapters for Kindle, Kobo and other e-readers.

```bash
goweeb --source mangafreak \
  --all \
  --ebook-friendly \
  "one piece"
```

The resulting structure looks similar to:

```text
One Piece/
├── Chapter 001/
│   ├── 001.jpg
│   ├── 002.jpg
│   └── ...
├── Chapter 002/
│   ├── 001.jpg
│   └── ...
└── ...
```

### Combining options

```bash
goweeb \
  --source mangafreak \
  --range 1-50 \
  --scan-dir scans \
  --ebook-friendly \
  "one piece"
```

Example on Windows:

```powershell
goweeb \
  --source animesama \
  -d "C:\Users\<your-username>\scans" \
  -r 1-50 \
  "one piece"
```

## Tips

> [!TIP]
>
> * Search behavior depends on the selected provider.
> * Some providers may expose several versions of the same manga, such as color and black-and-white editions.
> * When only one manga or scan version is available, goweeb automatically selects it.
> * Use `--range` when you already know exactly which chapters you want to download.
> * If you are unsure which chapters are available, omit `--range` and use the interactive selection instead.
> * Use `--debug` when developing or troubleshooting a provider.
> * If a supported provider changes its domain but remains compatible with the existing scraper, you can override the default URL using the `--domain` / `-u` option or the custom domain field in the TUI.


## Contributing

Contributions are very welcome.

One of the main goals of goweeb’s architecture is to make it relatively easy to add support for new manga websites without having to modify the entire application.

**See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and instructions on how to add a new provider.**

## Thanks

If you like this tool, it would be very nice if you could leave a star ⭐. Thank you!
