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
client        *kalshi.ClientWithResponses
env           string
authenticated bool // false = no credentials; balance and order features disabled

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

// Active category filter ("" = all series visible).
categoryFilter string

// Table models — one per hierarchy level.
categoriesTable table.Model
seriesTable     table.Model
eventsTable     table.Model
marketsTable    table.Model

// Orderbook viewport (scrollable text).
orderbookVP viewport.Model
obContent   string // pre-rendered orderbook text

// Client-side row filter (/ key).
filterInput textinput.Model
filterMode  bool   // true = textinput is open and focused
filterQuery string // active query (persists after closing the input bar)

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

// New creates the initial model.
// Pass authenticated=false when no API credentials are available; market browsing
// (categories/series/events/markets/orderbook) still works via public endpoints,
// but balance display and order entry are disabled.
func New(client *kalshi.ClientWithResponses, env string, authenticated bool) Model {
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
client:        client,
env:           env,
authenticated: authenticated,
nav:           []navEntry{{label: "categories", screen: ScreenCategories}},
screen:        ScreenCategories,
spinner:       sp,
filterInput:   fi,
loading:       true,
}
}

// Init fires the initial data-loading commands.
// We load all series upfront; categories are derived from that payload.
func (m Model) Init() tea.Cmd {
cmds := tea.Batch(loadSeries(m.client), m.spinner.Tick)
if m.authenticated {
return tea.Batch(cmds, loadBalance(m.client))
}
return cmds
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

rows := make([]categoryRow, 0, len(cats)+1)
// "All" meta-row.
allTags := make(map[string]struct{})
for _, tagSet := range catMap {
for t := range tagSet {
allTags[t] = struct{}{}
}
}
rows = append(rows, categoryRow{
name:        "",
seriesCount: len(m.seriesData),
tags:        sortedKeys(allTags),
})
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

// applyFilter rebuilds the current screen's table rows filtered by filterQuery.
func (m *Model) applyFilter() {
q := strings.ToLower(m.filterQuery)
switch m.screen {
case ScreenCategories:
filtered := make([]categoryRow, 0, len(m.categoryRows))
for _, r := range m.categoryRows {
label := r.name
if label == "" {
label = "all"
}
if q == "" ||
strings.Contains(strings.ToLower(label), q) ||
containsTag(r.tags, q) {
filtered = append(filtered, r)
}
}
m.initCategoriesTable(filtered)

case ScreenSeriesList:
filtered := make([]kalshi.Series, 0, len(m.seriesData))
for _, s := range m.seriesData {
// Category gate: only show series matching the active category filter.
if m.categoryFilter != "" && s.Category != m.categoryFilter {
continue
}
if q == "" ||
strings.Contains(strings.ToLower(s.Ticker), q) ||
strings.Contains(strings.ToLower(s.Title), q) ||
strings.Contains(strings.ToLower(s.Category), q) ||
containsTag(s.Tags, q) {
filtered = append(filtered, s)
}
}
m.initSeriesTable(filtered)

case ScreenEventsList:
filtered := make([]kalshi.EventData, 0, len(m.eventsData))
for _, e := range m.eventsData {
if q == "" ||
strings.Contains(strings.ToLower(e.EventTicker), q) ||
strings.Contains(strings.ToLower(e.SubTitle), q) {
filtered = append(filtered, e)
}
}
m.initEventsTable(filtered)

case ScreenMarketsList:
filtered := make([]kalshi.Market, 0, len(m.marketsData))
for _, mkt := range m.marketsData {
if q == "" ||
strings.Contains(strings.ToLower(mkt.Ticker), q) ||
strings.Contains(strings.ToLower(string(mkt.Status)), q) {
filtered = append(filtered, mkt)
}
}
m.initMarketsTable(filtered)
}
}

// containsTag returns true if any tag in tags contains the query string.
func containsTag(tags []string, q string) bool {
for _, t := range tags {
if strings.Contains(strings.ToLower(t), q) {
return true
}
}
return false
}

// --- table initializers ---

func (m *Model) initCategoriesTable(rows []categoryRow) {
w := m.contentWidth()
tagsW := w - 14 - 8 - 4
if tagsW < 20 {
tagsW = 20
}
cols := []table.Column{
{Title: "CATEGORY", Width: 14},
{Title: "SERIES", Width: 8},
{Title: "TAGS", Width: tagsW},
}
trows := make([]table.Row, 0, len(rows))
for _, r := range rows {
name := r.name
if name == "" {
name = "(all)"
}
trows = append(trows, table.Row{
name,
strings.NewReplacer().Replace(formatInt(r.seriesCount)),
truncate(strings.Join(r.tags, ", "), tagsW),
})
}
t := table.New(
table.WithColumns(cols),
table.WithRows(trows),
table.WithFocused(true),
table.WithHeight(m.contentHeight()-2),
)
t.SetStyles(tableStyles())
m.categoriesTable = t
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
tickerW := 14
tagsW := 20
titleW := w - tickerW - tagsW - 12 - 12 - 6
if titleW < 16 {
titleW = 16
}
cols := []table.Column{
{Title: "TICKER", Width: tickerW},
{Title: "TITLE", Width: titleW},
{Title: "CATEGORY", Width: 12},
{Title: "FREQ", Width: 8},
{Title: "TAGS", Width: tagsW},
}
rows := make([]table.Row, 0, len(series))
for _, s := range series {
rows = append(rows, table.Row{
s.Ticker,
truncate(s.Title, titleW),
s.Category,
s.Frequency,
truncate(strings.Join(s.Tags, ", "), tagsW),
})
}
t := table.New(
table.WithColumns(cols),
table.WithRows(rows),
table.WithFocused(true),
table.WithHeight(m.contentHeight()-2),
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
table.WithHeight(m.contentHeight()-2),
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
table.WithHeight(m.contentHeight()-2),
)
t.SetStyles(tableStyles())
m.marketsTable = t
}
