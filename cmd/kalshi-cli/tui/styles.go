package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	clrPurple = lipgloss.Color("#7C3AED")
	clrGreen  = lipgloss.Color("#10B981")
	clrRed    = lipgloss.Color("#EF4444")
	clrYellow = lipgloss.Color("#F59E0B")
	clrMuted  = lipgloss.Color("#6B7280")
	clrDim    = lipgloss.Color("#374151")
	clrFg     = lipgloss.Color("#F3F4F6")

	headerStyle = lipgloss.NewStyle().
			Background(clrPurple).
			Foreground(clrFg).
			Bold(true).
			Padding(0, 1)

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(clrMuted)

	breadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(clrFg).
				Bold(true)

	breadcrumbSepStyle = lipgloss.NewStyle().
				Foreground(clrDim)

	envDemoStyle = lipgloss.NewStyle().
			Background(clrYellow).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1)

	envProdStyle = lipgloss.NewStyle().
			Background(clrGreen).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1)

	balanceStyle = lipgloss.NewStyle().
			Foreground(clrGreen).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Background(clrDim).
			Foreground(clrMuted).
			Padding(0, 1)

	errStyle    = lipgloss.NewStyle().Foreground(clrRed)
	loadStyle   = lipgloss.NewStyle().Foreground(clrMuted).Italic(true)
	helpStyle   = lipgloss.NewStyle().Foreground(clrMuted)
	contentPad  = lipgloss.NewStyle().Padding(0, 1)
	orderbookStyle = lipgloss.NewStyle().Padding(1, 2)
)

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(clrDim).
		BorderBottom(true).
		Bold(true).
		Foreground(clrMuted)
	s.Selected = s.Selected.
		Foreground(clrFg).
		Background(lipgloss.Color("#1D4ED8")).
		Bold(false)
	return s
}
