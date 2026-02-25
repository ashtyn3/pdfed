package cmd

import (
"fmt"
"os"
"sort"
"strings"

lpdf "github.com/ledongthuc/pdf"
"github.com/pdfcpu/pdfcpu/pkg/api"
"github.com/sahilm/fuzzy"
"github.com/spf13/cobra"
)

var (
maxResults    int
noInteractive bool
searchThresh  int
)

var searchCmd = &cobra.Command{
Use:   "search <input.pdf> [query]",
Short: "Live fuzzy-search text across a PDF",
Long: fmt.Sprintf(`Interactively search for text across all pages of a PDF.

%s
  pdfed search document.pdf
  pdfed search document.pdf "introduction"
  pdfed search report.pdf --no-interactive -n 5

%s
  Type to filter in real time. Matched characters are highlighted.
  Tab switches to split mode without leaving the TUI.
  Ctrl+O opens the selected page in sioyek.`, bold("Examples:"), bold("Notes:")),
Args: cobra.RangeArgs(1, 2),
RunE: runSearch,
}

func init() {
rootCmd.AddCommand(searchCmd)
searchCmd.Flags().IntVarP(&maxResults, "max-results", "n", 20, "Max results (non-interactive mode)")
searchCmd.Flags().IntVarP(&searchThresh, "threshold", "t", 30, "Min match score 0-100 (non-interactive)")
searchCmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Print results and exit without TUI")
}

func runSearch(cmd *cobra.Command, args []string) error {
inputFile := args[0]
var initialQuery string
if len(args) > 1 {
initialQuery = args[1]
}

if _, err := os.Stat(inputFile); os.IsNotExist(err) {
return fmt.Errorf("input file not found: %s", inputFile)
}
if !strings.HasSuffix(strings.ToLower(inputFile), ".pdf") {
return fmt.Errorf("input file must be a PDF: %s", inputFile)
}

printInfo(fmt.Sprintf("Loading text from %s...", inputFile))
allLines, err := loadLines(inputFile)
if err != nil {
return fmt.Errorf("failed to extract text: %w", err)
}
if len(allLines) == 0 {
return fmt.Errorf("no extractable text found in %s", inputFile)
}

if noInteractive {
return staticSearch(allLines, initialQuery)
}

pageCount, _ := api.PageCountFile(inputFile)
return runAppTUI(inputFile, pageCount, allLines, modeSearch, initialQuery)
}

// ── non-interactive fallback ───────────────────────────────────────────────────

func staticSearch(allLines []indexedLine, query string) error {
if query == "" {
return fmt.Errorf("provide a query or remove --no-interactive")
}
strs := lineStrings(allLines)
matches := fuzzy.Find(query, strs)
count := 0
for _, m := range matches {
score := scorePercent(m.Score, len(query))
if score < searchThresh {
continue
}
line := allLines[m.Index]
scoreCol := green
if score < 60 {
scoreCol = yellow
}
fmt.Printf("  %s %s  %s\n",
cyan(fmt.Sprintf("p.%-4d", line.page)),
scoreCol(fmt.Sprintf("[%3d%%]", score)),
buildHighlightedLipgloss(line.text, m.MatchedIndexes),
)
count++
if count >= maxResults {
break
}
}
if count == 0 {
printWarning(fmt.Sprintf("No matches for %q", query))
}
return nil
}

// ── text extraction helpers ────────────────────────────────────────────────────

// indexedLine is a line of text tagged with its 1-based page number.
type indexedLine struct {
page int
text string
}

func loadLines(filename string) ([]indexedLine, error) {
pageTexts, err := extractTextByPage(filename)
if err != nil {
return nil, err
}
pages := make([]int, 0, len(pageTexts))
for p := range pageTexts {
pages = append(pages, p)
}
sort.Ints(pages)

var lines []indexedLine
for _, page := range pages {
for _, l := range strings.Split(pageTexts[page], "\n") {
l = strings.TrimSpace(l)
if len(l) >= 2 {
lines = append(lines, indexedLine{page: page, text: l})
}
}
}
return lines, nil
}

func lineStrings(lines []indexedLine) []string {
s := make([]string, len(lines))
for i, l := range lines {
s[i] = l.text
}
return s
}

func buildHighlightedLipgloss(text string, matchedIdxs []int) string {
if len(matchedIdxs) == 0 {
return text
}
matchSet := make(map[int]bool, len(matchedIdxs))
for _, idx := range matchedIdxs {
matchSet[idx] = true
}
runes := []rune(text)
var b strings.Builder
for i, r := range runes {
if matchSet[i] {
b.WriteString(matchStyle.Render(string(r)))
} else {
b.WriteRune(r)
}
}
return b.String()
}

func scorePercent(rawScore, queryLen int) int {
if queryLen == 0 {
return 0
}
worst := -(queryLen * queryLen)
if rawScore <= worst {
return 0
}
pct := int(float64(rawScore-worst) / float64(-worst) * 100)
if pct > 100 {
pct = 100
}
return pct
}

func extractTextByPage(filename string) (map[int]string, error) {
f, r, err := lpdf.Open(filename)
if err != nil {
return nil, err
}
defer f.Close()

result := make(map[int]string, r.NumPage())
for i := 1; i <= r.NumPage(); i++ {
p := r.Page(i)
if p.V.IsNull() {
continue
}
text, err := p.GetPlainText(nil)
if err != nil {
continue
}
result[i] = text
}
return result, nil
}
