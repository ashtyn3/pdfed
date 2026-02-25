package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/spf13/cobra"
)

var (
	rotateOutput string
	rotateDryRun bool
)

var rotateCmd = &cobra.Command{
	Use:   "rotate <input.pdf> <degrees>",
	Short: "Rotate pages in a PDF (90, 180, 270)",
	Long: `Rotate pages clockwise by the specified degrees (must be a multiple of 90).
Without -o, the file is rotated in-place.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		deg, err := strconv.Atoi(args[1])
		if err != nil || deg%90 != 0 {
			return fmt.Errorf("degrees must be a multiple of 90 (e.g. 90, 180, 270)")
		}
		return runRotate(args[0], deg)
	},
}

var rotatePages string

func init() {
	rotateCmd.Flags().StringVarP(&rotatePages, "pages", "p", "", "Page ranges to rotate (e.g. 1-3,5)")
	rotateCmd.Flags().StringVarP(&rotateOutput, "output", "o", "", "Output file (default: in-place)")
	rotateCmd.Flags().BoolVarP(&rotateDryRun, "dry-run", "n", false, "Preview without writing")
	rootCmd.AddCommand(rotateCmd)
}

func runRotate(inFile string, degrees int) error {
	var pages []string
	if rotatePages != "" {
		pages = strings.Split(rotatePages, ",")
	}

	pageDesc := "all pages"
	if rotatePages != "" {
		pageDesc = "pages " + rotatePages
	}
	printInfo(fmt.Sprintf("Rotating %s by %d° in %s…", pageDesc, degrees, inFile))

	if rotateDryRun {
		printInfo(fmt.Sprintf("[dry-run] would rotate %s by %d°", pageDesc, degrees))
		return nil
	}

	outFile := rotateOutput
	if outFile == "" {
		outFile = inFile // in-place
	}

	// pdfcpu RotateFile with empty outFile rotates in-place
	if err := api.RotateFile(inFile, outFile, degrees, pages, pdfConfig()); err != nil {
		return err
	}

	fi, _ := os.Stat(outFile)
	size := ""
	if fi != nil {
		size = fmt.Sprintf(" (%s)", humanSize(fi.Size()))
	}
	if outFile == inFile {
		printSuccess(fmt.Sprintf("Rotated in-place: %s%s", outFile, size))
	} else {
		printSuccess(fmt.Sprintf("Created: %s%s", outFile, size))
	}
	return nil
}
