package main_test

import (
	"strings"
	"testing"
)

func TestOrdersList(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows orders table", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "orders", "list")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") || !strings.Contains(stdout, "SIDE") {
			t.Errorf("expected headers, got: %q", stdout)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25-T51") {
			t.Errorf("expected ticker in output, got: %q", stdout)
		}
	})

	t.Run("filter by status flag is accepted", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "orders", "list", "--status", "resting")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "TICKER") {
			t.Errorf("expected output with status filter, got: %q", stdout)
		}
	})

	t.Run("HTTP 401 returns non-zero exit", func(t *testing.T) {
		errSrv := newErrorServer(t, 401, `{"error":{"code":"authentication_error"}}`)
		_, _, exitCode := runCLI(t, errSrv.URL, "orders", "list")
		if exitCode == 0 {
			t.Error("expected non-zero exit on HTTP 401")
		}
	})
}

func TestOrdersCreate(t *testing.T) {
	srv := newMockServer(t)

	t.Run("creates order and shows result", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL,
			"orders", "create",
			"--ticker", "KXHIGHNY-26APR25-T51",
			"--side", "yes",
			"--action", "buy",
			"--count", "1",
			"--yes-price", "45",
		)
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25-T51") {
			t.Errorf("expected ticker in output, got: %q", stdout)
		}
	})

	t.Run("missing required ticker flag returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL,
			"orders", "create",
			"--side", "yes",
			"--action", "buy",
			"--yes-price", "45",
		)
		if exitCode == 0 {
			t.Error("expected non-zero exit when --ticker is missing")
		}
	})

	t.Run("invalid side returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL,
			"orders", "create",
			"--ticker", "KXHIGHNY-26APR25-T51",
			"--side", "invalid",
			"--action", "buy",
			"--yes-price", "45",
		)
		if exitCode == 0 {
			t.Error("expected non-zero exit with invalid side")
		}
	})

	t.Run("missing price flags returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL,
			"orders", "create",
			"--ticker", "KXHIGHNY-26APR25-T51",
			"--side", "yes",
			"--action", "buy",
		)
		if exitCode == 0 {
			t.Error("expected non-zero exit when both price flags are missing")
		}
	})
}

func TestOrdersGet(t *testing.T) {
	srv := newMockServer(t)

	t.Run("shows single order", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "orders", "get", "order-abc-000001")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "KXHIGHNY-26APR25-T51") {
			t.Errorf("expected ticker in output, got: %q", stdout)
		}
	})

	t.Run("missing order ID returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL, "orders", "get")
		if exitCode == 0 {
			t.Error("expected non-zero exit when order ID is missing")
		}
	})
}

func TestOrdersCancel(t *testing.T) {
	srv := newMockServer(t)

	t.Run("cancels order and shows canceled status", func(t *testing.T) {
		stdout, _, exitCode := runCLI(t, srv.URL, "orders", "cancel", "order-abc-000001")
		if exitCode != 0 {
			t.Fatalf("exit code %d", exitCode)
		}
		if !strings.Contains(stdout, "canceled") {
			t.Errorf("expected canceled status in output, got: %q", stdout)
		}
	})

	t.Run("missing order ID returns error", func(t *testing.T) {
		_, _, exitCode := runCLI(t, srv.URL, "orders", "cancel")
		if exitCode == 0 {
			t.Error("expected non-zero exit when order ID is missing")
		}
	})
}
