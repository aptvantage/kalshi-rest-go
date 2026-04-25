package tui

import (
"fmt"
"strconv"
"strings"
)

// fmtCents converts a fixed-point dollar string like "0.4500" → "45¢".
func fmtCents(fp string) string {
v := parseFP(fp)
if v <= 0 {
return "–"
}
cents := int(v * 100)
return fmt.Sprintf("%d¢", cents)
}

// fmtDollarsFromCents converts an int64 cents value to "$X.XX".
func fmtDollarsFromCents(cents int64) string {
return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

// fmtSpread returns the ask−bid spread in cents, e.g. "3¢".
func fmtSpread(bidFP, askFP string) string {
bid := parseFP(bidFP)
ask := parseFP(askFP)
if bid <= 0 || ask <= 0 {
return "–"
}
spread := int((ask - bid) * 100)
return fmt.Sprintf("%d¢", spread)
}

// parseFP parses a fixed-point dollar string to float64.
func parseFP(s string) float64 {
v, _ := strconv.ParseFloat(s, 64)
return v
}

// truncate shortens s to at most max runes, appending "…" if truncated.
func truncate(s string, max int) string {
runes := []rune(s)
if len(runes) <= max {
return s
}
if max <= 1 {
return "…"
}
return string(runes[:max-1]) + "…"
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
if s == nil {
return ""
}
return *s
}

// shortID returns the last 8 hex chars of a UUID for compact display.
func shortID(id string) string {
id = strings.ReplaceAll(id, "-", "")
if len(id) <= 8 {
return id
}
return "…" + id[len(id)-8:]
}
