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
		return nil
	}

	outFile := optimizeOutput
	if outFile == "" {
		outFile = inFile // in-place
	}

	if err := api.OptimizeFile(inFile, outFile, pdfConfig()); err != nil {
		return err
	}

	fi2, _ := os.Stat(outFile)
	if fi2 != nil {
		saved := origSize - fi2.Size()
		pct := float64(saved) / float64(origSize) * 100
		if saved > 0 {
			printSuccess(fmt.Sprintf("%s → %s (saved %s, %.1f%%)", humanSize(origSize), humanSize(fi2.Size()), humanSize(saved), pct))
		} else {
			printSuccess(fmt.Sprintf("%s → %s (already optimal)", humanSize(origSize), humanSize(fi2.Size())))
		}
	}
	return nil
}
