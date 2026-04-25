// kalshi-cli is a command-line interface for the Kalshi Trade API.
//
// Usage:
//
//	kalshi-cli [--env prod|demo] <command> [flags]
//	kalshi-cli                              # launch interactive TUI
//
// Authentication is configured via environment variables:
//
//	KALSHI_KEY_ID      — your API key ID
//	KALSHI_KEY_FILE    — path to your RSA private key PEM file
//
// Commands: exchange, markets, orders, portfolio
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aptvantage/kalshi-rest-go/auth"
	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/aptvantage/kalshi-rest-go/cmd/kalshi-cli/tui"
	"github.com/spf13/cobra"
)

const (
	prodBase = "https://api.elections.kalshi.com/trade-api/v2"
	demoBase = "https://demo-api.kalshi.co/trade-api/v2"
)

var (
	flagEnv   string
	flagDebug string // path to debug log file; empty = no logging
)

func main() {
	root := &cobra.Command{
		Use:   "kalshi-cli",
		Short: "CLI for the Kalshi Trade API",
		Long:  "Interact with Kalshi's Trade REST API from the command line.\n\nSet KALSHI_KEY_ID and KALSHI_KEY_FILE before use.",
		// Running with no subcommand launches the interactive TUI.
		RunE: func(cmd *cobra.Command, args []string) error {
			// Don't show usage text when RunE returns an error — usage is
			// only helpful for flag/argument mistakes, not runtime failures.
			cmd.SilenceUsage = true

			// Wire up debug log file when --debug is set.
			// tea.LogToFile must be called before tea.NewProgram; it redirects
			// the standard log package to the file so any log.Printf calls in
			// TUI code appear there without corrupting the terminal.
			if flagDebug != "" {
				if err := os.MkdirAll(filepath.Dir(flagDebug), 0o755); err != nil {
					return fmt.Errorf("create debug log directory: %w", err)
				}
				f, err := tea.LogToFile(flagDebug, "")
				if err != nil {
					return fmt.Errorf("open debug log %q: %w", flagDebug, err)
				}
				defer f.Close()
			}

			// Try authenticated client first; fall back to public-only access.
			// Series, events, markets, and orderbook are all public endpoints.
			// Balance and order entry require credentials.
			client, authErr := newClient()
			authenticated := authErr == nil
			if !authenticated {
				var err error
				client, err = newUnauthClient()
				if err != nil {
					return fmt.Errorf("failed to create API client: %w", err)
				}
			}

			p := tea.NewProgram(
				tui.New(client, flagEnv, authenticated),
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)
			_, err := p.Run()
			return err
		},
	}

	root.PersistentFlags().StringVar(&flagEnv, "env", "prod", "API environment: prod or demo")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, wide, json, yaml")
	root.Flags().StringVar(&flagDebug, "debug", "", "Write TUI debug log to this file (e.g. --debug /tmp/kalshi-debug.log)")

	root.AddCommand(
		newExchangeCmd(),
		newSeriesCmd(),
		newEventsCmd(),
		newMarketsCmd(),
		newOrdersCmd(),
		newPortfolioCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// newClient builds an authenticated Kalshi API client for the selected environment.
// Auth is configured via environment variables:
//
//	KALSHI_KEY_ID      — API key ID (required)
//	KALSHI_KEY_FILE    — path to RSA private key PEM file (optional if KALSHI_PRIVATE_KEY is set)
//	KALSHI_PRIVATE_KEY — PEM-encoded RSA private key (alternative to KALSHI_KEY_FILE)
func newClient() (*kalshi.ClientWithResponses, error) {
	keyID := os.Getenv("KALSHI_KEY_ID")
	if keyID == "" {
		return nil, fmt.Errorf("KALSHI_KEY_ID must be set")
	}

	var signer *auth.Signer
	var err error

	if keyPEM := os.Getenv("KALSHI_PRIVATE_KEY"); keyPEM != "" {
		signer, err = auth.NewSignerFromPEM(keyID, []byte(keyPEM))
	} else if keyFile := os.Getenv("KALSHI_KEY_FILE"); keyFile != "" {
		signer, err = auth.NewSignerFromFile(keyID, keyFile)
	} else {
		return nil, fmt.Errorf("KALSHI_KEY_FILE or KALSHI_PRIVATE_KEY must be set")
	}
	if err != nil {
		return nil, fmt.Errorf("load signing key: %w", err)
	}

	return kalshi.NewClientWithResponses(baseURL(), kalshi.WithHTTPClient(auth.NewClient(signer)))
}

// newUnauthClient creates an unauthenticated client for public endpoints.
func newUnauthClient() (*kalshi.ClientWithResponses, error) {
	return kalshi.NewClientWithResponses(baseURL(), kalshi.WithHTTPClient(&http.Client{}))
}

func baseURL() string {
	if flagEnv == "demo" {
		return demoBase
	}
	return prodBase
}
