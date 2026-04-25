// Package tui implements the interactive terminal UI for kalshi-cli.
//
// Navigation hierarchy mirrors the Kalshi data model:
//
//	Series  →  Events  →  Markets  →  Orderbook
//
// Launch with tea.NewProgram(tui.New(client, env), tea.WithAltScreen()).
package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
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
)

// navEntry is one breadcrumb step.
type navEntry struct {
	label  string
	screen Screen
}

// Model is the root Bubble Tea model. All mutable state lives here.
type Model struct {
	// Infrastructure
	client *kalshi.ClientWithResponses
	env    string

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
	orderbookVP  viewport.Model
	obContent    string // pre-rendered orderbook text

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

// New creates the initial model. The program calls Init() to fire startup commands.
func New(client *kalshi.ClientWithResponses, env string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = loadStyle

	return Model{
		client:  client,
		env:     env,
		nav:     []navEntry{{label: "series", screen: ScreenSeriesList}},
		screen:  ScreenSeriesList,
		spinner: sp,
		loading: true,
	}
}

// Init fires the initial data-loading commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadSeries(m.client),
		loadBalance(m.client),
		m.spinner.Tick,
	)
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

// initSeriesTable builds the series table using current terminal dimensions.
func (m *Model) initSeriesTable(series []kalshi.Series) {
	w := m.contentWidth()
	tickerW := 14
	titleW := w - tickerW - 12 - 12 - 4 // fill remaining width with title
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
