package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type pageLabelEntry struct {
	startIndex int    // 0-based physical page index where this label range begins
	style      string // "D"=decimal, "r"=lowercase roman, "R"=uppercase roman, "a"/"A"=alpha, ""=prefix only
	prefix     string
	startValue int    // numbering starts at this value (default 1)
}

// pageLabelsMap maps printed label string → list of 1-based physical page indices.
type pageLabelsMap map[string][]int

// readPageLabels opens a PDF and returns a map from printed page label → physical page indices (1-based).
// If the PDF has no PageLabels entry, every page's label equals its 1-based index as a string.
func readPageLabels(inputFile string) (pageLabelsMap, int, error) {
	f, err := os.Open(inputFile)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	ctx, err := api.ReadContext(f, model.NewDefaultConfiguration())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read PDF context: %w", err)
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return nil, 0, err
	}
	pageCount := ctx.PageCount

	entries, err := extractLabelEntries(ctx)
	if err != nil {
		return nil, pageCount, fmt.Errorf("failed to parse page labels: %w", err)
	}

	labels := generateLabels(entries, pageCount)

	m := make(pageLabelsMap, len(labels))
	for physIdx, label := range labels {
		physPage := physIdx + 1
		m[label] = append(m[label], physPage)
	}
	return m, pageCount, nil
}

// extractLabelEntries reads the PageLabels number tree from the catalog and returns sorted entries.
func extractLabelEntries(ctx *model.Context) ([]pageLabelEntry, error) {
	rootDict := ctx.RootDict
	if rootDict == nil {
		return nil, nil
	}

	obj, found := rootDict["PageLabels"]
	if !found {
		return nil, nil
	}

	obj, err := ctx.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
	}

	dict, ok := obj.(types.Dict)
	if !ok {
		return nil, fmt.Errorf("PageLabels is not a dict")
	}

	var entries []pageLabelEntry
	if err := walkNumberTree(ctx, dict, &entries); err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].startIndex < entries[j].startIndex
	})

	return entries, nil
}

// walkNumberTree recursively walks a PDF number tree node, collecting pageLabelEntry values.
func walkNumberTree(ctx *model.Context, node types.Dict, entries *[]pageLabelEntry) error {
	if nums := node.ArrayEntry("Nums"); nums != nil {
		for i := 0; i+1 < len(nums); i += 2 {
			keyObj, err := ctx.Dereference(nums[i])
			if err != nil {
				return err
			}
			key, ok := keyObj.(types.Integer)
			if !ok {
				continue
			}

			valObj, err := ctx.Dereference(nums[i+1])
			if err != nil {
				return err
			}

			entry := pageLabelEntry{startIndex: int(key), startValue: 1}

			if valDict, ok := valObj.(types.Dict); ok {
				if s := valDict.NameEntry("S"); s != nil {
					entry.style = *s
				}
				if p := valDict.StringEntry("P"); p != nil {
					entry.prefix = *p
				}
				if st := valDict.IntEntry("St"); st != nil {
					entry.startValue = *st
				}
			}

			*entries = append(*entries, entry)
		}
	}

	if kids := node.ArrayEntry("Kids"); kids != nil {
		for _, kidObj := range kids {
			kidObj, err := ctx.Dereference(kidObj)
			if err != nil {
				return err
			}
			kidDict, ok := kidObj.(types.Dict)
			if !ok {
				continue
			}
			if err := walkNumberTree(ctx, kidDict, entries); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateLabels produces a label string for each physical page (0-based index).
func generateLabels(entries []pageLabelEntry, pageCount int) []string {
	labels := make([]string, pageCount)

	if len(entries) == 0 {
		for i := range labels {
			labels[i] = fmt.Sprintf("%d", i+1)
		}
		return labels
	}

	for i := 0; i < pageCount; i++ {
		entry := findEntry(entries, i)
		offset := i - entry.startIndex
		num := entry.startValue + offset
		labels[i] = entry.prefix + formatNumber(num, entry.style)
	}

	return labels
}

func findEntry(entries []pageLabelEntry, pageIndex int) pageLabelEntry {
	result := entries[0]
	for _, e := range entries {
		if e.startIndex <= pageIndex {
			result = e
		} else {
			break
		}
	}
	return result
}

func formatNumber(n int, style string) string {
	switch style {
	case "D":
		return fmt.Sprintf("%d", n)
	case "r":
		return toRoman(n, false)
	case "R":
		return toRoman(n, true)
	case "a":
		return toAlpha(n, false)
	case "A":
		return toAlpha(n, true)
	default:
		return ""
	}
}

func toRoman(n int, upper bool) string {
	if n <= 0 {
		return fmt.Sprintf("%d", n)
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}

	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	if upper {
		return strings.ToUpper(b.String())
	}
	return b.String()
}

func toAlpha(n int, upper bool) string {
	if n <= 0 {
		return fmt.Sprintf("%d", n)
	}
	var b strings.Builder
	for n > 0 {
		n--
		b.WriteByte(byte('a' + n%26))
		n /= 26
	}
	s := reverseString(b.String())
	if upper {
		return strings.ToUpper(s)
	}
	return s
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// resolveRealPages takes a comma-separated range string of real page numbers and
// resolves them to physical page indices (1-based) using the PDF's page labels.
func resolveRealPages(rangeStr string, labelsMap pageLabelsMap, pageCount int) ([]int, error) {
	var result []int
	seen := make(map[int]bool)

	parts := strings.Split(rangeStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			startLabel := strings.TrimSpace(rangeParts[0])
			endLabel := strings.TrimSpace(rangeParts[1])

			startPages, ok := labelsMap[startLabel]
			if !ok {
				return nil, fmt.Errorf("page label %q not found in PDF", startLabel)
			}
			endPages, ok := labelsMap[endLabel]
			if !ok {
				return nil, fmt.Errorf("page label %q not found in PDF", endLabel)
			}

			startPhys := startPages[0]
			endPhys := endPages[len(endPages)-1]

			if startPhys > endPhys {
				return nil, fmt.Errorf("invalid range: page %q (PDF page %d) comes after page %q (PDF page %d)", startLabel, startPhys, endLabel, endPhys)
			}

			for i := startPhys; i <= endPhys; i++ {
				if !seen[i] {
					result = append(result, i)
					seen[i] = true
				}
			}
		} else {
			physPages, ok := labelsMap[part]
			if !ok {
				return nil, fmt.Errorf("page label %q not found in PDF", part)
			}
			for _, p := range physPages {
				if !seen[p] {
					result = append(result, p)
					seen[p] = true
				}
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid pages specified")
	}

	return result, nil
}
