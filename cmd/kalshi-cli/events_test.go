package main_test

import (
	"strings"
	"testing"
)

func TestEventsList(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows events table", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "events", "list")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "EVENT_TICKER") || !strings.Contains(stdout, "SERIES") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25") {
			t.Errorf("expected event ticker in output, got: %q", stdout)
		}
	})

	t.Run("filter by series ticker passes param", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "events", "list", "--series-ticker", "KXHIGHNY")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "KXHIGHNY") {
			t.Errorf("expected KXHIGHNY series in output, got: %q", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "events", "list", "-o", "json")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, `"event_ticker"`) {
			t.Errorf("expected JSON output, got: %q", stdout)
		}
	})
}

func TestEventsGet(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows event details", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "events", "get", "KXHIGHNY-26APR25")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25") {
			t.Errorf("expected event ticker in output, got: %q", stdout)
		}
	})

	t.Run("missing ticker arg returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL, "events", "get")
		if exitCode == 0 {
			t.Error("expected non-zero exit when ticker arg is missing")
		}
	})

	t.Run("HTTP 404 returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 404, `{"error":{"code":"not_found","message":"event not found"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "events", "get", "NONEXISTENT-EVENT")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 404")
		}
	})
}
