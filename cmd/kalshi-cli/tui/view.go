package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full terminal UI for the current model state.
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing…"
	}

	header := m.viewHeader()
	content := m.viewContent()
	statusBar := m.viewStatusBar()
	helpBar := m.viewHelpBar()

	return strings.Join([]string{header, content, statusBar, helpBar}, "\n")
}

// viewHeader renders the top bar: breadcrumb on left, env badge + balance on right.
func (m Model) viewHeader() string {
	crumb := m.viewBreadcrumb()
	right := m.viewHeaderRight()

	// Fill space between crumb and right-aligned content.
	crumbW := visibleWidth(crumb)
	rightW := visibleWidth(right)
	gap := m.width - crumbW - rightW - 2 // -2 for the padding added by headerStyle
	if gap < 1 {
		gap = 1
	}
	spacer := strings.Repeat(" ", gap)

	inner := crumb + spacer + right
	return headerStyle.Width(m.width).Render(inner)
}

// viewBreadcrumb renders "series > KXHIGHNY > events" navigation trail.
func (m Model) viewBreadcrumb() string {
	sep := breadcrumbSepStyle.Render(" › ")
	parts := make([]string, len(m.nav))
	for i, entry := range m.nav {
		if i == len(m.nav)-1 {
			parts[i] = breadcrumbActiveStyle.Render(entry.label)
		} else {
			parts[i] = breadcrumbStyle.Render(entry.label)
		}
	}
	return strings.Join(parts, sep)
}

// viewHeaderRight renders "[demo] $500.00" or "[prod] –".
func (m Model) viewHeaderRight() string {
	var envBadge string
	if m.env == "demo" {
		envBadge = envDemoStyle.Render("DEMO")
	} else {
		envBadge = envProdStyle.Render("PROD")
	}

	bal := "  –"
	if !m.authenticated {
		bal = "  " + errStyle.Render("no auth")
	} else if m.balance != nil {
		bal = "  " + balanceStyle.Render(fmtDollarsFromCents(*m.balance))
	}
	return envBadge + bal
}

// viewContent renders the main body based on current screen.
func (m Model) viewContent() string {
	if m.loading && !hasTableData(m) {
		lines := m.contentHeight()
		if lines < 1 {
			lines = 1
		}
		placeholder := m.spinner.View() + loadStyle.Render(" Loading…")
		return contentPad.Render(placeholder) + strings.Repeat("\n", lines-1)
	}

	switch m.screen {
	case ScreenSeriesList:
		return contentPad.Render(m.seriesTable.View())
	case ScreenEventsList:
		return contentPad.Render(m.eventsTable.View())
	case ScreenMarketsList:
		return contentPad.Render(m.marketsTable.View())
	case ScreenOrderbook:
		return orderbookStyle.Render(m.orderbookVP.View())
	}
	return ""
}

// viewStatusBar renders the bottom status line (errors, loading indicator, row count).
func (m Model) viewStatusBar() string {
	var msg string
	switch {
	case m.err != nil:
		msg = errStyle.Render("✗ " + m.err.Error())
	case m.loading:
		msg = m.spinner.View() + loadStyle.Render(" refreshing…")
	default:
		msg = m.viewRowCount()
	}
	return statusBarStyle.Width(m.width).Render(msg)
}

func (m Model) viewRowCount() string {
	switch m.screen {
	case ScreenSeriesList:
		return fmt.Sprintf("%d series", len(m.seriesData))
	case ScreenEventsList:
		return fmt.Sprintf("%d events  (series: %s)", len(m.eventsData), m.selectedSeriesTicker)
	case ScreenMarketsList:
		return fmt.Sprintf("%d markets  (event: %s)", len(m.marketsData), m.selectedEventTicker)
	case ScreenOrderbook:
		return fmt.Sprintf("orderbook: %s  (↑/↓ scroll)", m.selectedMarketTicker)
	}
	return ""
}

// viewHelpBar renders one-line keybinding hints for the current screen.
func (m Model) viewHelpBar() string {
	var hints []string

	switch m.screen {
	case ScreenSeriesList:
		hints = []string{"↑/↓ navigate", "enter select", "/ filter", "r refresh", "q quit"}
	case ScreenEventsList:
		hints = []string{"↑/↓ navigate", "enter select", "esc back", "r refresh", "q quit"}
	case ScreenMarketsList:
		hints = []string{"↑/↓ navigate", "enter orderbook", "esc back", "r refresh", "q quit"}
	case ScreenOrderbook:
		hints = []string{"↑/↓ scroll", "esc back", "r refresh", "q quit"}
	}

	return helpStyle.Render("  " + strings.Join(hints, "  ·  "))
}

// hasTableData reports whether there is already data to display (avoids
// blanking a populated table during a background refresh).
func hasTableData(m Model) bool {
	switch m.screen {
	case ScreenSeriesList:
		return len(m.seriesData) > 0
	case ScreenEventsList:
		return len(m.eventsData) > 0
	case ScreenMarketsList:
		return len(m.marketsData) > 0
	case ScreenOrderbook:
		return m.obContent != ""
	}
	return false
}

// visibleWidth returns the display width of s, accounting for ANSI escape sequences.
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}
