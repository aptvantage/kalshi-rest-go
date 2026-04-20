package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

func newMarketsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "markets",
		Short: "Browse markets and order books",
	}

	// markets list
	var (
		listLimit       int
		listCursor      string
		listStatus      string
		listSeriesTicker string
		listEventTicker  string
		listTickers      string
	)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List markets",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newUnauthClient()
			if err != nil {
				return err
			}
			params := kalshi.GetMarketsParams{}
			if listLimit > 0 {
				lim := kalshi.MarketLimitQuery(listLimit)
				params.Limit = &lim
			}
			if listCursor != "" {
				params.Cursor = &listCursor
			}
			if listStatus != "" {
				s := kalshi.GetMarketsParamsStatus(listStatus)
				params.Status = &s
			}
			if listSeriesTicker != "" {
				params.SeriesTicker = &listSeriesTicker
			}
			if listEventTicker != "" {
				params.EventTicker = &listEventTicker
			}
			if listTickers != "" {
				params.Tickers = &listTickers
			}
			resp, err := client.GetMarketsWithResponse(context.Background(), &params)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return prettyPrint(resp.JSON200)
		},
	}
	listCmd.Flags().IntVar(&listLimit, "limit", 20, "Max number of markets to return")
	listCmd.Flags().StringVar(&listCursor, "cursor", "", "Pagination cursor")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Market status filter: open, closed, settled")
	listCmd.Flags().StringVar(&listSeriesTicker, "series-ticker", "", "Filter by series ticker (e.g. KXBTCD)")
	listCmd.Flags().StringVar(&listEventTicker, "event-ticker", "", "Filter by event ticker")
	listCmd.Flags().StringVar(&listTickers, "tickers", "", "Comma-separated list of specific market tickers")

	// markets get <ticker>
	getCmd := &cobra.Command{
		Use:   "get <ticker>",
		Short: "Get a single market by ticker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newUnauthClient()
			if err != nil {
				return err
			}
			resp, err := client.GetMarketWithResponse(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return prettyPrint(resp.JSON200)
		},
	}

	// markets orderbook <ticker>
	orderbookCmd := &cobra.Command{
		Use:   "orderbook <ticker>",
		Short: "Get order book for a market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newUnauthClient()
			if err != nil {
				return err
			}
			resp, err := client.GetMarketOrderbookWithResponse(context.Background(), args[0], &kalshi.GetMarketOrderbookParams{})
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return prettyPrint(resp.JSON200)
		},
	}

	cmd.AddCommand(listCmd, getCmd, orderbookCmd)
	return cmd
}
