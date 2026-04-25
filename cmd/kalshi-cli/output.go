package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"
)

var flagOutput string

func isWide() bool { return flagOutput == "wide" }

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printYAML(v any) error {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	return enc.Encode(v)
}

func printTable(headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
}

// render dispatches to json, yaml, or table depending on flagOutput.
func render(data any, tableFunc func(wide bool) ([]string, [][]string)) error {
	switch flagOutput {
	case "json":
		return printJSON(data)
	case "yaml":
		return printYAML(data)
	default: // "table", "wide", or empty
		headers, rows := tableFunc(isWide())
		return printTable(headers, rows)
	}
}

// --- formatting helpers ---

func fmtCents(d string) string {
	f, err := strconv.ParseFloat(d, 64)
	if err != nil || d == "" {
		return "-"
	}
	return fmt.Sprintf("%.0f¢", f*100)
}

func fmtDollars(cents int64) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}

func parseFP(d string) float64 {
	f, _ := strconv.ParseFloat(d, 64)
	return f
}

func fmtSpread(bid, ask string) string {
	b := parseFP(bid)
	a := parseFP(ask)
	diff := a - b
	if diff < 0 {
		diff = 0
	}
	return fmt.Sprintf("%.0f¢", diff*100)
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.UTC().Format("01/02 15:04Z")
}

func fmtTimeVal(t time.Time) string {
	return t.UTC().Format("01/02 15:04Z")
}

func fmtBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8] + "…"
	}
	return id
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
