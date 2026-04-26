package tui

import (
	"testing"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
)

// --- wrapTags tests ---

func TestWrapTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		maxWidth int
		want     []string
	}{
		{
			name:     "empty tags",
			tags:     []string{},
			maxWidth: 40,
			want:     nil,
		},
		{
			name:     "single tag fits",
			tags:     []string{"temperature"},
			maxWidth: 40,
			want:     []string{"temperature"},
		},
		{
			name:     "two tags fit on one line",
			tags:     []string{"temperature", "new york"},
			maxWidth: 40,
			want:     []string{"temperature, new york"},
		},
		{
			name:     "second tag forces wrap",
			tags:     []string{"temperature", "new york"},
			maxWidth: 12, // "temperature" = 11, ", new york" = 10 → won't fit
			want:     []string{"temperature", "new york"},
		},
		{
			name:     "three tags wrap across two lines",
			tags:     []string{"temperature", "new york", "precipitation"},
			maxWidth: 25, // "temperature, new york" = 21 ✓, + ", precipitation" won't fit
			want:     []string{"temperature, new york", "precipitation"},
		},
		{
			name:     "all tags fit on one line",
			tags:     []string{"a", "b", "c"},
			maxWidth: 40,
			want:     []string{"a, b, c"},
		},
		{
			name:     "each tag on its own line (narrow width)",
			tags:     []string{"alpha", "beta", "gamma"},
			maxWidth: 5,
			want:     []string{"alpha", "beta", "gamma"},
		},
		{
			name:     "exact fit — comma+space included",
			tags:     []string{"abc", "de"},
			maxWidth: 7, // "abc, de" = 7 exactly
			want:     []string{"abc, de"},
		},
		{
			name:     "one over exact fit wraps",
			tags:     []string{"abc", "def"},
			maxWidth: 7, // "abc, def" = 8 > 7
			want:     []string{"abc", "def"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := wrapTags(tc.tags, tc.maxWidth)
			if len(got) != len(tc.want) {
				t.Errorf("wrapTags(%v, %d) = %v, want %v", tc.tags, tc.maxWidth, got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("wrapTags(%v, %d)[%d] = %q, want %q", tc.tags, tc.maxWidth, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// --- initCategoriesTable: multi-row wrapping ---

func TestCategoriesTableWrapsLongTagLists(t *testing.T) {
	m := minModel()
	m.width = 60 // narrow so tags are forced to wrap
	m.screen = ScreenCategories
	m.seriesData = []kalshi.Series{
		{
			Ticker:   "KXONE",
			Category: "Weather",
			Tags:     []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"},
		},
	}
	m.buildCategoryRows()
	m.applyFilter()

	rows := m.categoriesTable.Rows()
	if len(rows) < 2 {
		t.Errorf("expected multiple rows (tag wrapping), got %d", len(rows))
		return
	}
	// First row must have the category name.
	if rows[0][0] != "Weather" {
		t.Errorf("row[0] category = %q, want Weather", rows[0][0])
	}
	// Continuation rows must have empty category cell.
	for i, row := range rows[1:] {
		if row[0] != "" {
			t.Errorf("continuation row[%d] category = %q, want empty", i+1, row[0])
		}
	}
}

func TestCategoriesTableDynamicCategoryWidth(t *testing.T) {
	m := minModel()
	m.screen = ScreenCategories
	m.seriesData = []kalshi.Series{
		{Ticker: "KX1", Category: "Science & Technology", Tags: []string{"ai"}},
		{Ticker: "KX2", Category: "Sports", Tags: []string{"basketball"}},
	}
	m.buildCategoryRows()
	m.applyFilter()

	cols := m.categoriesTable.Columns()
	if len(cols) == 0 {
		t.Fatal("no columns in categories table")
	}
	// Column width must accommodate "Science & Technology" (20 chars) + padding.
	if cols[0].Width < 20 {
		t.Errorf("CATEGORY column width = %d, want >= 20 to fit 'Science & Technology'", cols[0].Width)
	}
}


func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"", "anything", true},
		{"foo", "foobar", true},
		{"foo", "bazfoo", true},
		{"foo", "bar", false},
		{"foo*", "foobar", true},
		{"foo*", "foo", true},
		{"foo*", "barfoo", false},
		{"nhl*", "nhl eastern conference", true},
		{"nhl*", "nhl", true},
		{"nhl*", "nba", false},
		{"weather", "weather", true},
		{"weather", "WEATHER", false}, // caller must lowercase
	}

	for _, tc := range tests {
		got := matchGlob(tc.pattern, tc.value)
		if got != tc.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tc.pattern, tc.value, got, tc.want)
		}
	}
}

// --- parseTerms tests ---

func TestParseTerms(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hockey", []string{"hockey"}},
		{"hockey | basketball", []string{"hockey", "basketball"}},
		{"  hockey  |  basketball  ", []string{"hockey", "basketball"}},
		{"hockey|basketball", []string{"hockey", "basketball"}},
		{"", []string{}},
		{"  ", []string{}},
		{"*", []string{"*"}},
	}

	for _, tc := range tests {
		got := parseTerms(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("parseTerms(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("parseTerms(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

// --- matchTerms tests ---

func TestMatchTerms(t *testing.T) {
	fields := map[string][]string{
		"category": {"Weather"},
		"tag":      {"Temperature", "New York"},
		"ticker":   {"KXHIGHNY"},
	}

	tests := []struct {
		terms []string
		want  bool
	}{
		// Empty terms matches everything
		{[]string{}, true},
		// Wildcard matches everything
		{[]string{"*"}, true},
		// Substring match across all fields
		{[]string{"weather"}, true},
		{[]string{"temperature"}, true},
		{[]string{"new york"}, true},
		{[]string{"kxhighny"}, true},
		{[]string{"nomatch"}, false},
		// OR semantics: any term can match
		{[]string{"nomatch", "weather"}, true},
		{[]string{"nomatch", "nomatch2"}, false},
		// Field qualifier: category:weather
		{[]string{"category:weather"}, true},
		{[]string{"category:sports"}, false},
		// Field qualifier: tag:temp*
		{[]string{"tag:temp*"}, true},
		{[]string{"tag:nfl"}, false},
		// Field qualifier on unknown field never matches
		{[]string{"unknown:weather"}, false},
		// Prefix glob on unqualified term
		{[]string{"weath*"}, true},
		{[]string{"kxhigh*"}, true},
	}

	for _, tc := range tests {
		got := matchTerms(tc.terms, fields)
		if got != tc.want {
			t.Errorf("matchTerms(%v, fields) = %v, want %v", tc.terms, got, tc.want)
		}
	}
}

func TestWrapText(t *testing.T) {
cases := []struct {
input    string
maxWidth int
want     []string
}{
{"hello world", 20, []string{"hello world"}},
{"hello world", 5, []string{"hello", "world"}},
{"a b c", 3, []string{"a b", "c"}},
{"short", 100, []string{"short"}},
{"", 20, nil},
{"one", 3, []string{"one"}},
// Long title wraps at word boundaries
{"Will the Federal Reserve cut rates in 2025", 20, []string{"Will the Federal", "Reserve cut rates in", "2025"}},
}
for _, tc := range cases {
got := wrapText(tc.input, tc.maxWidth)
if len(got) != len(tc.want) {
t.Errorf("wrapText(%q, %d) = %v, want %v", tc.input, tc.maxWidth, got, tc.want)
continue
}
for i := range got {
if got[i] != tc.want[i] {
t.Errorf("wrapText(%q, %d)[%d] = %q, want %q", tc.input, tc.maxWidth, i, got[i], tc.want[i])
}
}
}
}

func TestSeriesTableWrapsLongTitles(t *testing.T) {
m := minModel()
m.width = 80
m.height = 40
series := []kalshi.Series{
{
Ticker:    "KXTEST",
Title:     "Will the Federal Reserve cut interest rates before the end of 2025 fiscal year",
Category:  "Finance",
Frequency: "daily",
Tags:      []string{"rates", "fed"},
},
}
m.initSeriesTable(series)
rows := m.seriesTable.Rows()
if len(rows) < 2 {
t.Fatalf("expected multiple rows for long title, got %d", len(rows))
}
// First row has ticker; continuation rows have empty ticker
if rows[0][0] != "KXTEST" {
t.Errorf("first row ticker = %q, want KXTEST", rows[0][0])
}
if rows[1][0] != "" {
t.Errorf("continuation row ticker = %q, want empty", rows[1][0])
}
}

func TestSeriesTableDynamicColumnWidths(t *testing.T) {
m := minModel()
m.width = 200
m.height = 40
series := []kalshi.Series{
{Ticker: "SHORT", Title: "T", Category: "Science & Technology", Frequency: "daily", Tags: []string{"a"}},
{Ticker: "LONGERTICKER", Title: "T", Category: "Politics", Frequency: "weekly", Tags: []string{"b"}},
}
m.initSeriesTable(series)
cols := m.seriesTable.Columns()
// TICKER col should accommodate "LONGERTICKER" (12 chars) + padding
if cols[0].Width < 14 {
t.Errorf("ticker col width %d < 14, expected to fit LONGERTICKER", cols[0].Width)
}
// CATEGORY col should accommodate "Science & Technology" (20 chars) + padding
if cols[2].Width < 22 {
t.Errorf("category col width %d < 22, expected to fit Science & Technology", cols[2].Width)
}
}
