// Package tui implements the interactive terminal UI for kalshi-cli.
//
// Navigation hierarchy mirrors the Kalshi data model:
//
//Series  →  Events  →  Markets  →  Orderbook
//
// Launch with tea.NewProgram(tui.New(client, env, authenticated), tea.WithAltScreen()).
package tui

import (
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
ScreenSeriesList Screen = iota
ScreenEventsList
ScreenMarketsList
ScreenOrderbook
ScreenOrderEntry
)

// navEntry is one breadcrumb step.
type navEntry struct {
label  string
screen Screen
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

// Table models — one per hierarchy level.
seriesTable  table.Model
eventsTable  table.Model
marketsTable table.Model

// Orderbook viewport (scrollable text).
orderbookVP viewport.Model
obContent   string // pre-rendered orderbook text

// Client-side row filter.
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
// (series/events/markets/orderbook) still works via public endpoints, but balance
// display and order entry are disabled.
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
nav:           []navEntry{{label: "series", screen: ScreenSeriesList}},
screen:        ScreenSeriesList,
spinner:       sp,
filterInput:   fi,
loading:       true,
}
}

// Init fires the initial data-loading commands.
func (m Model) Init() tea.Cmd {
cmds := tea.Batch(loadSeries(m.client), m.spinner.Tick)
if m.authenticated {
return tea.Batch(cmds, loadBalance(m.client))
}
return cmds
}

// --- helpers ---

func (m Model) contentHeight() int {
if m.height < 4 {
return 1
}
return m.height - 3 // header (1) + status bar (1) + help bar (1)
}

func (m Model) contentWidth() int {
if m.width < 4 {
return 40
}
return m.width
}

// applyFilter rebuilds the current screen's table rows from raw data, keeping only
// rows whose text columns contain filterQuery (case-insensitive).
func (m *Model) applyFilter() {
q := strings.ToLower(m.filterQuery)
switch m.screen {
case ScreenSeriesList:
filtered := make([]kalshi.Series, 0, len(m.seriesData))
for _, s := range m.seriesData {
if q == "" ||
strings.Contains(strings.ToLower(s.Ticker), q) ||
strings.Contains(strings.ToLower(s.Title), q) ||
strings.Contains(strings.ToLower(s.Category), q) {
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

// initSeriesTable builds the series table using current terminal dimensions.
func (m *Model) initSeriesTable(series []kalshi.Series) {
w := m.contentWidth()
tickerW := 14
titleW := w - tickerW - 12 - 12 - 4
if titleW < 20 {
titleW = 20
}
cols := []table.Column{
{Title: "TICKER", Width: tickerW},
{Title: "TITLE", Width: titleW},
{Title: "CATEGORY", Width: 12},
{Title: "FREQUENCY", Width: 10},
}
rows := make([]table.Row, 0, len(series))
for _, s := range series {
rows = append(rows, table.Row{
s.Ticker,
truncate(s.Title, titleW),
s.Category,
s.Frequency,
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

// initEventsTable builds the events table using current terminal dimensions.
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

// initMarketsTable builds the markets table using current terminal dimensions.
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
