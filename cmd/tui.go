package cmd

import (
"fmt"
"os"
"path/filepath"
"sort"
"strconv"
"strings"

"github.com/charmbracelet/bubbles/textinput"
tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
"github.com/pdfcpu/pdfcpu/pkg/api"
"github.com/sahilm/fuzzy"
)

// ── mode ──────────────────────────────────────────────────────────────────────

type appMode int

const (
modeSearch appMode = iota
modeSplit
)

// ── model ─────────────────────────────────────────────────────────────────────

type appModel struct {
filename  string
baseName  string
allLines  []indexedLine
strs      []string
pageCount int
input     textinput.Model

// Vi modal state: false = normal mode, true = insert mode
insertMode bool

// Search state
matches      fuzzy.Matches
searchCursor int
searchOffset int
selected     *indexedLine

// Split state
splitPoints map[int]bool
currentPage int
outDir      string

// UI
mode      appMode
height    int
width     int
statusMsg string
splitting bool
}

type splitDoneMsg struct {
count int
err   error
}

// ── styles ────────────────────────────────────────────────────────────────────

var (
selStyle    = lipgloss.NewStyle().Reverse(true)
dimStyle    = lipgloss.NewStyle().Faint(true)
pageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
matchStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
headerStyle = lipgloss.NewStyle().Bold(true)
insertBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("-- INSERT --")
normalBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")).Render("-- NORMAL --")
segColors   = []lipgloss.Color{"4", "2", "5", "3", "1"}
)

// ── constructor ───────────────────────────────────────────────────────────────

func newAppModel(filename string, pageCount int, allLines []indexedLine, startMode appMode, initial string) appModel {
ti := textinput.New()
ti.Placeholder = "search…"
ti.Prompt = "  / "
ti.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
ti.SetValue(initial)
ti.Blur() // start in normal mode

base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
dir := output // global flag from split.go
if strings.HasSuffix(strings.ToLower(dir), ".pdf") {
dir = filepath.Dir(dir)
}
if allLines == nil {
allLines = []indexedLine{}
}

m := appModel{
input:       ti,
allLines:    allLines,
strs:        lineStrings(allLines),
pageCount:   pageCount,
splitPoints: make(map[int]bool),
currentPage: 1,
filename:    filename,
baseName:    base,
outDir:      dir,
mode:        startMode,
insertMode:  false,
}
if initial != "" {
m.matches = fuzzy.Find(initial, m.strs)
m.insertMode = true
m.input.Focus()
}
return m
}

// ── bubbletea interface ───────────────────────────────────────────────────────

func (m appModel) Init() tea.Cmd {
if m.insertMode {
return textinput.Blink
}
return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
switch msg := msg.(type) {

case tea.WindowSizeMsg:
m.height, m.width = msg.Height, msg.Width
return m, nil

case splitDoneMsg:
m.splitting = false
if msg.err != nil {
m.statusMsg = "error: " + msg.err.Error()
} else {
m.statusMsg = fmt.Sprintf("✓ created %d file(s)", msg.count)
}
return m, tea.Quit

case sioyekMsg:
if msg.err != nil {
m.statusMsg = "sioyek: " + msg.err.Error()
} else {
m.statusMsg = ""
}
return m, nil

case tea.MouseMsg:
if m.splitting {
return m, nil
}
return m.handleMouse(msg)

case tea.KeyMsg:
if m.splitting {
return m, nil
}
if m.insertMode {
return m.handleInsertKey(msg)
}
return m.handleNormalKey(msg)
}
return m, nil
}

// ── normal mode (vi command mode) ─────────────────────────────────────────────

func (m appModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
switch msg.Type {
case tea.KeyCtrlC:
return m, tea.Quit
case tea.KeyEsc:
return m, tea.Quit
case tea.KeyEnter:
return m.doEnter()
case tea.KeyTab:
return m.doTab()
case tea.KeyCtrlO:
return m.doSioyek()
case tea.KeyUp:
return m.doPrevResult()
case tea.KeyDown:
return m.doNextResult()
case tea.KeyLeft:
if m.mode == modeSplit {
return m.doPrevPage()
}
case tea.KeyRight:
if m.mode == modeSplit {
return m.doNextPage()
}
}

switch msg.String() {
case "q":
return m, tea.Quit
case "/":
m.insertMode = true
m.input.Focus()
return m, textinput.Blink
case "j":
return m.doNextResult()
case "k":
return m.doPrevResult()
case "o":
return m.doSioyek()
}

if m.mode == modeSplit {
switch msg.String() {
case "h":
return m.doPrevPage()
case "l":
return m.doNextPage()
case "g":
m.currentPage = 1
return m, nil
case "G":
m.currentPage = m.pageCount
return m, nil
case "x":
return m.doMarkSplit()
case "e":
if len(m.splitPoints) > 0 {
seg := m.currentSegment()
m.splitting = true
m.statusMsg = fmt.Sprintf("extracting p.%d–%d…", seg.start, seg.end)
return m, m.executeExtract(seg)
}
}
}
return m, nil
}

// ── insert mode (typing) ──────────────────────────────────────────────────────

func (m appModel) handleInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
switch msg.Type {
case tea.KeyCtrlC:
return m, tea.Quit
case tea.KeyEsc:
m.insertMode = false
m.input.Blur()
return m, nil
case tea.KeyTab:
m.insertMode = false
m.input.Blur()
return m.doTab()
case tea.KeyCtrlO:
return m.doSioyek()
case tea.KeyEnter:
return m.doEnter()
case tea.KeyUp, tea.KeyCtrlP:
return m.doPrevResult()
case tea.KeyDown, tea.KeyCtrlN:
return m.doNextResult()
}

var cmd tea.Cmd
m.input, cmd = m.input.Update(msg)
query := m.input.Value()
if query == "" {
m.matches = nil
m.searchCursor = 0
m.searchOffset = 0
} else {
prev := m.searchCursor
m.matches = fuzzy.Find(query, m.strs)
if prev >= len(m.matches) {
m.searchCursor = 0
m.searchOffset = 0
}
}
if m.mode == modeSplit && len(m.matches) > 0 {
m.currentPage = m.allLines[m.matches[m.searchCursor].Index].page
}
return m, cmd
}

// ── mouse ─────────────────────────────────────────────────────────────────────

func (m appModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
switch msg.Button {
case tea.MouseButtonWheelUp:
return m.doPrevResult()
case tea.MouseButtonWheelDown:
return m.doNextResult()
case tea.MouseButtonLeft:
if msg.Action != tea.MouseActionPress {
return m, nil
}
// Click on input row → enter insert mode
if msg.Y == m.height-2 {
if !m.insertMode {
m.insertMode = true
m.input.Focus()
return m, textinput.Blink
}
return m, nil
}
// Timeline click (split mode, always row 1 after header)
if m.mode == modeSplit && msg.Y == 1 {
if page := m.pageAtX(msg.X); page > 0 {
m.currentPage = page
}
return m, nil
}
// Results list click
// search: header(0) sep(1) results(2+)
// split:  header(0) timeline(1) blank(2) segs(3) files(4) blank(5) sep(6) results(7+)
resultsStart := 2
if m.mode == modeSplit {
resultsStart = 7
}
idx := msg.Y - resultsStart + m.searchOffset
if idx >= 0 && idx < len(m.matches) {
m.searchCursor = idx
if m.mode == modeSplit {
m.currentPage = m.allLines[m.matches[idx].Index].page
}
}
}
return m, nil
}

// pageAtX maps an X terminal column to a page number in the timeline row.
func (m appModel) pageAtX(x int) int {
splits := m.sortedSplits()
splitSet := make(map[int]bool, len(splits))
for _, s := range splits {
splitSet[s] = true
}
available := m.width - 4
splitChars := len(splits) * 3
pw := (available - splitChars) / m.pageCount
if pw > 4 {
pw = 4
}
if pw < 1 {
pw = 1
}
cur := 2 // leading "  "
for page := 1; page <= m.pageCount; page++ {
if page > 1 && splitSet[page] {
cur += 3
}
if x >= cur && x < cur+pw {
return page
}
cur += pw
}
return 0
}

// ── action helpers ────────────────────────────────────────────────────────────

func (m appModel) doEnter() (tea.Model, tea.Cmd) {
switch m.mode {
case modeSearch:
if len(m.matches) > 0 && m.searchCursor < len(m.matches) {
l := m.allLines[m.matches[m.searchCursor].Index]
m.selected = &l
}
return m, tea.Quit
case modeSplit:
if len(m.splitPoints) == 0 {
m.statusMsg = "⚠ no split points — x to mark"
return m, nil
}
m.splitting = true
m.statusMsg = "splitting all segments…"
return m, m.executeSplitAll()
}
return m, nil
}

func (m appModel) doTab() (tea.Model, tea.Cmd) {
if m.mode == modeSearch {
m.mode = modeSplit
} else {
m.mode = modeSearch
}
m.statusMsg = ""
return m, nil
}

func (m appModel) doSioyek() (tea.Model, tea.Cmd) {
page := m.activePage()
if page > 0 {
m.statusMsg = fmt.Sprintf("opening p.%d in sioyek…", page)
return m, openInSioyek(m.filename, page)
}
m.statusMsg = "no page selected"
return m, nil
}

func (m appModel) doPrevResult() (tea.Model, tea.Cmd) {
if m.searchCursor > 0 {
m.searchCursor--
if m.searchCursor < m.searchOffset {
m.searchOffset = m.searchCursor
}
if m.mode == modeSplit && len(m.matches) > 0 {
m.currentPage = m.allLines[m.matches[m.searchCursor].Index].page
}
}
return m, nil
}

func (m appModel) doNextResult() (tea.Model, tea.Cmd) {
if m.searchCursor < len(m.matches)-1 {
m.searchCursor++
vis := m.resultsVisible()
if m.searchCursor >= m.searchOffset+vis {
m.searchOffset = m.searchCursor - vis + 1
}
if m.mode == modeSplit && len(m.matches) > 0 {
m.currentPage = m.allLines[m.matches[m.searchCursor].Index].page
}
}
return m, nil
}

func (m appModel) doPrevPage() (tea.Model, tea.Cmd) {
if m.currentPage > 1 {
m.currentPage--
}
return m, nil
}

func (m appModel) doNextPage() (tea.Model, tea.Cmd) {
if m.currentPage < m.pageCount {
m.currentPage++
}
return m, nil
}

func (m appModel) doMarkSplit() (tea.Model, tea.Cmd) {
if m.currentPage > 1 {
if m.splitPoints[m.currentPage] {
delete(m.splitPoints, m.currentPage)
} else {
m.splitPoints[m.currentPage] = true
}
m.input.SetValue("")
m.matches = nil
m.searchCursor = 0
m.searchOffset = 0
}
return m, nil
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m appModel) View() string {
if m.height == 0 {
return ""
}
var sb strings.Builder
sb.WriteString(m.renderHeader() + "\n")
switch m.mode {
case modeSearch:
sb.WriteString(m.renderSearchPanel())
case modeSplit:
sb.WriteString(m.renderSplitPanel())
}
return sb.String()
}

func (m appModel) renderHeader() string {
info := ""
if m.mode == modeSplit && len(m.splitPoints) > 0 {
info = dimStyle.Render(fmt.Sprintf("  %d pages  %d split(s) → %d seg(s)", m.pageCount, len(m.splitPoints), len(m.splitPoints)+1))
} else {
info = dimStyle.Render(fmt.Sprintf("  %d pages", m.pageCount))
}
left := headerStyle.Render(" "+m.baseName) + info

tabs := renderModeTab("SEARCH", m.mode == modeSearch) +
"  " +
renderModeTab("SPLIT", m.mode == modeSplit)

pad := m.width - lipgloss.Width(left) - lipgloss.Width(tabs) - 2
if pad < 1 {
pad = 1
}
return left + strings.Repeat(" ", pad) + tabs
}

func renderModeTab(label string, active bool) string {
if active {
return lipgloss.NewStyle().Bold(true).
Foreground(lipgloss.Color("0")).
Background(lipgloss.Color("2")).
Render(" " + label + " ")
}
return dimStyle.Render("[" + label + "]")
}

func (m appModel) renderSearchPanel() string {
var sb strings.Builder
vis := m.resultsVisible()

sb.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

if len(m.matches) == 0 && m.input.Value() == "" {
sb.WriteString(dimStyle.Render("  / to search…") + "\n")
vis--
}
for i := m.searchOffset; i < m.searchOffset+vis; i++ {
if i >= len(m.matches) {
sb.WriteString("\n")
continue
}
sb.WriteString(m.renderResultRow(i) + "\n")
}

sb.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")
sb.WriteString(m.input.View() + "\n")
sb.WriteString(m.renderHints())
return sb.String()
}

func (m appModel) renderSplitPanel() string {
var sb strings.Builder

sb.WriteString(m.renderTimeline() + "\n\n")
sb.WriteString(m.renderSegments() + "\n\n")

sb.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

vis := m.resultsVisible()
if len(m.matches) == 0 && m.input.Value() == "" {
sb.WriteString(dimStyle.Render("  / search · h/l or click timeline to navigate") + "\n")
vis--
}
for i := m.searchOffset; i < m.searchOffset+vis; i++ {
if i >= len(m.matches) {
sb.WriteString("\n")
continue
}
sb.WriteString(m.renderResultRow(i) + "\n")
}

sb.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")
sb.WriteString(m.input.View() + "\n")
sb.WriteString(m.renderHints())
return sb.String()
}

func (m appModel) renderResultRow(i int) string {
match := m.matches[i]
line := m.allLines[match.Index]
pageStr := fmt.Sprintf("p.%-4d", line.page)
if i == m.searchCursor {
return selStyle.Render(fmt.Sprintf("▶ %-5s  %s", pageStr, line.text))
}
return fmt.Sprintf("  %s  %s", pageStyle.Render(pageStr), buildHighlightedLipgloss(line.text, match.MatchedIndexes))
}

func (m appModel) renderHints() string {
badge := normalBadge
if m.insertMode {
badge = insertBadge
}

if m.statusMsg != "" {
return badge + "  " + dimStyle.Render(m.statusMsg)
}

var hints string
if m.insertMode {
switch m.mode {
case modeSearch:
hints = "↑/↓ navigate · enter select · esc normal · tab → split"
case modeSplit:
hints = "↑/↓ jump to result · esc normal"
}
} else {
switch m.mode {
case modeSearch:
hints = "/ search · j/k navigate · enter select · o sioyek · tab → split · q quit"
case modeSplit:
hints = "/ search · h/l page · j/k results · x mark · e extract · enter split · g/G first/last · o sioyek · tab → search · q quit"
}
}
return badge + "  " + dimStyle.Render(hints)
}

func (m appModel) renderTimeline() string {
segs := m.buildSegments()
splits := m.sortedSplits()
splitSet := make(map[int]bool, len(splits))
for _, s := range splits {
splitSet[s] = true
}

pageSegment := make([]int, m.pageCount+1)
for si, seg := range segs {
for p := seg.start; p <= seg.end; p++ {
pageSegment[p] = si
}
}

available := m.width - 4
splitChars := len(splits) * 3
pw := (available - splitChars) / m.pageCount
switch {
case pw > 4:
pw = 4
case pw < 1:
pw = 1
}

var sb strings.Builder
sb.WriteString("  ")
for page := 1; page <= m.pageCount; page++ {
if page > 1 && splitSet[page] {
sb.WriteString(dimStyle.Render(" │ "))
}
color := segColors[pageSegment[page]%len(segColors)]
style := lipgloss.NewStyle().Foreground(color)
if page == m.currentPage {
style = style.Reverse(true).Bold(true)
}
var token string
if pw >= 2 {
token = fmt.Sprintf("%*d ", pw-1, page)
} else {
token = "▪"
}
sb.WriteString(style.Render(token))
}
if m.splitPoints[m.currentPage] {
sb.WriteString(dimStyle.Render("  ← remove"))
} else if m.currentPage > 1 {
sb.WriteString(dimStyle.Render("  ← split here"))
}
return sb.String()
}

func (m appModel) renderSegments() string {
segs := m.buildSegments()
activeSeg := m.currentSegment()

var segStrs, fileStrs []string
for i, seg := range segs {
color := segColors[i%len(segColors)]
label := fmt.Sprintf("p.%d–%d", seg.start, seg.end)
style := lipgloss.NewStyle().Foreground(color).Bold(true)
if seg == activeSeg {
style = style.Underline(true)
label = "▶ " + label
}
segStrs = append(segStrs, style.Render(label))
fileStrs = append(fileStrs, m.segmentFilename(seg))
}

segsLine := "  " + strings.Join(segStrs, dimStyle.Render("  ·  "))
filesLine := dimStyle.Render("  → " + strings.Join(fileStrs, "  ·  "))
if lipgloss.Width(filesLine) > m.width {
filesLine = filesLine[:m.width-1] + "…"
}
return segsLine + "\n" + filesLine
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (m appModel) resultsVisible() int {
var fixed int
switch m.mode {
case modeSearch:
fixed = 5 // header sep results sep input hints
case modeSplit:
fixed = 10 // header timeline blank segs files blank sep results sep input hints
}
v := m.height - fixed
if v < 0 {
return 0
}
return v
}

func (m appModel) activePage() int {
if m.mode == modeSplit {
return m.currentPage
}
if len(m.matches) > 0 && m.searchCursor < len(m.matches) {
return m.allLines[m.matches[m.searchCursor].Index].page
}
return 0
}

// ── segment helpers ───────────────────────────────────────────────────────────

type segment struct{ start, end int }

func (m appModel) buildSegments() []segment {
splits := m.sortedSplits()
segs := make([]segment, 0, len(splits)+1)
start := 1
for _, s := range splits {
segs = append(segs, segment{start, s - 1})
start = s
}
segs = append(segs, segment{start, m.pageCount})
return segs
}

func (m appModel) sortedSplits() []int {
splits := make([]int, 0, len(m.splitPoints))
for p := range m.splitPoints {
splits = append(splits, p)
}
sort.Ints(splits)
return splits
}

func (m appModel) currentSegment() segment {
for _, seg := range m.buildSegments() {
if m.currentPage >= seg.start && m.currentPage <= seg.end {
return seg
}
}
return segment{1, m.pageCount}
}

func (m appModel) segmentFilename(seg segment) string {
name := fmt.Sprintf("%s_p%d-%d.pdf", m.baseName, seg.start, seg.end)
if m.outDir != "" {
return filepath.Join(m.outDir, name)
}
return name
}

// ── async operations ──────────────────────────────────────────────────────────

func (m appModel) executeSplitAll() tea.Cmd {
segs := m.buildSegments()
filename, baseName, outDir := m.filename, m.baseName, m.outDir
return func() tea.Msg {
if outDir != "" {
if err := os.MkdirAll(outDir, 0755); err != nil {
return splitDoneMsg{err: err}
}
}
for _, seg := range segs {
outFile := fmt.Sprintf("%s_p%d-%d.pdf", baseName, seg.start, seg.end)
if outDir != "" {
outFile = filepath.Join(outDir, outFile)
}
sel := make([]string, 0, seg.end-seg.start+1)
for p := seg.start; p <= seg.end; p++ {
sel = append(sel, strconv.Itoa(p))
}
if err := api.CollectFile(filename, outFile, sel, pdfConfig()); err != nil {
return splitDoneMsg{err: err}
}
}
return splitDoneMsg{count: len(segs)}
}
}

func (m appModel) executeExtract(seg segment) tea.Cmd {
filename, baseName, outDir := m.filename, m.baseName, m.outDir
return func() tea.Msg {
if outDir != "" {
if err := os.MkdirAll(outDir, 0755); err != nil {
return splitDoneMsg{err: err}
}
}
outFile := fmt.Sprintf("%s_p%d-%d.pdf", baseName, seg.start, seg.end)
if outDir != "" {
outFile = filepath.Join(outDir, outFile)
}
sel := make([]string, 0, seg.end-seg.start+1)
for p := seg.start; p <= seg.end; p++ {
sel = append(sel, strconv.Itoa(p))
}
if err := api.CollectFile(filename, outFile, sel, pdfConfig()); err != nil {
return splitDoneMsg{err: err}
}
return splitDoneMsg{count: 1}
}
}

// ── entry point ───────────────────────────────────────────────────────────────

func runAppTUI(filename string, pageCount int, allLines []indexedLine, startMode appMode, initial string) error {
m := newAppModel(filename, pageCount, allLines, startMode, initial)
p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
result, err := p.Run()
if err != nil {
return err
}
if final, ok := result.(appModel); ok {
if final.selected != nil {
fmt.Printf("p.%d: %s\n", final.selected.page, final.selected.text)
}
if strings.HasPrefix(final.statusMsg, "✓") {
printSuccess(strings.TrimPrefix(final.statusMsg, "✓ "))
} else if strings.HasPrefix(final.statusMsg, "error") {
printError(final.statusMsg)
}
}
return nil
}
