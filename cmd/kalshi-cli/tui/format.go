// format.go provides display-formatting helpers for the TUI package.
// These mirror the helpers in the parent package's output.go, scoped here
// to avoid cross-package coupling between main and tui.
package tui

import (
	"fmt"
	"strconv"
)

func fmtCents(d string) string {
	f, err := strconv.ParseFloat(d, 64)
	if err != nil || d == "" {
		return "-"
	}
	return fmt.Sprintf("%.0f¢", f*100)
}

func fmtDollarsFromCents(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func parseFP(d string) float64 {
	f, _ := strconv.ParseFloat(d, 64)
	return f
}

func fmtSpread(bid, ask string) string {
	diff := parseFP(ask) - parseFP(bid)
	if diff < 0 {
		diff = 0
	}
	return fmt.Sprintf("%.0f¢", diff*100)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func derefStr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
