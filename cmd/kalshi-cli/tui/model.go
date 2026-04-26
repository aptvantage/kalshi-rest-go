// Package tui implements the interactive terminal UI for kalshi-cli.
//
// Navigation hierarchy:
//
//Categories  →  Series (filtered)  →  Events  →  Markets  →  Orderbook
//
// Categories are derived client-side from the loaded series data, so there is
// no extra API call. The categories screen is the TUI root.
//
// Launch with tea.NewProgram(tui.New(client, env, authenticated), tea.WithAltScreen()).
package tui

import (
"sort"
"strings"

"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/table"
"github.com/charmbracelet/bubbles/textinput"
"github.com/charmbracelet/bubbles/viewport"
tea "github.com/charmbracelet/bubbletea"

"github.com/aptvantage/kalshi-rest-go/kalshi"
)

// Screen represents the currently visible pane.
type Screen int

const (
ScreenCategories  Screen = iota // root: browse categories derived from series
ScreenSeriesList                // series filtered by selected category (or all)
ScreenEventsList                // open events for a series
ScreenMarketsList               // markets inside an event
ScreenOrderbook                 // scrollable orderbook for a market
ScreenOrderEntry                // order entry form
)

// navEntry is one breadcrumb step.
type navEntry struct {
label  string
screen Screen
}

// categoryRow is a derived row built from loaded series data.
type categoryRow struct {
name        string   // category name, or "" for "All"
seriesCount int      // number of series in this category
tags        []string // unique tags across all series in this category
}

// orderForm holds TUI-level state for the order entry form.
type orderForm struct {
ticker     string
side       string // "yes" or "no"
action     string // "buy" or "sell"
countInput textinput.Model
priceInput textinput.Model
postOnly   bool
// focus: 0=side 1=action 2=count 3=price 4=postOnly 5=submit 6=cancel
focus      int
submitting bool
result     string
err        error
}

func newOrderForm(ticker string) orderForm {
count := textinput.New()
count.Placeholder = "1"
count.Width = 6
count.CharLimit = 5

price := textinput.New()
price.Placeholder = "50"
price.Width = 4
price.CharLimit = 2

return orderForm{
ticker:     ticker,
side:       "yes",
action:     "buy",
countInput: count,
priceInput: price,
postOnly:   true,
focus:      2, // start on count
}
}

// Model is the root Bubble Tea model. All mutable state lives here.
type Model struct {
// Infrastructure
client *kalshi.ClientWithResponses
env    string

// Authentication state. authFailed is set when verifyAuth (or any mid-session
// call) receives a 401; the TUI shows an error screen and disables all actions.
authFailed bool
authErr    error

// Navigation breadcrumb stack (bottom = root, top = current).
nav    []navEntry
screen Screen

// Account
balance *int64

// Raw data (retained so we can navigate back without re-fetching).
seriesData  []kalshi.Series
eventsData  []kalshi.EventData
marketsData []kalshi.Market

// Derived category rows (rebuilt after seriesData loads).
categoryRows []categoryRow

// Table models — one per hierarchy level.
categoriesTable table.Model
seriesTable     table.Model
eventsTable     table.Model
marketsTable    table.Model

// Orderbook viewport (scrollable text).
orderbookVP viewport.Model
obContent   string // pre-rendered orderbook text

// Client-side row filter (/ key).
filterInput   textinput.Model
filterMode    bool              // true = textinput is open and focused
screenFilters map[Screen]string // per-screen active filter query

// Order entry form.
orderForm orderForm

// Spinner shown during loading.
spinner spinner.Model
loading bool

// Terminal dimensions (set on tea.WindowSizeMsg).
width  int
height int

// Last error.
err error

// Context for refresh — which entity the current screen is showing.
selectedSeriesTicker string
selectedEventTicker  string
selectedMarketTicker string
}

// New creates the initial model. Authentication is verified via verifyAuth during
// Init(); the TUI will not load data until credentials are confirmed valid.
func New(client *kalshi.ClientWithResponses, env string) Model {
sp := spinner.New()
sp.Spinner = spinner.Dot
sp.Style = loadStyle

fi := textinput.New()
fi.Placeholder = "type to filter…"
fi.Width = 40
fi.CharLimit = 50
fi.PromptStyle = filterLabelStyle
fi.Prompt = "/ "

return Model{
client:      client,
env:         env,
nav:         []navEntry{{label: "categories", screen: ScreenCategories}},
screen:      ScreenCategories,
spinner:     sp,
filterInput:   fi,
screenFilters: make(map[Screen]string),
loading:       true,
}
}

// Init fires verifyAuth first; series and balance load only after auth succeeds.
func (m Model) Init() tea.Cmd {
return tea.Batch(verifyAuth(m.client), m.spinner.Tick)
}
// currentFilter returns the active filter query for the current screen.
func (m Model) currentFilter() string {
return m.screenFilters[m.screen]
}


// --- layout helpers ---

func (m Model) contentHeight() int {
if m.height < 4 {
return 1
}
return m.height - 3 // header(1) + status bar(1) + help bar(1)
}

func (m Model) contentWidth() int {
if m.width < 4 {
return 40
}
return m.width
}

// tableHeight returns the number of data rows the table should display.
// It shrinks by 1 when the filter bar is visible so the total render stays
// within the terminal height (header border adds an extra line).
func (m Model) tableHeight() int {
h := m.contentHeight() - 2
if m.filterMode || m.currentFilter() != "" {
h--
}
if h < 1 {
return 1
}
return h
}

// --- category derivation ---

// buildCategoryRows rebuilds categoryRows from the current seriesData.
// The first row is always "All" (no filter).
func (m *Model) buildCategoryRows() {
// Gather categories and their tags.
catMap := make(map[string]map[string]struct{})
catCount := make(map[string]int)
for _, s := range m.seriesData {
cat := s.Category
if catMap[cat] == nil {
catMap[cat] = make(map[string]struct{})
}
catCount[cat]++
for _, t := range s.Tags {
catMap[cat][t] = struct{}{}
}
}

// Sort categories alphabetically.
cats := make([]string, 0, len(catMap))
for c := range catMap {
cats = append(cats, c)
}
sort.Strings(cats)

rows := make([]categoryRow, 0, len(cats))
for _, c := range cats {
rows = append(rows, categoryRow{
name:        c,
seriesCount: catCount[c],
tags:        sortedKeys(catMap[c]),
})
}
m.categoryRows = rows
}

func sortedKeys(m map[string]struct{}) []string {
keys := make([]string, 0, len(m))
for k := range m {
keys = append(keys, k)
}
sort.Strings(keys)
return keys
}

// --- filter / table builders ---

// parseTerms splits a filter query on "|" into OR terms.
// Each term is trimmed and lowercased. Single-term queries return [query].
// e.g. "hockey | basketball" → ["hockey", "basketball"]
func parseTerms(q string) []string {
parts := strings.Split(q, "|")
terms := make([]string, 0, len(parts))
for _, p := range parts {
p = strings.TrimSpace(p)
if p != "" {
terms = append(terms, p)
}
}
return terms
}

// matchGlob matches pattern against value (both lowercased by caller).
// "*" matches everything; "foo*" matches values starting with "foo";
// otherwise a substring match is used.
func matchGlob(pattern, value string) bool {
if pattern == "*" || pattern == "" {
return true
}
if strings.HasSuffix(pattern, "*") {
return strings.HasPrefix(value, strings.TrimSuffix(pattern, "*"))
}
return strings.Contains(value, pattern)
}

// matchTerms returns true if any OR term matches the field map.
// Field map keys: "category", "tag", "ticker", "title", etc.
// Term syntax:
//   - "*"             → matches everything
//   - "hockey"        → substring match across all fields
//   - "hockey*"       → prefix match across all fields
//   - "category:ba*"  → match only the "category" field with pattern "ba*"
//   - "tag:nhl"       → match only tag fields containing "nhl"
func matchTerms(terms []string, fields map[string][]string) bool {
if len(terms) == 0 {
return true
}
for _, term := range terms {
term = strings.ToLower(strings.TrimSpace(term))
if term == "" || term == "*" {
return true
}
qualifier, pattern, hasQualifier := strings.Cut(term, ":")
if hasQualifier {
if vals, ok := fields[qualifier]; ok {
for _, v := range vals {
if matchGlob(pattern, strings.ToLower(v)) {
return true
}
}
}
} else {
for _, vals := range fields {
for _, v := range vals {
if matchGlob(term, strings.ToLower(v)) {
return true
}
}
}
}
}
return false
}

// applyFilter rebuilds the current screen's table rows filtered by the per-screen query.
//
// Filter syntax:
//   - "|"          OR: "hockey | basketball" matches either
//   - "*"          wildcard: "hockey*" prefix match; bare "*" matches all
//   - "category:X" restrict match to category name field only
//   - "tag:X"      restrict match to tag fields only
func (m *Model) applyFilter() {
q := strings.TrimSpace(m.screenFilters[m.screen])
terms := parseTerms(strings.ToLower(q))

// General terms (no field qualifier) are used to narrow the tags shown.
generalTerms := make([]string, 0, len(terms))
for _, t := range terms {
if !strings.Contains(t, ":") {
generalTerms = append(generalTerms, t)
}
}

switch m.screen {
case ScreenCategories:
filtered := make([]categoryRow, 0, len(m.categoryRows))
for _, r := range m.categoryRows {
catFields := map[string][]string{
"category": {r.name},
"tag":      r.tags,
}
if q != "" && q != "*" && !matchTerms(terms, catFields) {
continue
}
// Narrow displayed tags to those matching unqualified terms.
shownTags := r.tags
if len(generalTerms) > 0 {
shownTags = make([]string, 0, len(r.tags))
for _, tag := range r.tags {
if matchTerms(generalTerms, map[string][]string{"tag": {tag}}) {
shownTags = append(shownTags, tag)
}
}
}
filtered = append(filtered, categoryRow{
name:        r.name,
seriesCount: r.seriesCount,
tags:        shownTags,
})
}
m.initCategoriesTable(filtered)

case ScreenSeriesList:
parentQ := strings.TrimSpace(m.screenFilters[ScreenCategories])
parentTerms := parseTerms(strings.ToLower(parentQ))
filtered := make([]kalshi.Series, 0, len(m.seriesData))
for _, s := range m.seriesData {
fields := map[string][]string{
"category": {s.Category},
"tag":      s.Tags,
"ticker":   {s.Ticker},
"title":    {s.Title},
}
if parentQ != "" && parentQ != "*" && !matchTerms(parentTerms, fields) {
continue
}
if q != "" && q != "*" && !matchTerms(terms, fields) {
continue
}
filtered = append(filtered, s)
}
m.initSeriesTable(filtered)

case ScreenEventsList:
filtered := make([]kalshi.EventData, 0, len(m.eventsData))
for _, e := range m.eventsData {
fields := map[string][]string{
"ticker":   {e.EventTicker},
"subtitle": {e.SubTitle},
}
if q == "" || matchTerms(terms, fields) {
filtered = append(filtered, e)
}
}
m.initEventsTable(filtered)

case ScreenMarketsList:
filtered := make([]kalshi.Market, 0, len(m.marketsData))
for _, mkt := range m.marketsData {
fields := map[string][]string{
"ticker": {mkt.Ticker},
"status": {string(mkt.Status)},
}
if q == "" || matchTerms(terms, fields) {
filtered = append(filtered, mkt)
}
}
m.initMarketsTable(filtered)
}
}

// --- table initializers ---

func (m *Model) initCategoriesTable(rows []categoryRow) {
w := m.contentWidth()

// Size the category column to fit the longest name.
catW := 8
for _, r := range rows {
n := len(r.name)
if n == 0 {
n = 5 // "(all)"
}
if n > catW {
catW = n
}
}
if catW > 32 {
catW = 32
}
catW += 2 // breathing room

seriesW := 8
tagsW := w - catW - seriesW - 4
if tagsW < 20 {
tagsW = 20
}

cols := []table.Column{
{Title: "CATEGORY", Width: catW},
{Title: "SERIES", Width: seriesW},
{Title: "TAGS", Width: tagsW},
}

trows := make([]table.Row, 0, len(rows))
for _, r := range rows {
name := r.name
if name == "" {
name = "(all)"
}
lines := wrapTags(r.tags, tagsW)
if len(lines) == 0 {
lines = []string{""}
}
// First visual row carries the category name and series count.
trows = append(trows, table.Row{truncate(name, catW), formatInt(r.seriesCount), lines[0]})
// Continuation rows leave the first two cells blank.
for _, line := range lines[1:] {
trows = append(trows, table.Row{"", "", line})
}
}
t := table.New(
table.WithColumns(cols),
table.WithRows(trows),
table.WithFocused(true),
table.WithHeight(m.tableHeight()),
)
t.SetStyles(tableStyles())
m.categoriesTable = t
}

// wrapTags fits tags into lines of at most maxWidth characters, separated by ", ".
func wrapTags(tags []string, maxWidth int) []string {
if len(tags) == 0 {
return nil
}
var lines []string
current := ""
for _, tag := range tags {
switch {
case current == "":
current = tag
case len(current)+2+len(tag) <= maxWidth:
current += ", " + tag
default:
lines = append(lines, current)
current = tag
}
}
if current != "" {
lines = append(lines, current)
}
return lines
}

// wrapText wraps s into lines of at most maxWidth characters, breaking on word boundaries.
func wrapText(s string, maxWidth int) []string {
if maxWidth <= 0 {
return []string{s}
}
words := strings.Fields(s)
if len(words) == 0 {
return nil
}
var lines []string
current := ""
for _, w := range words {
switch {
case current == "":
current = w
case len(current)+1+len(w) <= maxWidth:
current += " " + w
default:
lines = append(lines, current)
current = w
}
}
if current != "" {
lines = append(lines, current)
}
return lines
}

func formatInt(n int) string {
return strings.TrimRight(strings.TrimRight(
func() string {
if n == 0 {
return "0"
}
result := ""
neg := n < 0
if neg {
n = -n
}
for n > 0 {
result = string(rune('0'+n%10)) + result
n /= 10
}
if neg {
result = "-" + result
}
return result
}(),
""), "")
}

func (m *Model) initSeriesTable(series []kalshi.Series) {
w := m.contentWidth()

// Dynamic widths: each fixed column sized to its widest content.
tickerW, catW, freqW := 6, 8, 4
for _, s := range series {
if l := len(s.Ticker); l > tickerW {
tickerW = l
}
if l := len(s.Category); l > catW {
catW = l
}
if l := len(s.Frequency); l > freqW {
freqW = l
}
}
tickerW = min(tickerW+2, 20)
catW = min(catW+2, 26)
freqW = min(freqW+2, 12)

// TAGS: sized to widest joined tag string, capped so TITLE still gets space.
tagsW := 6
for _, s := range series {
if l := len(strings.Join(s.Tags, ", ")); l > tagsW {
tagsW = l
}
}
if tagsW > 28 {
tagsW = 28
}

// TITLE fills remaining space.
const colSep = 5
titleW := w - tickerW - catW - freqW - tagsW - colSep
if titleW < 20 {
titleW = 20
}

cols := []table.Column{
{Title: "TICKER", Width: tickerW},
{Title: "TITLE", Width: titleW},
{Title: "CATEGORY", Width: catW},
{Title: "FREQ", Width: freqW},
{Title: "TAGS", Width: tagsW},
}

rows := make([]table.Row, 0, len(series))
for _, s := range series {
titleLines := wrapText(s.Title, titleW)
tagLines := wrapTags(s.Tags, tagsW)
nLines := len(titleLines)
if len(tagLines) > nLines {
nLines = len(tagLines)
}
if nLines == 0 {
nLines = 1
}
for i := 0; i < nLines; i++ {
var titleCell, tagCell string
if i < len(titleLines) {
titleCell = titleLines[i]
}
if i < len(tagLines) {
tagCell = tagLines[i]
}
if i == 0 {
rows = append(rows, table.Row{s.Ticker, titleCell, s.Category, s.Frequency, tagCell})
} else {
rows = append(rows, table.Row{"", titleCell, "", "", tagCell})
}
}
}

t := table.New(
table.WithColumns(cols),
table.WithRows(rows),
table.WithFocused(true),
table.WithHeight(m.tableHeight()),
)
t.SetStyles(tableStyles())
m.seriesTable = t
}

func (m *Model) initEventsTable(events []kalshi.EventData) {
w := m.contentWidth()
subW := w - 24 - 14 - 4
if subW < 20 {
subW = 20
}
cols := []table.Column{
{Title: "EVENT", Width: 24},
{Title: "SUBTITLE", Width: subW},
{Title: "STRIKE_DATE", Width: 12},
}
rows := make([]table.Row, 0, len(events))
for _, e := range events {
strikeDate := "-"
if e.StrikeDate != nil {
strikeDate = e.StrikeDate.UTC().Format("2006-01-02")
} else if e.StrikePeriod != nil {
strikeDate = *e.StrikePeriod
}
rows = append(rows, table.Row{
e.EventTicker,
truncate(e.SubTitle, subW),
strikeDate,
})
}
t := table.New(
table.WithColumns(cols),
table.WithRows(rows),
table.WithFocused(true),
table.WithHeight(m.tableHeight()),
)
t.SetStyles(tableStyles())
m.eventsTable = t
}

func (m *Model) initMarketsTable(markets []kalshi.Market) {
cols := []table.Column{
{Title: "TICKER", Width: 24},
{Title: "STATUS", Width: 8},
{Title: "YES_BID", Width: 8},
{Title: "YES_ASK", Width: 8},
{Title: "SPREAD", Width: 7},
{Title: "VOL_24H", Width: 10},
}
rows := make([]table.Row, 0, len(markets))
for _, mkt := range markets {
rows = append(rows, table.Row{
mkt.Ticker,
string(mkt.Status),
fmtCents(string(mkt.YesBidDollars)),
fmtCents(string(mkt.YesAskDollars)),
fmtSpread(string(mkt.YesBidDollars), string(mkt.YesAskDollars)),
mkt.Volume24hFp,
})
}
t := table.New(
table.WithColumns(cols),
table.WithRows(rows),
table.WithFocused(true),
table.WithHeight(m.tableHeight()),
)
t.SetStyles(tableStyles())
m.marketsTable = t
}
