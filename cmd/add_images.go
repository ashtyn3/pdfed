package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/spf13/cobra"
)

var (
	addImagesImportDesc string
	addImagesPaper      string
)

var addImagesCmd = &cobra.Command{
	Use:   "add-images <out.pdf> <image> [image...]",
	Short: "Append images as new PDF pages (or create a PDF from images)",
	Long: fmt.Sprintf(`Each image becomes one page appended to the PDF. If the PDF does not exist yet, a new file is created.

Supported image types: JPEG, PNG, WebP, TIFF.

%s
  pdfed add-images doc.pdf scan001.png
  pdfed add-images album.pdf a.jpg b.jpg c.png
  pdfed add-images letter.pdf photo.jpg --paper Letter
  pdfed add-images out.pdf pic.jpg -i "papersize:A4,position:full,scalefactor:0.5"

%s
  Use --paper for a quick page size (A4, Letter, A4L, …). For full control, pass pdfcpu import options with -i (see pdfcpu documentation for "import").`,
		bold("Examples:"),
		bold("Layout:")),
	Args: cobra.MinimumNArgs(2),
	RunE: runAddImages,
}

func init() {
	rootCmd.AddCommand(addImagesCmd)
	addImagesCmd.Flags().StringVarP(&addImagesImportDesc, "import", "i", "", `pdfcpu import options, comma-separated "key:value" pairs`)
	addImagesCmd.Flags().StringVar(&addImagesPaper, "paper", "", `Page size (e.g. A4, Letter, A4L); ignored if empty`)
}

func isImagePath(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".tif", ".tiff":
		return true
	default:
		return false
	}
}

func runAddImages(cmd *cobra.Command, args []string) error {
	outPDF := args[0]
	imgPaths := args[1:]

	if !strings.HasSuffix(strings.ToLower(outPDF), ".pdf") {
		outPDF += ".pdf"
	}

	for _, p := range imgPaths {
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("image not found: %s", p)
			}
			return fmt.Errorf("cannot access %s: %w", p, err)
		}
		if !isImagePath(p) {
			printWarning(fmt.Sprintf("Unusual extension for an image (proceeding anyway): %s", filepath.Base(p)))
		}
	}

	imp, err := buildImportConfig()
	if err != nil {
		return err
	}

	exists := false
	if st, err := os.Stat(outPDF); err == nil {
		exists = true
		if st.IsDir() {
			return fmt.Errorf("output path is a directory: %s", outPDF)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot access %s: %w", outPDF, err)
	}

	if exists {
		printInfo(fmt.Sprintf("Appending %d image page(s) to %s…", len(imgPaths), outPDF))
	} else {
		printInfo(fmt.Sprintf("Creating %s with %d page(s)…", outPDF, len(imgPaths)))
	}

	if err := api.ImportImagesFile(imgPaths, outPDF, imp, pdfConfig()); err != nil {
		return fmt.Errorf("add images: %w", err)
	}

	outInfo, err := os.Stat(outPDF)
	if err != nil {
		return fmt.Errorf("output file: %w", err)
	}
	printSuccess(fmt.Sprintf("Wrote: %s (%s)", outPDF, formatFileSize(outInfo.Size())))
	if jsonOut {
		return jsonResultOK("add-images", map[string]interface{}{
			"output":      outPDF,
			"image_count": len(imgPaths),
			"size_bytes":  outInfo.Size(),
			"size_human":  formatFileSize(outInfo.Size()),
			"appended":    exists,
		})
	}
	return nil
}

func buildImportConfig() (*pdfcpu.Import, error) {
	var imp *pdfcpu.Import
	if strings.TrimSpace(addImagesImportDesc) != "" {
		parsed, err := api.Import(addImagesImportDesc, types.POINTS)
		if err != nil {
			return nil, fmt.Errorf("import options: %w", err)
		}
		imp = parsed
	} else {
		imp = pdfcpu.DefaultImportConfig()
	}

	if strings.TrimSpace(addImagesPaper) != "" {
		dim, name, err := types.ParsePageFormat(addImagesPaper)
		if err != nil {
			return nil, fmt.Errorf("paper size: %w", err)
		}
		imp.PageDim = dim
		imp.PageSize = name
		imp.UserDim = true
	}

	return imp, nil
}
