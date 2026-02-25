package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/fatih/color"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

type sioyekMsg struct{ err error }

func openInSioyek(filename string, page int) tea.Cmd {
	return func() tea.Msg {
		abs, err := filepath.Abs(filename)
		if err != nil {
			abs = filename
		}
		binary, err := exec.LookPath("sioyek")
		if err != nil {
			binary = "/Applications/sioyek.app/Contents/MacOS/sioyek"
		}
		cmd := exec.Command(binary, "--reuse-window", "--page", strconv.Itoa(page), abs)
		// Detach from our process so sioyek survives after pdfed exits.
		cmd.Stdout = nil
		cmd.Stderr = nil
		err = cmd.Start()
		if err != nil {
			return sioyekMsg{err}
		}
		// Don't wait — fire and forget.
		go func() { _ = cmd.Wait() }()
		return sioyekMsg{}
	}
}

func pdfConfig() *model.Configuration {
	return model.NewDefaultConfiguration()
}

var version = "0.2.0"

var quiet bool

var (
	bold    = color.New(color.Bold).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
)

var rootCmd = &cobra.Command{
	Use:   "pdfed",
	Short: "A fast, free PDF utility for splitting and merging PDFs",
	Long: fmt.Sprintf(`
%s - A fast, free PDF utility

%s
  • %s    Extract pages or page ranges from a PDF
  • %s    Combine multiple PDFs into one
  • %s     Show PDF metadata and properties
  • %s    Rotate pages (90, 180, 270°)
  • %s  Compress and reduce file size
  • %s   Password-protect a PDF
  • %s    Remove password protection
  • %s    Fuzzy-search text across a PDF

%s
  pdfed split input.pdf -p 1-5            Extract pages 1-5
  pdfed split input.pdf                   Interactive split TUI
  pdfed merge out.pdf a.pdf b.pdf         Merge PDFs
  pdfed info input.pdf                    Show metadata
  pdfed rotate input.pdf 90 -p 1-3       Rotate pages 1-3
  pdfed optimize input.pdf -o out.pdf     Compress PDF
  pdfed encrypt input.pdf --user-pw pass  Password-protect
  pdfed search input.pdf                  Fuzzy search TUI

%s https://github.com/pdfcpu/pdfcpu`,
		bold("pdfed"),
		bold("Commands:"),
		cyan("split"),
		cyan("merge"),
		cyan("info"),
		cyan("rotate"),
		cyan("optimize"),
		cyan("encrypt"),
		cyan("decrypt"),
		cyan("search"),
		bold("Examples:"),
		magenta("Powered by"),
	),
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, red("Error:"), err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s version %s\n", bold("pdfed"), cyan(version)))
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}

func printSuccess(msg string) {
	if !quiet {
		fmt.Println(green("✓"), msg)
	}
}

func printInfo(msg string) {
	if !quiet {
		fmt.Println(cyan("→"), msg)
	}
}

func printError(msg string) {
	fmt.Fprintln(os.Stderr, red("✗"), msg)
}

func printWarning(msg string) {
	if !quiet {
		fmt.Println(yellow("⚠"), msg)
	}
}

func printf(format string, args ...interface{}) {
	if !quiet {
		fmt.Printf(format, args...)
	}
}
