package tui

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
)

// Update processes every message dispatched by the Bubble Tea runtime.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
var cmds []tea.Cmd

switch msg := msg.(type) {
case tea.WindowSizeMsg:
m.width = msg.Width
m.height = msg.Height
if len(m.seriesData) > 0 {
m.applyFilter()
}
if m.screen == ScreenOrderbook {
m.orderbookVP = viewport.New(m.contentWidth()-4, m.contentHeight()-2)
m.orderbookVP.SetContent(m.obContent)
}
return m, nil

case spinner.TickMsg:
var cmd tea.Cmd
m.spinner, cmd = m.spinner.Update(msg)
cmds = append(cmds, cmd)

case TickMsg:
switch m.screen {
case ScreenCategories:
cmds = append(cmds, loadSeries(m.client))
case ScreenSeriesList:
cmds = append(cmds, loadSeries(m.client))
case ScreenEventsList:
cmds = append(cmds, loadEvents(m.client, m.selectedSeriesTicker))
case ScreenMarketsList:
cmds = append(cmds, loadMarkets(m.client, m.selectedEventTicker))
case ScreenOrderbook:
cmds = append(cmds, loadOrderbook(m.client, m.selectedMarketTicker))
}
cmds = append(cmds, tick())

	case SeriesLoadedMsg:
		m.loading = false
		m.err = nil
		m.seriesData = msg.Series
		log.Printf("[tui] series loaded: %d series", len(msg.Series))
		m.buildCategoryRows()
		m.applyFilter()

	case EventsLoadedMsg:
		m.loading = false
		m.err = nil
		m.eventsData = msg.Events
		log.Printf("[tui] events loaded: %d events for series %s", len(msg.Events), m.selectedSeriesTicker)
		m.applyFilter()

	case MarketsLoadedMsg:
		m.loading = false
		m.err = nil
		m.marketsData = msg.Markets
		log.Printf("[tui] markets loaded: %d markets for event %s", len(msg.Markets), m.selectedEventTicker)
		m.applyFilter()

	case BalanceLoadedMsg:
		bal := msg.Balance
		m.balance = &bal
		log.Printf("[tui] balance loaded: %d cents", bal)

	case OrderbookLoadedMsg:
		m.loading = false
		m.err = nil
		log.Printf("[tui] orderbook loaded: %s yes=%d no=%d levels",
			msg.Ticker, len(msg.Orderbook.YesDollars), len(msg.Orderbook.NoDollars))
		content := renderOrderbook(msg.Orderbook, msg.Ticker)
		m.obContent = content
		vp := viewport.New(m.contentWidth()-4, m.contentHeight()-2)
		vp.SetContent(content)
		m.orderbookVP = vp

	case OrderCreatedMsg:
		m.orderForm.submitting = false
		m.orderForm.result = fmt.Sprintf("✓ Order placed: %s", shortID(msg.Order.OrderId))
		m.orderForm.err = nil
		log.Printf("[tui] order created: %s", msg.Order.OrderId)
		if m.authenticated {
			cmds = append(cmds, loadBalance(m.client))
		}

	case ErrMsg:
		m.loading = false
		log.Printf("[tui] error (screen=%d): %v", m.screen, msg.Err)
		if m.screen == ScreenOrderEntry {
			m.orderForm.submitting = false
			m.orderForm.err = msg.Err
		} else {
			m.err = msg.Err
		}

case tea.KeyMsg:
if m.filterMode {
return m.updateFilterInput(msg)
}
return m.updateFocused(msg)
}

return m, tea.Batch(cmds...)
}

// updateFilterInput handles key events when the filter bar is open.
func (m Model) updateFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
switch msg.String() {
case "esc":
m.filterMode = false
m.filterQuery = ""
m.filterInput.SetValue("")
m.applyFilter()
return m, nil
case "enter":
m.filterMode = false
m.filterQuery = m.filterInput.Value()
m.applyFilter()
return m, nil
default:
var cmd tea.Cmd
m.filterInput, cmd = m.filterInput.Update(msg)
m.filterQuery = m.filterInput.Value()
m.applyFilter()
return m, cmd
}
}

// updateFocused handles navigation key events for the focused screen.
func (m Model) updateFocused(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
var cmds []tea.Cmd

if m.screen != ScreenOrderEntry {
switch {
case key.Matches(msg, DefaultKeyMap.Quit):
return m, tea.Quit

case key.Matches(msg, DefaultKeyMap.Filter):
switch m.screen {
case ScreenCategories, ScreenSeriesList, ScreenEventsList, ScreenMarketsList:
m.filterMode = true
m.filterInput.SetValue(m.filterQuery)
return m, m.filterInput.Focus()
}

case key.Matches(msg, DefaultKeyMap.Order):
if m.screen == ScreenMarketsList && m.authenticated {
row := m.marketsTable.SelectedRow()
if len(row) > 0 {
m.orderForm = newOrderForm(row[0])
m.nav = append(m.nav, navEntry{label: "order:" + row[0], screen: ScreenOrderEntry})
m.screen = ScreenOrderEntry
return m, m.syncOrderFormFocus()
}
}
}
}

switch m.screen {
case ScreenCategories:
switch {
case key.Matches(msg, DefaultKeyMap.Up):
m.categoriesTable.MoveUp(1)
case key.Matches(msg, DefaultKeyMap.Down):
m.categoriesTable.MoveDown(1)
case key.Matches(msg, DefaultKeyMap.Enter):
row := m.categoriesTable.SelectedRow()
if len(row) > 0 {
// row[0] is "(all)" or a real category name.
catName := row[0]
if catName == "(all)" {
m.categoryFilter = ""
m.nav = append(m.nav, navEntry{label: "all series", screen: ScreenSeriesList})
} else {
m.categoryFilter = catName
m.nav = append(m.nav, navEntry{label: catName, screen: ScreenSeriesList})
}
m.screen = ScreenSeriesList
m.filterQuery = ""
m.filterInput.SetValue("")
m.applyFilter()
}
}

case ScreenSeriesList:
switch {
case key.Matches(msg, DefaultKeyMap.Up):
m.seriesTable.MoveUp(1)
case key.Matches(msg, DefaultKeyMap.Down):
m.seriesTable.MoveDown(1)
case key.Matches(msg, DefaultKeyMap.Enter):
row := m.seriesTable.SelectedRow()
if len(row) > 0 {
m.selectedSeriesTicker = row[0]
m.nav = append(m.nav, navEntry{label: row[0], screen: ScreenEventsList})
m.screen = ScreenEventsList
m.loading = true
m.eventsData = nil
m.filterQuery = ""
m.filterInput.SetValue("")
cmds = append(cmds, loadEvents(m.client, m.selectedSeriesTicker))
}
case key.Matches(msg, DefaultKeyMap.Back):
m.navigateBack()
}

case ScreenEventsList:
switch {
case key.Matches(msg, DefaultKeyMap.Up):
m.eventsTable.MoveUp(1)
case key.Matches(msg, DefaultKeyMap.Down):
m.eventsTable.MoveDown(1)
case key.Matches(msg, DefaultKeyMap.Enter):
row := m.eventsTable.SelectedRow()
if len(row) > 0 {
m.selectedEventTicker = row[0]
m.nav = append(m.nav, navEntry{label: row[0], screen: ScreenMarketsList})
m.screen = ScreenMarketsList
m.loading = true
m.marketsData = nil
m.filterQuery = ""
m.filterInput.SetValue("")
cmds = append(cmds, loadMarkets(m.client, m.selectedEventTicker))
}
case key.Matches(msg, DefaultKeyMap.Back):
m.navigateBack()
}

case ScreenMarketsList:
switch {
case key.Matches(msg, DefaultKeyMap.Up):
m.marketsTable.MoveUp(1)
case key.Matches(msg, DefaultKeyMap.Down):
m.marketsTable.MoveDown(1)
case key.Matches(msg, DefaultKeyMap.Enter):
row := m.marketsTable.SelectedRow()
if len(row) > 0 {
m.selectedMarketTicker = row[0]
m.nav = append(m.nav, navEntry{label: row[0], screen: ScreenOrderbook})
m.screen = ScreenOrderbook
m.loading = true
cmds = append(cmds, loadOrderbook(m.client, m.selectedMarketTicker))
}
case key.Matches(msg, DefaultKeyMap.Back):
m.navigateBack()
}

case ScreenOrderbook:
switch {
case key.Matches(msg, DefaultKeyMap.Quit):
return m, tea.Quit
case key.Matches(msg, DefaultKeyMap.Up):
m.orderbookVP.LineUp(1)
case key.Matches(msg, DefaultKeyMap.Down):
m.orderbookVP.LineDown(1)
case key.Matches(msg, DefaultKeyMap.Back):
m.navigateBack()
}

case ScreenOrderEntry:
return m.updateOrderForm(msg)
}

return m, tea.Batch(cmds...)
}

// navigateBack pops the nav stack and resets filter state.
func (m *Model) navigateBack() {
if len(m.nav) <= 1 {
return
}
m.nav = m.nav[:len(m.nav)-1]
m.screen = m.nav[len(m.nav)-1].screen
m.filterMode = false
m.filterQuery = ""
m.filterInput.SetValue("")
// Rebuild the table we're returning to.
m.applyFilter()
}

// updateOrderForm handles key events on the order entry screen.
func (m Model) updateOrderForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
if m.orderForm.submitting {
return m, nil
}

switch msg.String() {
case "ctrl+c":
return m, tea.Quit
case "esc":
m.navigateBack()
return m, nil
case "ctrl+s":
return m.submitOrderForm()
case "tab", "down":
m.orderForm.focus = (m.orderForm.focus + 1) % 7
return m, m.syncOrderFormFocus()
case "shift+tab", "up":
m.orderForm.focus = (m.orderForm.focus + 6) % 7
return m, m.syncOrderFormFocus()
case " ":
switch m.orderForm.focus {
case 0:
if m.orderForm.side == "yes" {
m.orderForm.side = "no"
} else {
m.orderForm.side = "yes"
}
case 1:
if m.orderForm.action == "buy" {
m.orderForm.action = "sell"
} else {
m.orderForm.action = "buy"
}
case 4:
m.orderForm.postOnly = !m.orderForm.postOnly
case 5:
return m.submitOrderForm()
case 6:
m.navigateBack()
return m, nil
}
return m, nil
case "enter":
switch m.orderForm.focus {
case 5:
return m.submitOrderForm()
case 6:
m.navigateBack()
return m, nil
default:
m.orderForm.focus = (m.orderForm.focus + 1) % 7
return m, m.syncOrderFormFocus()
}
}

var cmd tea.Cmd
switch m.orderForm.focus {
case 2:
m.orderForm.countInput, cmd = m.orderForm.countInput.Update(msg)
case 3:
m.orderForm.priceInput, cmd = m.orderForm.priceInput.Update(msg)
}
return m, cmd
}

// syncOrderFormFocus focuses/blurs textinputs based on current focus index.
func (m *Model) syncOrderFormFocus() tea.Cmd {
var cmds []tea.Cmd
if m.orderForm.focus == 2 {
cmds = append(cmds, m.orderForm.countInput.Focus())
} else {
m.orderForm.countInput.Blur()
}
if m.orderForm.focus == 3 {
cmds = append(cmds, m.orderForm.priceInput.Focus())
} else {
m.orderForm.priceInput.Blur()
}
return tea.Batch(cmds...)
}

// submitOrderForm validates and dispatches the createOrder command.
func (m Model) submitOrderForm() (tea.Model, tea.Cmd) {
m.orderForm.result = ""
m.orderForm.err = nil

countStr := strings.TrimSpace(m.orderForm.countInput.Value())
priceStr := strings.TrimSpace(m.orderForm.priceInput.Value())

count, err := strconv.Atoi(countStr)
if err != nil || count <= 0 {
m.orderForm.err = fmt.Errorf("count must be a positive integer")
return m, nil
}
price, err := strconv.Atoi(priceStr)
if err != nil || price < 1 || price > 99 {
m.orderForm.err = fmt.Errorf("price must be 1–99 (cents)")
return m, nil
}

m.orderForm.submitting = true
return m, createOrder(
m.client,
m.orderForm.ticker,
m.orderForm.side,
m.orderForm.action,
count,
price,
m.orderForm.postOnly,
)
}

// renderOrderbook converts a Kalshi OrderbookCountFp to a formatted string.
func renderOrderbook(ob kalshi.OrderbookCountFp, ticker string) string {
boldTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F3F4F6"))
askStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
bidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
headerRowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

var sb strings.Builder
sb.WriteString(boldTitle.Render("Orderbook: "+ticker) + "\n\n")
sb.WriteString(headerRowStyle.Render(fmt.Sprintf("  %-10s %-10s  %-10s %-10s", "YES PRICE", "YES QTY", "NO PRICE", "NO QTY")) + "\n")
sb.WriteString(strings.Repeat("─", 46) + "\n")

maxLen := len(ob.YesDollars)
if len(ob.NoDollars) > maxLen {
maxLen = len(ob.NoDollars)
}

for i := 0; i < maxLen; i++ {
yp, yq := "", ""
np, nq := "", ""
if i < len(ob.YesDollars) && len(ob.YesDollars[i]) >= 2 {
yp = fmtCents(ob.YesDollars[i][0])
yq = ob.YesDollars[i][1]
}
if i < len(ob.NoDollars) && len(ob.NoDollars[i]) >= 2 {
np = fmtCents(ob.NoDollars[i][0])
nq = ob.NoDollars[i][1]
}
row := fmt.Sprintf("  %-10s %-10s  %-10s %-10s", yp, yq, np, nq)
if yp != "" {
sb.WriteString(askStyle.Render(row) + "\n")
} else {
sb.WriteString(bidStyle.Render(row) + "\n")
}
}

if maxLen == 0 {
sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Render("  no levels") + "\n")
}

sb.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("↑/↓ scroll  ·  esc back") + "\n")
return sb.String()
}
