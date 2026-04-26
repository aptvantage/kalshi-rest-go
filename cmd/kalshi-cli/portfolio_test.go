package main_test

import (
	"strings"
	"testing"
)

func TestPortfolioBalance(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows balance and portfolio value", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "portfolio", "balance")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "BALANCE") || !strings.Contains(stdout, "PORTFOLIO_VALUE") {
			t.Errorf("expected headers, got: %q", stdout)
		}
	})

	t.Run("json output includes balance field", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "portfolio", "balance", "-o", "json")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, `"balance"`) {
			t.Errorf("expected JSON with balance field, got: %q", stdout)
		}
	})

	t.Run("HTTP 401 returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 401, `{"error":{"code":"authentication_error"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "portfolio", "balance")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 401")
		}
	})
}

func TestPortfolioPositions(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows positions table", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "portfolio", "positions")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") || !strings.Contains(stdout, "POSITION") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25-T51") {
			t.Errorf("expected ticker in output, got: %q", stdout)
		}
	})

	t.Run("wide output includes extra columns", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "portfolio", "positions", "-o", "wide")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "FEES_PAID") {
			t.Errorf("expected FEES_PAID in wide output, got: %q", stdout)
		}
	})
}

func TestPortfolioFills(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows fills table", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "portfolio", "fills")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") || !strings.Contains(stdout, "SIDE") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "yes") {
			t.Errorf("expected side=yes in output, got: %q", stdout)
		}
	})
}
