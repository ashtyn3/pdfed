package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	pages    string
	pdfPages string

	extractAll bool
	output     string
	dryRun     bool
)

var splitCmd = &cobra.Command{
	Use:   "split <input.pdf>",
	Short: "Split a PDF by extracting pages or page ranges",
	Long: fmt.Sprintf(`Split a PDF file by extracting specific pages or page ranges.

%s
  pdfed split input.pdf -p 1-5               Extract pages 1-5 to input_pages_1-5.pdf
  pdfed split input.pdf -p 1,3,5             Extract pages 1, 3, and 5
  pdfed split input.pdf -p 1-3,7,10-15       Extract multiple ranges
  pdfed split input.pdf -p 1-5 -o chap.pdf   Specify output filename
  pdfed split input.pdf -p 1-5 -o ./out      Specify output directory
  pdfed split input.pdf -P 1-5               Use raw PDF page indices (short form)
  pdfed split input.pdf --pdf-pages 1-5      Use raw PDF page indices (long form)
  pdfed split input.pdf -e                   Extract each page to separate files
  pdfed split input.pdf -e -o ./pages        Extract all to directory

%s
  Page ranges use the format: start-end (e.g., 1-5)
  Individual pages are comma-separated (e.g., 1,3,5)
  Combine both: 1-3,5,7-10

  Use -p/--pages for real (printed) page numbers.
  Use -P/--pdf-pages for raw PDF page indices (1-based).`, bold("Examples:"), bold("Page Syntax:")),
	Args: cobra.ExactArgs(1),
	RunE: runSplit,
}

func init() {
	rootCmd.AddCommand(splitCmd)
	splitCmd.Flags().StringVarP(&pages, "pages", "p", "", "Real (printed) page ranges to extract (e.g., 1-5,7,10-15)")
	splitCmd.Flags().StringVarP(&pdfPages, "pdf-pages", "P", "", "PDF page ranges to extract by raw index (1-based)")
	splitCmd.Flags().BoolVarP(&extractAll, "extract-all", "e", false, "Extract each page to a separate file")
	splitCmd.Flags().StringVarP(&output, "output", "o", "", "Output file (.pdf) or directory")
	splitCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview what would be extracted without writing files")
}

func runSplit(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputFile)
	}

	if !strings.HasSuffix(strings.ToLower(inputFile), ".pdf") {
		return fmt.Errorf("input file must be a PDF: %s", inputFile)
	}

	if pages != "" && pdfPages != "" {
		return fmt.Errorf("specify either -p/--pages or --pdf-pages, not both")
	}

	if (pages != "" || pdfPages != "") && extractAll {
		return fmt.Errorf("cannot use page selection flags (-p/--pages or --pdf-pages) together with -e")
	}

	pageCount, err := api.PageCountFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read PDF: %w", err)
	}

	// No page selection → open interactive TUI in split mode.
	if pages == "" && pdfPages == "" && !extractAll && !dryRun {
		printInfo("Loading text for search…")
		allLines, _ := loadLines(inputFile)
		return runAppTUI(inputFile, pageCount, allLines, modeSplit, "")
	}

	printInfo(fmt.Sprintf("Input: %s (%d pages)", filepath.Base(inputFile), pageCount))

	if dryRun {
		printWarning("Dry run — no files will be written")
	}

	if extractAll {
		return extractAllPages(inputFile, pageCount)
	}

	return extractPageRanges(inputFile, pageCount)
}

func extractAllPages(inputFile string, pageCount int) error {
	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))

	outDir := output
	if outDir == "" {
		outDir = "."
	}

	if strings.HasSuffix(strings.ToLower(outDir), ".pdf") {
		return fmt.Errorf("output must be a directory when using -e, not a file")
	}

	if dryRun {
		printInfo(fmt.Sprintf("Would extract %d pages to %s/", pageCount, outDir))
		for i := 1; i <= pageCount; i++ {
			printf("  %s %s_page_%03d.pdf\n", cyan("→"), baseName, i)
		}
		return nil
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	printInfo(fmt.Sprintf("Extracting %d pages to %s/...", pageCount, outDir))

	var bar *progressbar.ProgressBar
	if !quiet {
		bar = progressbar.Default(int64(pageCount))
	}

	for i := 1; i <= pageCount; i++ {
		outputFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%03d.pdf", baseName, i))
		pageSelection := []string{strconv.Itoa(i)}

		if err := api.CollectFile(inputFile, outputFile, pageSelection, pdfConfig()); err != nil {
			return fmt.Errorf("failed to extract page %d: %w", i, err)
		}

		if bar != nil {
			_ = bar.Add(1)
		}
	}

	if bar != nil {
		_ = bar.Finish()
		fmt.Println()
	}

	printSuccess(fmt.Sprintf("Extracted %d pages to %s/", pageCount, outDir))
	return nil
}

func extractPageRanges(inputFile string, pageCount int) error {
	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))

	var pageList []int
	var rangeStr string

	if pdfPages != "" {
		rangeStr = pdfPages
		var err error
		pageList, err = parsePageRanges(rangeStr, pageCount)
		if err != nil {
			return err
		}
	} else {
		rangeStr = pages
		labelsMap, _, err := readPageLabels(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read page labels: %w", err)
		}
		pageList, err = resolveRealPages(rangeStr, labelsMap, pageCount)
		if err != nil {
			return err
		}
	}

	var outputFile string
	if output == "" {
		sanitizedPages := strings.ReplaceAll(rangeStr, ",", "_")
		outputFile = fmt.Sprintf("%s_pages_%s.pdf", baseName, sanitizedPages)
	} else if strings.HasSuffix(strings.ToLower(output), ".pdf") {
		outputFile = output
	} else {
		if err := os.MkdirAll(output, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		sanitizedPages := strings.ReplaceAll(rangeStr, ",", "_")
		outputFile = filepath.Join(output, fmt.Sprintf("%s_pages_%s.pdf", baseName, sanitizedPages))
	}

	pageSelection := make([]string, 0, len(pageList))
	for _, p := range pageList {
		pageSelection = append(pageSelection, strconv.Itoa(p))
	}

	printInfo(fmt.Sprintf("Extracting %d pages (PDF pages %v)", len(pageList), pageList))

	if dryRun {
		sanitizedPages := strings.ReplaceAll(rangeStr, ",", "_")
		previewFile := outputFile
		if previewFile == "" {
			previewFile = fmt.Sprintf("%s_pages_%s.pdf", baseName, sanitizedPages)
		}
		printInfo(fmt.Sprintf("Would create: %s", previewFile))
		return nil
	}

	if err := api.CollectFile(inputFile, outputFile, pageSelection, pdfConfig()); err != nil {
		return fmt.Errorf("failed to extract pages: %w", err)
	}

	outputInfo, err := os.Stat(outputFile)
	if err != nil {
		return fmt.Errorf("failed to get output file info: %w", err)
	}

	printSuccess(fmt.Sprintf("Created: %s (%s)", outputFile, formatFileSize(outputInfo.Size())))
	return nil
}

func parsePageRanges(rangeStr string, maxPage int) ([]int, error) {
	var pages []int
	seen := make(map[int]bool)

	parts := strings.Split(rangeStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", rangeParts[1])
			}

			if start < 1 || end > maxPage {
				return nil, fmt.Errorf("page range %d-%d out of bounds (document has %d pages)", start, end, maxPage)
			}

			if start > end {
				return nil, fmt.Errorf("invalid range: start (%d) > end (%d)", start, end)
			}

			for i := start; i <= end; i++ {
				if !seen[i] {
					pages = append(pages, i)
					seen[i] = true
				}
			}
		} else {
			page, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", part)
			}

			if page < 1 || page > maxPage {
				return nil, fmt.Errorf("page %d out of bounds (document has %d pages)", page, maxPage)
			}

			if !seen[page] {
				pages = append(pages, page)
				seen[page] = true
			}
		}
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no valid pages specified")
	}

	return pages, nil
}
