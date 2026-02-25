package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	forceOverwrite bool
	mergeDryRun    bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge <output.pdf> <input1.pdf> <input2.pdf|dir> [input3.pdf...]",
	Short: "Merge multiple PDFs into a single file",
	Long: fmt.Sprintf(`Merge multiple PDF files into a single output file.

%s
  pdfed merge combined.pdf file1.pdf file2.pdf
  pdfed merge output.pdf *.pdf
  pdfed merge result.pdf a.pdf b.pdf c.pdf -f
  pdfed merge out.pdf ./chapters/

%s
  Files are merged in the order specified.
  Pass a directory to merge all PDFs inside it (sorted by name).
  Use -f to overwrite existing output files.`, bold("Examples:"), bold("Notes:")),
	Args: cobra.MinimumNArgs(2),
	RunE: runMerge,
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().BoolVarP(&forceOverwrite, "force", "f", false, "Overwrite output file if it exists")
	mergeCmd.Flags().BoolVarP(&mergeDryRun, "dry-run", "n", false, "Preview which files would be merged without writing")
}

func runMerge(cmd *cobra.Command, args []string) error {
	outputFile := args[0]
	inputArgs := args[1:]

	if !strings.HasSuffix(strings.ToLower(outputFile), ".pdf") {
		outputFile += ".pdf"
	}

	if _, err := os.Stat(outputFile); err == nil && !forceOverwrite && !mergeDryRun {
		return fmt.Errorf("output file already exists: %s (use -f to overwrite)", outputFile)
	}

	if mergeDryRun {
		printWarning("Dry run — no files will be written")
	}

	// Expand any directory arguments into sorted PDF file lists.
	var inputFiles []string
	for _, arg := range inputArgs {
		info, err := os.Stat(arg)
		if os.IsNotExist(err) {
			return fmt.Errorf("input not found: %s", arg)
		}
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", arg, err)
		}
		if info.IsDir() {
			entries, err := os.ReadDir(arg)
			if err != nil {
				return fmt.Errorf("failed to read directory %s: %w", arg, err)
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
					inputFiles = append(inputFiles, filepath.Join(arg, e.Name()))
				}
			}
		} else {
			inputFiles = append(inputFiles, arg)
		}
	}

	validInputs := make([]string, 0, len(inputFiles))
	totalPages := 0

	for _, f := range inputFiles {
		if !strings.HasSuffix(strings.ToLower(f), ".pdf") {
			printWarning(fmt.Sprintf("Skipping non-PDF file: %s", f))
			continue
		}

		pageCount, err := api.PageCountFile(f)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f, err)
		}

		validInputs = append(validInputs, f)
		totalPages += pageCount
		printf("  %s %s (%d pages)\n", cyan("•"), filepath.Base(f), pageCount)
	}

	if len(validInputs) < 2 {
		return fmt.Errorf("need at least 2 PDF files to merge")
	}

	printInfo(fmt.Sprintf("Merging %d files (%d total pages)...", len(validInputs), totalPages))

	if mergeDryRun {
		printInfo(fmt.Sprintf("Would create: %s", outputFile))
		return nil
	}

	var bar *progressbar.ProgressBar
	if !quiet {
		bar = progressbar.Default(int64(len(validInputs)))
	}

	if err := api.MergeCreateFile(validInputs, outputFile, false, pdfConfig()); err != nil {
		return fmt.Errorf("failed to merge PDFs: %w", err)
	}

	if bar != nil {
		_ = bar.Set(len(validInputs))
		_ = bar.Finish()
		fmt.Println()
	}

	outputInfo, err := os.Stat(outputFile)
	if err != nil {
		return fmt.Errorf("failed to get output file info: %w", err)
	}

	printSuccess(fmt.Sprintf("Created: %s (%s)", outputFile, formatFileSize(outputInfo.Size())))
	return nil
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
