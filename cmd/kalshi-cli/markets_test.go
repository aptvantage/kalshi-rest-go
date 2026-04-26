package main_test

import (
	"strings"
	"testing"
)

func TestMarketsList(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows markets table", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "markets", "list")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") {
			t.Errorf("expected TICKER header, got: %q", stdout)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25-T51") {
			t.Errorf("expected market ticker in output, got: %q", stdout)
		}
	})

	t.Run("wide output includes extra columns", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "markets", "list", "-o", "wide")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25-T51") {
			t.Errorf("expected ticker in wide output, got: %q", stdout)
		}
	})

	t.Run("filter by status flag is accepted", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "markets", "list", "--status", "open")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") {
			t.Errorf("expected output with status filter, got: %q", stdout)
		}
	})

	t.Run("HTTP error returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 500, `{"error":{"code":"internal_error"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "markets", "list")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 500")
		}
	})
}

func TestMarketsOrderbook(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows orderbook for valid ticker", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "markets", "orderbook", "KXHIGHNY-26APR25-T51")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if len(stdout) == 0 {
			t.Error("expected non-empty output for orderbook")
		}
	})

	t.Run("missing ticker returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL, "markets", "orderbook")
		if exitCode == 0 {
			t.Error("expected non-zero exit when ticker arg is missing")
		}
	})

	t.Run("HTTP 404 returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 404, `{"error":{"code":"not_found"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "markets", "orderbook", "NONEXISTENT-MARKET")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 404")
		}
	})
}
