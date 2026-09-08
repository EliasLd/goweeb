
**English** | [Français](README.md)

> [!NOTE]
> This utility is intended to dynamically download
> manga scans in **french** on a **french** website :)

> [!TIP]
> I'm currently working on a big refactoring that'll make this tool
> download manga from multiple sources in english as well!

# goweeb-sama

![goweeb TUI demo](./assets/demo.gif)

**goweeb-sama** is a fast tool to download manga scans from the French website **anime-sama** (which does an excellent job by the way, thanks to them).

## Installation

Installation is very simple, just go to the **Releases** tab on the right side of the GitHub interface and download the latest **TUI** version of the tool for your operating system.

### Windows
1. Download the Windows archive
2. Place it in a folder of your choice
3. Open a terminal (PowerShell or CMD) in that folder and use the tool

### Linux / macOS
1. Download the archive corresponding to your system (*darwin* for macos)
2. Make it executable: `chmod +x goweeb`
3. (Optional) Move it to `/usr/local/bin` to use it from anywhere

## Usage

**goweeb-sama** is available in two versions:

### Graphical Interface (TUI) - Recommended for beginners

Simply launch the TUI executable:

```bash
# Linux / macOS
./goweeb

# Windows
goweeb.exe
```

Let the interactive interface guide you!

- Enter the manga name you're searching for
- Choose from the search results
- Select the version (color/black and white, VF/VA)
- Configure your download options
- Start downloading and follow the progress in real-time

### Command Line (CLI) - For advanced users

### Manga name format


**Important**: The manga name must be entered **in quotes**. You can use any search term (French, English, or Japanese title).

Examples:
- `"one piece"`
- `"jujutsu kaisen"`
- `"chainsaw man"`
- `"attack on titan"`

If multiple results match your search, the tool will let you choose the correct manga from an interactive list.

### Basic syntax

```bash
goweeb [options] "manga-name"
```

### Available options

| Option | Shortcut | Description |
|--------|----------|-------------|
| `--all` | `-a` | Download all available chapters |
| `--scan-dir <folder>` | `-d` | Folder to save PDF files (default: `pdf`) |
| `--keep-images` | `-k` | Keep images after PDF creation |
| `--domain <url>` | `-u` | Override anime-sama domain (e.g., `https://anime-sama.tv`) |
| `--ebook-friendly ` | none | Download chapters as images in separated subfolders (e.g. Chapter 001/, Chapter 002/). Compatible with [Kindle Comic Converter](https://github.com/ciromattia/kcc). |
| `--debug ` | none | Enable debug mode to display more logs (CLI only) |

### Usage examples

#### Download all chapters of a manga
```bash
goweeb --all "one piece"
# or
goweeb -a "jujutsu kaisen"
```

#### Download a range of chapters

After selecting the manga you want to download, the program will **ask you** to enter a range of chapter that you want to download.
Just follow the instruction and choose the range that suits you.

#### Specify a destination folder
```bash
# PDFs will be saved in the "my-mangas" folder
goweeb -d my-mangas --all "naruto"

# Or with the full path
goweeb --scan-dir ~/Documents/Mangas "one piece"
```

#### Specify a custom domain
```bash
# If the anime-sama domain changes
goweeb -u https://anime-sama.fr --all "one piece"
```

#### Download as an e-reader friendly structure

> [!WARNING]
> It's not about downloading in EPUB format, but rather downloading manga
> following a certain structure used by third party softwares to format manga for e-ink readers.

For example, this mode is very usedful when it comes to convert the downloaded manga with [Kindle Comic Converter](https://github.com/ciromattia/kcc)
and then upload it to your kindle, kobo or other e-reader.

```bash
goweeb -u https://anime-sama.fr --all --ebook-friendly "one piece"
```

#### Combining options
```bash
# Download chapters 1 to 50 of One Piece, in the "scans" folder on Windows
# while specifying a custom anime-sama domain
goweeb -d 'C:\Users\<your-username>\scans' -u https://anime-sama.tv "one piece"
```

### Tips

> [!TIP]
> - The search uses anime-sama's search engine directly, so you can use any name (French, English, Japanese)
> - If multiple scan versions exist (e.g., color/black and white), the tool will ask which one to download
> - Don't forget the quotes around the manga name!

> [!WARNING] 
> Keep in mind that the anime-sama domain changes for obvious reasons... so don't hesitate to check [anime-sama.pw](https://anime-sama.pw) to verify which domain is active. You can then specify it with the `-u` argument as shown above.

---

## Credits

Thanks to Anime Sama and their huge work! This project wouldn't exist without them.
