package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

func newPortfolioCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "View balance, positions, and fills",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "balance",
			Short: "Get your current balance",
			RunE: func(cmd *cobra.Command, args []string) error {
				client, err := newClient()
				if err != nil {
					return err
				}
				resp, err := client.GetBalanceWithResponse(context.Background(), &kalshi.GetBalanceParams{})
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
			Use:   "positions",
			Short: "List your current positions",
			RunE: func(cmd *cobra.Command, args []string) error {
				client, err := newClient()
				if err != nil {
					return err
				}
				resp, err := client.GetPositionsWithResponse(context.Background(), nil)
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
		newFillsCmd(),
	)
	return cmd
}

func newFillsCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "fills",
		Short: "List your trade fills",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			resp, err := client.GetFillsWithResponse(context.Background(), nil)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			_ = limit
			return prettyPrint(resp.JSON200)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Max fills to return")
	return cmd
}
