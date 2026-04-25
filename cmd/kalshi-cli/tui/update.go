package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.resizeTables()
		return m, nil

	case tea.KeyMsg:
		// Global keys — handled before delegating to focused component.
		switch {
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		case msg.String() == "q" && m.screen == ScreenSeriesList:
			return m, tea.Quit
		case msg.String() == "esc":
			m = m.navigateBack()
			return m, nil
		case msg.String() == "r":
			m.loading = true
			return m, tea.Batch(m.refreshScreen(), m.spinner.Tick)
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case SeriesLoadedMsg:
		m.loading = false
		m.seriesData = msg.Series
		m.initSeriesTable(msg.Series)
		return m, tick()

	case EventsLoadedMsg:
		m.loading = false
		m.eventsData = msg.Events
		m.initEventsTable(msg.Events)
		m.screen = ScreenEventsList
		m.nav = appendOrReplace(m.nav, navEntry{
			label:  m.selectedSeriesTicker,
			screen: ScreenEventsList,
		})
		return m, nil

	case MarketsLoadedMsg:
		m.loading = false
		m.marketsData = msg.Markets
		m.initMarketsTable(msg.Markets)
		m.screen = ScreenMarketsList
		m.nav = appendOrReplace(m.nav, navEntry{
			label:  m.selectedEventTicker,
			screen: ScreenMarketsList,
		})
		return m, nil

	case OrderbookLoadedMsg:
		m.loading = false
		m.obContent = renderOrderbook(msg.Ticker, msg.Orderbook)
		m.orderbookVP = viewport.New(m.contentWidth(), m.contentHeight())
		m.orderbookVP.SetContent(m.obContent)
		m.screen = ScreenOrderbook
		m.nav = appendOrReplace(m.nav, navEntry{
			label:  msg.Ticker,
			screen: ScreenOrderbook,
		})
		return m, nil

	case BalanceLoadedMsg:
		m.balance = &msg.Balance
		return m, nil

	case TickMsg:
		// Only auto-refresh the orderbook on tick — it's the most time-sensitive.
		if m.screen == ScreenOrderbook && m.selectedMarketTicker != "" {
			m.loading = true
			return m, tea.Batch(
				loadOrderbook(m.client, m.selectedMarketTicker),
				m.spinner.Tick,
				tick(),
			)
		}
		return m, tick()

	case ErrMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil
	}

	// Delegate remaining messages to the focused component.
	return m.updateFocused(msg)
}

// updateFocused delegates input to whichever component is active.
func (m Model) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.screen {
	case ScreenSeriesList:
		m.seriesTable, cmd = m.seriesTable.Update(msg)
		if isEnter(msg) {
			if row := m.seriesTable.SelectedRow(); len(row) > 0 {
				m.selectedSeriesTicker = row[0]
				m.loading = true
				return m, tea.Batch(cmd, loadEvents(m.client, m.selectedSeriesTicker), m.spinner.Tick)
			}
		}

	case ScreenEventsList:
		m.eventsTable, cmd = m.eventsTable.Update(msg)
		if isEnter(msg) {
			if row := m.eventsTable.SelectedRow(); len(row) > 0 {
				m.selectedEventTicker = row[0]
				m.loading = true
				return m, tea.Batch(cmd, loadMarkets(m.client, m.selectedEventTicker), m.spinner.Tick)
			}
		}

	case ScreenMarketsList:
		m.marketsTable, cmd = m.marketsTable.Update(msg)
		if isEnter(msg) {
			if row := m.marketsTable.SelectedRow(); len(row) > 0 {
				m.selectedMarketTicker = row[0]
				m.loading = true
				return m, tea.Batch(cmd, loadOrderbook(m.client, m.selectedMarketTicker), m.spinner.Tick)
			}
		}

	case ScreenOrderbook:
		m.orderbookVP, cmd = m.orderbookVP.Update(msg)
	}

	return m, cmd
}

// navigateBack pops one level from the breadcrumb and restores that screen.
func (m Model) navigateBack() Model {
	if len(m.nav) <= 1 {
		return m
	}
	m.nav = m.nav[:len(m.nav)-1]
	m.screen = m.nav[len(m.nav)-1].screen
	m.err = nil
	return m
}

// refreshScreen reloads data for the currently active screen.
func (m Model) refreshScreen() tea.Cmd {
	switch m.screen {
	case ScreenSeriesList:
		return loadSeries(m.client)
	case ScreenEventsList:
		return loadEvents(m.client, m.selectedSeriesTicker)
	case ScreenMarketsList:
		return loadMarkets(m.client, m.selectedEventTicker)
	case ScreenOrderbook:
		return loadOrderbook(m.client, m.selectedMarketTicker)
	}
	return nil
}

// resizeTables updates all component dimensions after a terminal resize.
func (m Model) resizeTables() Model {
	h := m.contentHeight() - 2 // table height excludes header row
	if h < 1 {
		h = 1
	}
	m.seriesTable.SetHeight(h)
	m.eventsTable.SetHeight(h)
	m.marketsTable.SetHeight(h)
	m.orderbookVP.Width = m.contentWidth()
	m.orderbookVP.Height = m.contentHeight()
	return m
}

// renderOrderbook produces a two-column YES/NO display for the viewport.
func renderOrderbook(ticker string, ob kalshi.OrderbookCountFp) string {
	const depth = 8
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  Orderbook: %s\n\n", ticker))
	sb.WriteString(fmt.Sprintf("  %-16s  %s\n", "YES (bids)", "NO (asks)"))
	sb.WriteString(fmt.Sprintf("  %-16s  %s\n", "──────────────", "──────────────"))

	// Sort YES descending (best bid first), NO ascending (best ask first).
	yes := make([]kalshi.PriceLevelDollarsCountFp, len(ob.YesDollars))
	copy(yes, ob.YesDollars)
	sort.Slice(yes, func(i, j int) bool { return parseFP(yes[i][0]) > parseFP(yes[j][0]) })

	no := make([]kalshi.PriceLevelDollarsCountFp, len(ob.NoDollars))
	copy(no, ob.NoDollars)
	sort.Slice(no, func(i, j int) bool { return parseFP(no[i][0]) < parseFP(no[j][0]) })

	rows := depth
	if len(yes) > rows {
		rows = depth
	}
	for i := 0; i < rows; i++ {
		yesCell := "     -      "
		noCell := "     -      "
		if i < len(yes) && len(yes[i]) >= 2 {
			yesCell = fmt.Sprintf("%4s × %-6s", fmtCents(yes[i][0]), yes[i][1])
		}
		if i < len(no) && len(no[i]) >= 2 {
			noCell = fmt.Sprintf("%4s × %-6s", fmtCents(no[i][0]), no[i][1])
		}
		sb.WriteString(fmt.Sprintf("  %-16s  %s\n", yesCell, noCell))
	}

	return sb.String()
}

// --- small helpers ---

func isEnter(msg tea.Msg) bool {
	km, ok := msg.(tea.KeyMsg)
	return ok && km.String() == "enter"
}

// appendOrReplace adds nav to the stack, or replaces the top entry if it's
// the same screen (prevents double-stacking on refresh).
func appendOrReplace(nav []navEntry, entry navEntry) []navEntry {
	if len(nav) > 0 && nav[len(nav)-1].screen == entry.screen {
		nav[len(nav)-1] = entry
		return nav
	}
	return append(nav, entry)
}
