package main_test

import (
	"strings"
	"testing"
)

func TestSeriesList(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows series table", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "series", "list")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") || !strings.Contains(stdout, "TITLE") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "KXHIGHNY") {
			t.Errorf("expected KXHIGHNY in output, got: %q", stdout)
		}
	})

	t.Run("wide output includes volume and tags", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "series", "list", "-o", "wide")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "VOLUME") || !strings.Contains(stdout, "TAGS") {
			t.Errorf("expected VOLUME and TAGS columns, got: %q", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "series", "list", "-o", "json")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, `"ticker"`) {
			t.Errorf("expected JSON with ticker, got: %q", stdout)
		}
	})

	t.Run("HTTP 500 returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 500, `{"error":{"code":"internal_error"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "series", "list")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 500")
		}
	})
}

func TestSeriesGet(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows single series", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "series", "get", "KXHIGHNY")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "KXHIGHNY") {
			t.Errorf("expected KXHIGHNY in output, got: %q", stdout)
		}
	})

	t.Run("missing ticker arg returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL, "series", "get")
		if exitCode == 0 {
			t.Error("expected non-zero exit when ticker arg is missing")
		}
	})
}

func TestSeriesCategories(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows categories and tags", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "series", "categories")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "CATEGORY") || !strings.Contains(stdout, "TAGS") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "weather") {
			t.Errorf("expected weather category, got: %q", stdout)
		}
	})
}
