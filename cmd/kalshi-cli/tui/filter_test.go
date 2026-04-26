package tui

import (
	"testing"
)

// --- matchGlob tests ---

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
