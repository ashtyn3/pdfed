package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <input.pdf>",
	Short: "Show PDF metadata and properties",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInfo(args[0])
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(inFile string) error {
	printInfo(fmt.Sprintf("Reading %s…", inFile))

	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := api.PDFInfo(f, inFile, nil, false, pdfConfig())
	if err != nil {
		return err
	}

	fi, err := os.Stat(inFile)
	if err != nil {
		return err
	}

	if jsonOut {
		return emitInfoJSON(inFile, fi, info)
	}

	row := func(label, value string) {
		if value != "" {
			fmt.Printf("  %s%-18s%s\n", cyan(label+":"), strings.Repeat(" ", max(0, 18-len(label+":")+1)), value)
		}
	}
	rowf := func(label, format string, args ...interface{}) {
		row(label, fmt.Sprintf(format, args...))
	}

	fmt.Println()
	fmt.Println(bold(" Document"))
	fmt.Println(strings.Repeat("─", 50))
	row("File", inFile)
	rowf("Size", "%s", humanSize(fi.Size()))
	rowf("Version", "PDF %s", info.Version)
	rowf("Pages", "%d", info.PageCount)

	if len(info.Dimensions) > 0 {
		d := info.Dimensions[0]
		rowf("Page size", "%.1f × %.1f pt  (%.1f × %.1f mm)",
			d.Width, d.Height,
			d.Width*0.352778, d.Height*0.352778)
	}

	fmt.Println()
	fmt.Println(bold(" Metadata"))
	fmt.Println(strings.Repeat("─", 50))
	row("Title", info.Title)
	row("Author", info.Author)
	row("Subject", info.Subject)
	row("Creator", info.Creator)
	row("Producer", info.Producer)
	row("Created", info.CreationDate)
	row("Modified", info.ModificationDate)
	if len(info.Keywords) > 0 {
		row("Keywords", strings.Join(info.Keywords, ", "))
	}

	fmt.Println()
	fmt.Println(bold(" Features"))
	fmt.Println(strings.Repeat("─", 50))
	feature := func(label string, on bool) {
		mark := dimStyle.Render("✗")
		if on {
			mark = green("✓")
		}
		fmt.Printf("  %s  %s\n", mark, label)
	}
	feature("Encrypted", info.Encrypted)
	feature("Linearized (web optimized)", info.Linearized)
	feature("Tagged", info.Tagged)
	feature("Watermarked", info.Watermarked)
	feature("Bookmarks/Outlines", info.Outlines)
	feature("Form fields", info.Form)
	feature("Signatures", info.Signatures)
	feature("Attachments", len(info.Attachments) > 0)
	fmt.Println()
	return nil
}

func emitInfoJSON(path string, fi os.FileInfo, info *pdfcpu.PDFInfo) error {
	doc := map[string]interface{}{
		"file":        path,
		"size_bytes":  fi.Size(),
		"size_human":  humanSize(fi.Size()),
		"pdf_version": info.Version,
		"page_count":  info.PageCount,
	}
	if len(info.Dimensions) > 0 {
		d := info.Dimensions[0]
		doc["page_width_pt"] = d.Width
		doc["page_height_pt"] = d.Height
		doc["page_width_mm"] = d.Width * 0.352778
		doc["page_height_mm"] = d.Height * 0.352778
	}
	meta := map[string]interface{}{
		"title":      info.Title,
		"author":     info.Author,
		"subject":    info.Subject,
		"creator":    info.Creator,
		"producer":   info.Producer,
		"created":    info.CreationDate,
		"modified":   info.ModificationDate,
		"keywords":   info.Keywords,
	}
	features := map[string]interface{}{
		"encrypted":       info.Encrypted,
		"linearized":      info.Linearized,
		"tagged":          info.Tagged,
		"watermarked":     info.Watermarked,
		"outlines":        info.Outlines,
		"form_fields":     info.Form,
		"signatures":      info.Signatures,
		"has_attachments": len(info.Attachments) > 0,
	}
	return jsonResultOK("info", map[string]interface{}{
		"input":    path,
		"document": doc,
		"metadata": meta,
		"features": features,
	})
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
