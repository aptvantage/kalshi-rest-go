package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newExchangeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exchange",
		Short: "Exchange-level information",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "status",
			Short: "Get exchange status",
			RunE: func(cmd *cobra.Command, args []string) error {
				client, err := newUnauthClient()
				if err != nil {
					return err
				}
				resp, err := client.GetExchangeStatusWithResponse(context.Background())
				if err != nil {
					return fmt.Errorf("request failed: %w", err)
				}
				if resp.StatusCode() != 200 {
					fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
					os.Exit(1)
				}
				return prettyPrint(resp.JSON200)
			},
		},
		&cobra.Command{
			Use:   "limits",
			Short: "Get your API rate limits",
			RunE: func(cmd *cobra.Command, args []string) error {
				client, err := newClient()
				if err != nil {
					return err
				}
				resp, err := client.GetAccountApiLimitsWithResponse(context.Background())
				if err != nil {
					return fmt.Errorf("request failed: %w", err)
				}
				if resp.StatusCode() != 200 {
					fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
					os.Exit(1)
				}
				return prettyPrint(resp.JSON200)
			},
		},
	)
	return cmd
}

func prettyPrint(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
