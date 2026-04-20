// kalshi-cli is a command-line interface for the Kalshi Trade API.
//
// Usage:
//
//	kalshi-cli [--env prod|demo] <command> [flags]
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

	"github.com/aptvantage/kalshi-rest-go/auth"
	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

const (
	prodBase = "https://api.elections.kalshi.com/trade-api/v2"
	demoBase = "https://demo-api.kalshi.co/trade-api/v2"
)

var (
	flagEnv string
)

func main() {
	root := &cobra.Command{
		Use:   "kalshi-cli",
		Short: "CLI for the Kalshi Trade API",
		Long:  "Interact with Kalshi's Trade REST API from the command line.\n\nSet KALSHI_KEY_ID and KALSHI_KEY_FILE before use.",
	}

	root.PersistentFlags().StringVar(&flagEnv, "env", "prod", "API environment: prod or demo")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, wide, json, yaml")

	root.AddCommand(
		newExchangeCmd(),
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
