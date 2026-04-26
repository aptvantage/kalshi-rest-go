package tui

import (
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
			if len(rows) != len(tc.wantTickers) {
				got := make([]string, len(rows))
				for i, r := range rows {
					got[i] = r[0]
				}
				t.Errorf("got %d rows %v, want %d %v", len(rows), got, len(tc.wantTickers), tc.wantTickers)
				return
			}
			for i, row := range rows {
				if row[0] != tc.wantTickers[i] {
					t.Errorf("row[%d] ticker = %q, want %q", i, row[0], tc.wantTickers[i])
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
