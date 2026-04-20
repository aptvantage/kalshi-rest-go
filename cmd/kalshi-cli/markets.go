package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

func newMarketsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "markets",
		Short: "Browse markets and order books",
	}

	var (
		listLimit        int
		listCursor       string
		listStatus       string
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
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return marketsTable(resp.JSON200.Markets, wide)
			})
		},
	}
	listCmd.Flags().IntVar(&listLimit, "limit", 20, "Max number of markets to return")
	listCmd.Flags().StringVar(&listCursor, "cursor", "", "Pagination cursor")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Market status filter: open, closed, settled")
	listCmd.Flags().StringVar(&listSeriesTicker, "series-ticker", "", "Filter by series ticker (e.g. KXBTCD)")
	listCmd.Flags().StringVar(&listEventTicker, "event-ticker", "", "Filter by event ticker")
	listCmd.Flags().StringVar(&listTickers, "tickers", "", "Comma-separated list of specific market tickers")

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
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return marketsTable([]kalshi.Market{resp.JSON200.Market}, wide)
			})
		},
	}

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
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return orderbookTable(resp.JSON200.OrderbookFp, wide)
			})
		},
	}

	cmd.AddCommand(listCmd, getCmd, orderbookCmd)
	return cmd
}

func marketsTable(markets []kalshi.Market, wide bool) ([]string, [][]string) {
	headers := []string{"TICKER", "STATUS", "YES_BID", "YES_ASK", "SPREAD", "CLOSE"}
	if wide {
		headers = append(headers, "VOL_24H", "OPEN_INT", "LAST", "EVENT")
	}
	rows := make([][]string, 0, len(markets))
	for _, m := range markets {
		row := []string{
			m.Ticker,
			string(m.Status),
			fmtCents(string(m.YesBidDollars)),
			fmtCents(string(m.YesAskDollars)),
			fmtSpread(string(m.YesBidDollars), string(m.YesAskDollars)),
			fmtTimeVal(m.CloseTime),
		}
		if wide {
			row = append(row,
				m.Volume24hFp,
				m.OpenInterestFp,
				fmtCents(string(m.LastPriceDollars)),
				m.EventTicker,
			)
		}
		rows = append(rows, row)
	}
	return headers, rows
}

func orderbookTable(ob kalshi.OrderbookCountFp, wide bool) ([]string, [][]string) {
	headers := []string{"SIDE", "PRICE", "SIZE"}
	var rows [][]string

	// YES side — sort descending by price (best bid first)
	yesSide := make([]kalshi.PriceLevelDollarsCountFp, len(ob.YesDollars))
	copy(yesSide, ob.YesDollars)
	sort.Slice(yesSide, func(i, j int) bool {
		return parseFP(yesSide[i][0]) > parseFP(yesSide[j][0])
	})
	limit := 5
	if wide {
		limit = 10
	}
	for i, level := range yesSide {
		if i >= limit {
			break
		}
		if len(level) < 2 {
			continue
		}
		rows = append(rows, []string{"YES", fmtCents(level[0]), level[1]})
	}

	// NO side — sort ascending by price (best NO ask first = lowest price)
	noSide := make([]kalshi.PriceLevelDollarsCountFp, len(ob.NoDollars))
	copy(noSide, ob.NoDollars)
	sort.Slice(noSide, func(i, j int) bool {
		return parseFP(noSide[i][0]) < parseFP(noSide[j][0])
	})
	for i, level := range noSide {
		if i >= limit {
			break
		}
		if len(level) < 2 {
			continue
		}
		rows = append(rows, []string{"NO", fmtCents(level[0]), level[1]})
	}

	return headers, rows
}
