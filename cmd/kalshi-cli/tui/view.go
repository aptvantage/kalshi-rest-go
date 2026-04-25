package tui

import (
"fmt"
"strings"

tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
)

// View renders the entire terminal screen as a string.
// Called after every Update; must be a pure function of model state.
func (m Model) View() string {
if m.width == 0 {
return "loading…"
}
return strings.Join([]string{
m.viewHeader(),
m.viewContent(),
m.viewStatusBar(),
m.viewHelpBar(),
}, "\n")
}

// ---- header ----

func (m Model) viewHeader() string {
left := m.viewBreadcrumb()
right := m.viewHeaderRight()
gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
if gap < 0 {
gap = 0
}
return headerStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) viewBreadcrumb() string {
parts := make([]string, len(m.nav))
for i, n := range m.nav {
if i == len(m.nav)-1 {
parts[i] = breadcrumbActiveStyle.Render(n.label)
} else {
parts[i] = breadcrumbStyle.Render(n.label)
}
}
sep := breadcrumbSepStyle.Render(" › ")
return strings.Join(parts, sep)
}

func (m Model) viewHeaderRight() string {
var envBadge string
if m.env == "demo" {
envBadge = envDemoStyle.Render("DEMO")
} else {
envBadge = envProdStyle.Render("PROD")
}
var bal string
if m.authenticated {
if m.balance != nil {
bal = balanceStyle.Render(fmtDollarsFromCents(*m.balance))
} else {
bal = loadStyle.Render("loading…")
}
} else {
bal = errStyle.Render("no auth")
}
return bal + "  " + envBadge
}

// ---- content ----

func (m Model) viewContent() string {
h := m.contentHeight()

var parts []string

// Filter bar (shown when filter mode is active OR a query is set).
if m.filterMode || m.filterQuery != "" {
parts = append(parts, m.viewFilterBar())
h-- // consumed one line
}

switch m.screen {
case ScreenSeriesList, ScreenEventsList, ScreenMarketsList:
parts = append(parts, m.viewTable(h))
case ScreenOrderbook:
parts = append(parts, orderbookStyle.Render(m.orderbookVP.View()))
case ScreenOrderEntry:
parts = append(parts, m.viewOrderEntry())
default:
parts = append(parts, "")
}

return strings.Join(parts, "\n")
}

func (m Model) viewFilterBar() string {
var content string
if m.filterMode {
content = m.filterInput.View()
} else {
content = filterLabelStyle.Render("filter: ") +
lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6")).Render(m.filterQuery) +
"  " +
lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("(/ edit · esc clear)")
}
return filterBarStyle.Width(m.width).Render(content)
}

func (m Model) viewTable(height int) string {
if m.loading {
return contentPad.Render(m.spinner.View() + "  loading…")
}
if m.err != nil {
return contentPad.Render(errStyle.Render("Error: " + m.err.Error()))
}

_ = height // table handles its own height
switch m.screen {
case ScreenSeriesList:
return m.seriesTable.View()
case ScreenEventsList:
return m.eventsTable.View()
case ScreenMarketsList:
return m.marketsTable.View()
}
return ""
}

// ---- order entry form ----

func (m Model) viewOrderEntry() string {
f := m.orderForm
var sb strings.Builder

cursor := func(i int) string {
if f.focus == i {
return focusCursorStyle.Render("▶ ")
}
return "  "
}
active := func(i int, label string) string {
if f.focus == i {
return formActiveStyle.Render(label)
}
return label
}
toggle := func(i int, current string) string {
label := "[" + current + "]"
if f.focus == i {
return formActiveStyle.Render(label)
}
return label
}
checkbox := func(i int, checked bool) string {
mark := "[ ]"
if checked {
mark = "[x]"
}
if f.focus == i {
return formActiveStyle.Render(mark)
}
return mark
}

title := formTitleStyle.Render("New Order")
ticker := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(f.ticker)
sb.WriteString(fmt.Sprintf("  %s  %s\n\n", title, ticker))

// Side (0)
sb.WriteString(fmt.Sprintf("%sside    %s\n", cursor(0), toggle(0, f.side)))
// Action (1)
sb.WriteString(fmt.Sprintf("%saction  %s\n", cursor(1), toggle(1, f.action)))
// Count (2)
sb.WriteString(fmt.Sprintf("%scount   %s\n", cursor(2), active(2, f.countInput.View())))
// Price (3)
sb.WriteString(fmt.Sprintf("%sprice   %s ¢\n", cursor(3), active(3, f.priceInput.View())))
// Post-Only (4)
sb.WriteString(fmt.Sprintf("%spost-only %s\n\n", cursor(4), checkbox(4, f.postOnly)))

// Submit / Cancel (5, 6)
var submitBtn, cancelBtn string
if f.submitting {
submitBtn = loadStyle.Render("submitting…")
} else if f.focus == 5 {
submitBtn = focusedBtnStyle.Render("[ Submit ]")
} else {
submitBtn = "[ Submit ]"
}
if f.focus == 6 {
cancelBtn = focusedBtnStyle.Render("[ Cancel ]")
} else {
cancelBtn = "[ Cancel ]"
}
sb.WriteString(fmt.Sprintf("  %s  %s\n", submitBtn, cancelBtn))

// Result / error feedback.
if f.result != "" {
sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(f.result) + "\n")
}
if f.err != nil {
sb.WriteString("\n  " + errStyle.Render(f.err.Error()) + "\n")
}

return contentPad.Render(sb.String())
}

// ---- status bar ----

func (m Model) viewStatusBar() string {
var parts []string

switch m.screen {
case ScreenSeriesList:
n := len(m.seriesTable.Rows())
parts = append(parts, fmt.Sprintf("%d series", n))
case ScreenEventsList:
n := len(m.eventsTable.Rows())
parts = append(parts, fmt.Sprintf("%d events", n))
case ScreenMarketsList:
n := len(m.marketsTable.Rows())
parts = append(parts, fmt.Sprintf("%d markets", n))
case ScreenOrderbook:
parts = append(parts, m.selectedMarketTicker)
case ScreenOrderEntry:
parts = append(parts, "order entry")
}

if m.filterQuery != "" && !m.filterMode {
parts = append(parts, "filter: "+m.filterQuery)
}

text := strings.Join(parts, "  ·  ")
return statusBarStyle.Width(m.width).Render(text)
}

// ---- help bar ----

func (m Model) viewHelpBar() string {
var hints []string

if m.filterMode {
hints = []string{"enter  apply", "esc  clear filter", "type to filter"}
} else {
switch m.screen {
case ScreenSeriesList, ScreenEventsList, ScreenMarketsList:
hints = []string{"↑↓  navigate", "⏎  open", "esc  back", "/  filter"}
if m.screen == ScreenMarketsList && m.authenticated {
hints = append(hints, "o  new order")
}
case ScreenOrderbook:
hints = []string{"↑↓  scroll", "esc  back", "q  quit"}
case ScreenOrderEntry:
hints = []string{"tab  next field", "space  toggle", "ctrl+s  submit", "esc  back"}
}
}

var parts []string
for _, h := range hints {
parts = append(parts, helpStyle.Render(h))
}
return helpStyle.Width(m.width).Render(strings.Join(parts, "   "))
}

// Ensure Model satisfies tea.Model at compile time.
var _ tea.Model = Model{}
