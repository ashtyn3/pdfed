# pdfed

A fast, free PDF utility for splitting and merging PDFs. A lightweight alternative to commercial PDF tools.

## Features

- **Split PDFs** - Extract specific pages or page ranges
- **Merge PDFs** - Combine multiple PDFs into one
- **Fast** - Native Go binary, no runtime dependencies
- **Free** - Open source, no watermarks, no limits

## Installation

```bash
# Clone and build
git clone <repo-url>
cd pdfed
go build -o pdfed .

# Or install directly
go install .
```

## Usage

### Split PDF

Extract specific pages or page ranges from a PDF:

```bash
# Extract pages 1-5
pdfed split input.pdf -p 1-5

# Extract specific pages
pdfed split input.pdf -p 1,3,5,7

# Combine ranges and individual pages
pdfed split input.pdf -p 1-3,5,10-15

# Extract each page to a separate file
pdfed split input.pdf -e

# Specify output directory
pdfed split input.pdf -p 1-10 -o ./extracted
```

### Merge PDFs

Combine multiple PDF files into one:

```bash
# Merge two files
pdfed merge output.pdf file1.pdf file2.pdf

# Merge multiple files
pdfed merge combined.pdf chapter1.pdf chapter2.pdf chapter3.pdf

# Merge all PDFs in current directory
pdfed merge all.pdf *.pdf

# Overwrite existing output file
pdfed merge output.pdf a.pdf b.pdf -f
```

## Examples

```bash
# Split a book into chapters
pdfed split book.pdf -p 1-25 -o chapter1
pdfed split book.pdf -p 26-50 -o chapter2

# Extract just the cover page
pdfed split document.pdf -p 1

# Merge scanned pages
pdfed merge complete_scan.pdf page001.pdf page002.pdf page003.pdf

# Extract every page for review
pdfed split large_document.pdf -e -o ./pages
```

## Built With

- [pdfcpu](https://github.com/pdfcpu/pdfcpu) - PDF processing library (Apache 2.0)
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [color](https://github.com/fatih/color) - Terminal colors

## License

MIT License
