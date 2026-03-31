package cmd

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/spf13/cobra"
)

var (
	optimizeOutput string
	optimizeDryRun bool
)

var optimizeCmd = &cobra.Command{
	Use:   "optimize <input.pdf>",
	Short: "Compress and optimize a PDF to reduce file size",
	Long: `Removes redundant objects, compresses streams, and deduplicates resources.
Without -o, the optimized file replaces the original.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOptimize(args[0])
	},
}

func init() {
	optimizeCmd.Flags().StringVarP(&optimizeOutput, "output", "o", "", "Output file (default: in-place)")
	optimizeCmd.Flags().BoolVarP(&optimizeDryRun, "dry-run", "n", false, "Preview without writing")
	rootCmd.AddCommand(optimizeCmd)
}

func runOptimize(inFile string) error {
	fi, err := os.Stat(inFile)
	if err != nil {
		return err
	}
	origSize := fi.Size()

	printInfo(fmt.Sprintf("Optimizing %s (%s)…", inFile, humanSize(origSize)))

	if optimizeDryRun {
		printInfo("[dry-run] would optimize (no output written)")
		if jsonOut {
			out := optimizeOutput
			if out == "" {
				out = inFile
			}
			return jsonResultOK("optimize", map[string]interface{}{
				"dry_run":           true,
				"input":             inFile,
				"output":            out,
				"input_size_bytes":  origSize,
				"input_size_human":  humanSize(origSize),
			})
		}
		return nil
	}

	outFile := optimizeOutput
	if outFile == "" {
		outFile = inFile // in-place
	}

	if err := api.OptimizeFile(inFile, outFile, pdfConfig()); err != nil {
		return err
	}

	fi2, err := os.Stat(outFile)
	if err != nil {
		if jsonOut {
			return fmt.Errorf("stat output after optimize: %w", err)
		}
		return nil
	}
	saved := origSize - fi2.Size()
	pct := float64(0)
	if origSize > 0 {
		pct = float64(saved) / float64(origSize) * 100
	}
	if saved > 0 {
		printSuccess(fmt.Sprintf("%s → %s (saved %s, %.1f%%)", humanSize(origSize), humanSize(fi2.Size()), humanSize(saved), pct))
	} else {
		printSuccess(fmt.Sprintf("%s → %s (already optimal)", humanSize(origSize), humanSize(fi2.Size())))
	}
	if jsonOut {
		outSz := fi2.Size()
		return jsonResultOK("optimize", map[string]interface{}{
			"input":             inFile,
			"output":            outFile,
			"input_size_bytes":  origSize,
			"output_size_bytes": outSz,
			"saved_bytes":       saved,
			"saved_percent":     pct,
			"input_size_human":  humanSize(origSize),
			"output_size_human": humanSize(outSz),
			"saved_human":       humanSize(abs64(saved)),
		})
	}
	return nil
}
