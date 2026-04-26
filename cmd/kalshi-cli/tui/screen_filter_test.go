package tui

import (
	"sort"
	"testing"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
)

// minModel returns a minimal Model suitable for filter tests (no client needed).
func minModel() Model {
	return Model{
		screen:        ScreenCategories,
		nav:           []navEntry{{label: "categories", screen: ScreenCategories}},
		screenFilters: make(map[Screen]string),
		width:         200,
		height:        40,
		seriesSortCol: "vol",
		seriesSortAsc: false,
	}
}

// testSeries returns a fixed set of series covering two categories.
func testSeries() []kalshi.Series {
	return []kalshi.Series{
		{Ticker: "KXHIGHNY", Title: "High Temp NYC", Category: "Weather", Tags: []string{"temperature", "new york"}},
		{Ticker: "KXHIGHLA", Title: "High Temp LA", Category: "Weather", Tags: []string{"temperature", "los angeles"}},
		{Ticker: "KXNBA", Title: "NBA Winner", Category: "Sports", Tags: []string{"basketball", "nba"}},
		{Ticker: "KXNHL", Title: "NHL Winner", Category: "Sports", Tags: []string{"hockey", "nhl"}},
	}
}

// --- currentFilter ---

func TestCurrentFilterReturnsPerScreenFilter(t *testing.T) {
	m := minModel()

	// Empty by default on any screen.
	if got := m.currentFilter(); got != "" {
		t.Errorf("currentFilter() on fresh model = %q, want empty", got)
	}

	// Setting the categories filter is visible only on that screen.
	m.screenFilters[ScreenCategories] = "weather"
	if got := m.currentFilter(); got != "weather" {
		t.Errorf("currentFilter() on ScreenCategories = %q, want %q", got, "weather")
	}

	// Switching to series screen sees its own (empty) filter, not the parent's.
	m.screen = ScreenSeriesList
	if got := m.currentFilter(); got != "" {
		t.Errorf("currentFilter() on ScreenSeriesList = %q, want empty (isolated from category filter)", got)
	}

	// Set a series-level filter; category filter must be unaffected.
	m.screenFilters[ScreenSeriesList] = "new york"
	if got := m.currentFilter(); got != "new york" {
		t.Errorf("currentFilter() on ScreenSeriesList = %q, want %q", got, "new york")
	}
	m.screen = ScreenCategories
	if got := m.currentFilter(); got != "weather" {
		t.Errorf("currentFilter() on ScreenCategories after series filter set = %q, want %q", got, "weather")
	}
}

// --- applyFilter: ScreenSeriesList AND logic ---

func TestApplyFilterSeriesAndLogic(t *testing.T) {
	tests := []struct {
		name           string
		categoryFilter string
		seriesFilter   string
		wantTickers    []string
	}{
		{
			name:           "both empty — all series shown",
			categoryFilter: "",
			seriesFilter:   "",
			wantTickers:    []string{"KXHIGHNY", "KXHIGHLA", "KXNBA", "KXNHL"},
		},
		{
			name:           "category filter only — weather",
			categoryFilter: "weather",
			seriesFilter:   "",
			wantTickers:    []string{"KXHIGHNY", "KXHIGHLA"},
		},
		{
			name:           "series filter only — new york",
			categoryFilter: "",
			seriesFilter:   "new york",
			wantTickers:    []string{"KXHIGHNY"},
		},
		{
			name:           "both filters AND — weather + new york",
			categoryFilter: "weather",
			seriesFilter:   "new york",
			wantTickers:    []string{"KXHIGHNY"},
		},
		{
			name:           "category matches but series filter excludes all",
			categoryFilter: "weather",
			seriesFilter:   "basketball",
			wantTickers:    []string{},
		},
		{
			name:           "series matches but category filter excludes all",
			categoryFilter: "politics",
			seriesFilter:   "new york",
			wantTickers:    []string{},
		},
		{
			name:           "wildcard category filter passes everything through",
			categoryFilter: "*",
			seriesFilter:   "",
			wantTickers:    []string{"KXHIGHNY", "KXHIGHLA", "KXNBA", "KXNHL"},
		},
		{
			name:           "field-qualified category filter",
			categoryFilter: "category:sports",
			seriesFilter:   "",
			wantTickers:    []string{"KXNBA", "KXNHL"},
		},
		{
			name:           "field-qualified filters on both levels — sports + hockey",
			categoryFilter: "category:sports",
			seriesFilter:   "tag:hockey",
			wantTickers:    []string{"KXNHL"},
		},
		{
			name:           "OR in category filter — weather | sports",
			categoryFilter: "weather | sports",
			seriesFilter:   "",
			wantTickers:    []string{"KXHIGHNY", "KXHIGHLA", "KXNBA", "KXNHL"},
		},
		{
			name:           "OR in series filter — new york | los angeles",
			categoryFilter: "weather",
			seriesFilter:   "new york | los angeles",
			wantTickers:    []string{"KXHIGHNY", "KXHIGHLA"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := minModel()
			m.screen = ScreenSeriesList
			m.seriesData = testSeries()
			m.screenFilters[ScreenCategories] = tc.categoryFilter
			m.screenFilters[ScreenSeriesList] = tc.seriesFilter
			m.applyFilter()

			rows := m.seriesTable.Rows()
			// Collect non-empty tickers (skip continuation rows from multi-row wrap).
			gotTickers := make([]string, 0, len(rows))
			for _, row := range rows {
				if row[0] != "" {
					gotTickers = append(gotTickers, row[0])
				}
			}
			if len(gotTickers) != len(tc.wantTickers) {
				t.Errorf("got tickers %v, want %v", gotTickers, tc.wantTickers)
				return
			}
			// Sort both for order-independent comparison (filter logic, not sort logic).
			sort.Strings(gotTickers)
			wantSorted := make([]string, len(tc.wantTickers))
			copy(wantSorted, tc.wantTickers)
			sort.Strings(wantSorted)
			for i := range gotTickers {
				if gotTickers[i] != wantSorted[i] {
					t.Errorf("ticker set mismatch: got %v, want %v", gotTickers, wantSorted)
					break
				}
			}
		})
	}
}

// --- applyFilter: ScreenCategories is independent of series-level filter ---

func TestApplyFilterCategoriesIgnoresSeriesFilter(t *testing.T) {
	m := minModel()
	m.screen = ScreenCategories
	m.seriesData = testSeries()
	m.buildCategoryRows()

	// A series-level filter must not leak into the categories screen.
	m.screenFilters[ScreenSeriesList] = "basketball"
	m.screenFilters[ScreenCategories] = "weather"
	m.applyFilter()

	rows := m.categoriesTable.Rows()
	if len(rows) != 1 {
		names := make([]string, len(rows))
		for i, r := range rows {
			names[i] = r[0]
		}
		t.Errorf("got %d category rows %v, want 1 (Weather only)", len(rows), names)
		return
	}
	if rows[0][0] != "Weather" {
		t.Errorf("category row[0] = %q, want Weather", rows[0][0])
	}
}

// --- applyFilter: ScreenSeriesList with empty seriesData ---

func TestApplyFilterSeriesEmptyData(t *testing.T) {
	m := minModel()
	m.screen = ScreenSeriesList
	m.seriesData = []kalshi.Series{}
	m.screenFilters[ScreenCategories] = "weather"
	m.screenFilters[ScreenSeriesList] = "temperature"
	m.applyFilter()

	if rows := m.seriesTable.Rows(); len(rows) != 0 {
		t.Errorf("expected 0 rows for empty seriesData, got %d", len(rows))
	}
}

// --- sort ---

func strPtr(s string) *kalshi.FixedPointCount { return &s }

func TestSortSeriesByVolDesc(t *testing.T) {
m := minModel()
m.seriesSortCol = "vol"
m.seriesSortAsc = false
s := []kalshi.Series{
{Ticker: "LOW", VolumeFp: strPtr("100.00")},
{Ticker: "HIGH", VolumeFp: strPtr("50000.00")},
{Ticker: "MED", VolumeFp: strPtr("1000.00")},
}
m.sortSeries(s)
want := []string{"HIGH", "MED", "LOW"}
for i, want := range want {
if s[i].Ticker != want {
t.Errorf("pos %d = %q, want %q", i, s[i].Ticker, want)
}
}
}

func TestSortSeriesByVolAsc(t *testing.T) {
m := minModel()
m.seriesSortCol = "vol"
m.seriesSortAsc = true
s := []kalshi.Series{
{Ticker: "LOW", VolumeFp: strPtr("100.00")},
{Ticker: "HIGH", VolumeFp: strPtr("50000.00")},
{Ticker: "MED", VolumeFp: strPtr("1000.00")},
}
m.sortSeries(s)
want := []string{"LOW", "MED", "HIGH"}
for i, want := range want {
if s[i].Ticker != want {
t.Errorf("pos %d = %q, want %q", i, s[i].Ticker, want)
}
}
}

func TestSortSeriesByTitleAsc(t *testing.T) {
m := minModel()
m.seriesSortCol = "title"
m.seriesSortAsc = true
s := []kalshi.Series{
{Ticker: "C", Title: "Zebra"},
{Ticker: "A", Title: "Apple"},
{Ticker: "B", Title: "Mango"},
}
m.sortSeries(s)
want := []string{"A", "B", "C"}
for i, want := range want {
if s[i].Ticker != want {
t.Errorf("pos %d = %q, want %q", i, s[i].Ticker, want)
}
}
}

func TestSortSeriesTickerTiebreaker(t *testing.T) {
// When primary key is equal, ticker tiebreaker should produce stable desc order.
m := minModel()
m.seriesSortCol = "vol"
m.seriesSortAsc = false // desc; tiebreaker also desc
s := []kalshi.Series{
{Ticker: "AAA", VolumeFp: strPtr("100.00")},
{Ticker: "ZZZ", VolumeFp: strPtr("100.00")},
{Ticker: "MMM", VolumeFp: strPtr("100.00")},
}
m.sortSeries(s)
// All volumes equal → tiebreaker is ticker descending.
want := []string{"ZZZ", "MMM", "AAA"}
for i, want := range want {
if s[i].Ticker != want {
t.Errorf("pos %d = %q, want %q", i, s[i].Ticker, want)
}
}
}

func TestNextSeriesSortCol(t *testing.T) {
m := minModel()
m.seriesSortCol = "vol"
m.seriesSortAsc = false

m.nextSeriesSortCol()
if m.seriesSortCol != "title" {
t.Errorf("after vol, got %q, want title", m.seriesSortCol)
}
if !m.seriesSortAsc {
t.Errorf("title should default to asc")
}

m.nextSeriesSortCol()
if m.seriesSortCol != "ticker" {
t.Errorf("after title, got %q, want ticker", m.seriesSortCol)
}

m.nextSeriesSortCol()
m.nextSeriesSortCol()
// Should have cycled back to vol.
if m.seriesSortCol != "vol" {
t.Errorf("after full cycle, got %q, want vol", m.seriesSortCol)
}
if m.seriesSortAsc {
t.Errorf("vol should default to desc (asc=false)")
}
}

func TestFmtVolume(t *testing.T) {
cases := []struct {
input string
want  string
}{
{"0.00", "-"},
{"500.00", "500"},
{"1500.00", "1.5K"},
{"1500000.00", "1.5M"},
}
for _, tc := range cases {
got := fmtVolume(&tc.input)
if got != tc.want {
t.Errorf("fmtVolume(%q) = %q, want %q", tc.input, got, tc.want)
}
}
}

func TestFmtFee(t *testing.T) {
cases := []struct {
ft   kalshi.SeriesFeeType
mult float64
want string
}{
{"quadratic_with_maker_fees", 1.0, "maker"},
{"quadratic", 0.8, "quad×0.8"},
{"flat", 1.0, "flat"},
{"quadratic_with_maker_fees", 0.5, "maker×0.5"},
}
for _, tc := range cases {
got := fmtFee(tc.ft, tc.mult)
if got != tc.want {
t.Errorf("fmtFee(%q, %g) = %q, want %q", tc.ft, tc.mult, got, tc.want)
}
}
}
