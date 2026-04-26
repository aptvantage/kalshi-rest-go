package main_test

import (
	"strings"
	"testing"
)

func TestExchangeStatus(t *testing.T) {
	srv := newMockServer(t)

	t.Run("default table output", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "exchange", "status")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "EXCHANGE") || !strings.Contains(stdout, "TRADING") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "true") {
			t.Errorf("expected exchange_active=true, got: %q", stdout)
		}
	})

	t.Run("wide output includes EST_RESUME", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "exchange", "status", "-o", "wide")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "EST_RESUME") {
			t.Errorf("expected EST_RESUME header, got: %q", stdout)
		}
	})

	t.Run("json output", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "exchange", "status", "-o", "json")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, `"exchange_active"`) {
			t.Errorf("expected JSON output, got: %q", stdout)
		}
	})

	t.Run("HTTP error returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 503, `{"error":{"code":"service_unavailable","message":"maintenance"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "exchange", "status")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 503")
		}
	})
}

func TestExchangeLimits(t *testing.T) {
	srv := newMockServer(t)

	t.Run("table output shows tier and limits", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "exchange", "limits")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TIER") || !strings.Contains(stdout, "READ/s") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "standard") {
			t.Errorf("expected tier=standard, got: %q", stdout)
		}
	})

	t.Run("HTTP 401 returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 401, `{"error":{"code":"authentication_error","message":"unauthenticated"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "exchange", "limits")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 401")
		}
	})
}
