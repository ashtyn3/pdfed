# pdfed

> It works, I just made it for school work. But beware this is all slop. Like I haven't looked at the code at all. 

A fast, free PDF utility with a live fuzzy-search TUI and interactive split editor. A lightweight alternative to commercial PDF tools.

## Features

- **Interactive TUI** — vi-modal interface with live fuzzy search, timeline-based split editor, and mouse support
- **Split PDFs** — extract pages/ranges non-interactively, or use the interactive timeline to mark splits visually
- **Merge PDFs** — combine multiple PDFs or an entire directory of PDFs into one
- **Info** — display metadata, page dimensions, and feature flags at a glance
- **Rotate** — rotate any page selection by 90 / 180 / 270°
- **Optimize** — compress and deduplicate objects to reduce file size
- **Encrypt / Decrypt** — password-protect or unlock PDFs
- **Search** — live fuzzy-search across all PDF text content
- **Sioyek integration** — open the current page in sioyek directly from the TUI
- **Fast** — native Go binary, no runtime dependencies

## Installation

```bash
git clone <repo-url>
cd pdfed
go build -o pdfed .

# or
go install .
```

## Commands

### `info` — PDF metadata

```bash
pdfed info input.pdf
```

Displays file size, PDF version, page count, page dimensions, title/author/creator/dates, and a feature checklist (encrypted, tagged, bookmarks, forms, etc.).

---

### `split` — Extract pages

```bash
# Interactive TUI (no flags → opens the split editor)
pdfed split input.pdf

# Extract a page range
pdfed split input.pdf -p 1-5

# Extract specific pages
pdfed split input.pdf -p 1,3,5,7

# Combine ranges
pdfed split input.pdf -p 1-3,5,10-15

# Extract each page to a separate file
pdfed split input.pdf -e

# Output to a directory
pdfed split input.pdf -p 1-10 -o ./extracted/

# Preview without writing
pdfed split input.pdf -p 1-5 --dry-run
```

#### Interactive split TUI

```
pdfed split input.pdf
```

The TUI opens in **NORMAL mode** (vi-style):

| Key | Action |
|-----|--------|
| `h` / `l` or `←` / `→` | Move timeline cursor |
| `j` / `k` or `↑` / `↓` | Navigate search results |
| `/` | Enter INSERT mode — type to fuzzy-search |
| `Esc` | Return to NORMAL mode |
| `x` | Mark / unmark split at current page |
| `e` | Extract only the current segment |
| `Enter` | Split all segments into files |
| `g` / `G` | Jump to first / last page |
| `o` | Open current page in sioyek |
| `Tab` | Switch between SEARCH and SPLIT panels |
| `q` / `Esc` | Quit |

Click the timeline to jump to a page. Scroll wheel navigates results. Click the input bar to enter INSERT mode.

---

### `merge` — Combine PDFs

```bash
# Merge files in order
pdfed merge output.pdf file1.pdf file2.pdf file3.pdf

# Merge all PDFs in a directory (sorted by name)
pdfed merge output.pdf ./scans/

# Preview without writing
pdfed merge output.pdf a.pdf b.pdf --dry-run
```

---

### `search` — Fuzzy search

```bash
# Interactive live-search TUI
pdfed search input.pdf

# Non-interactive (scripting)
pdfed search input.pdf "passive voice" --no-interactive
pdfed search input.pdf "passive voice" --no-interactive -n 5 -t 50
```

The search TUI uses the same vi-modal interface as split (`/` to type, `Esc` for NORMAL, `Enter` to print the selected result). `Tab` switches to the split panel without reloading.

---

### `rotate` — Rotate pages

```bash
# Rotate all pages 90° clockwise (in-place)
pdfed rotate input.pdf 90

# Rotate specific pages
pdfed rotate input.pdf 180 -p 1-3

# Write to a new file
pdfed rotate input.pdf 270 -o rotated.pdf

# Preview
pdfed rotate input.pdf 90 --dry-run
```

---

### `optimize` — Compress

```bash
# Optimize in-place
pdfed optimize input.pdf

# Write to a new file
pdfed optimize input.pdf -o smaller.pdf

# Preview
pdfed optimize input.pdf --dry-run
```

Reports before/after sizes and savings percentage.

---

### `encrypt` — Password-protect

```bash
# Encrypt with a user password
pdfed encrypt input.pdf --user-pw "secret"

# Separate user and owner passwords
pdfed encrypt input.pdf --user-pw "open" --owner-pw "edit" -o locked.pdf
```

---

### `decrypt` — Remove password

```bash
pdfed decrypt input.pdf --password "secret"
pdfed decrypt input.pdf --password "secret" -o unlocked.pdf
```

---

## Global flags

| Flag | Description |
|------|-------------|
| `-q` / `--quiet` | Suppress non-essential output |
| `--dry-run` / `-n` | Preview without writing (split, merge, rotate, optimize) |

## Built With

- [pdfcpu](https://github.com/pdfcpu/pdfcpu) — PDF processing library (Apache 2.0)
- [bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [cobra](https://github.com/spf13/cobra) — CLI framework
- [fuzzy](https://github.com/sahilm/fuzzy) — fuzzy matching
- [color](https://github.com/fatih/color) — terminal colors

## License

MIT License
