package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

// parseDate accepts "today", "tomorrow", or "YYYY-MM-DD" and returns a Unix timestamp.
func parseDate(s string) (int64, error) {
	now := time.Now()
	switch strings.ToLower(s) {
	case "today":
		y, m, d := now.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix(), nil
	case "tomorrow":
		y, m, d := now.AddDate(0, 0, 1).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix(), nil
	default:
		t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
		if err != nil {
			return 0, fmt.Errorf("invalid date %q: use YYYY-MM-DD, 'today', or 'tomorrow'", s)
		}
		return t.Unix(), nil
	}
}

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
		listMinClose     string
		listMaxClose     string
		listMveFilter    string
		listSearch       string
		listAll          bool
	)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List markets",
		Long: `List markets with optional filters.

Filter examples:
  kalshi-cli markets list --status open --series-ticker KXBTCD
  kalshi-cli markets list --status open --min-close today --max-close 2026-05-01
  kalshi-cli markets list --search "NYC" --status open
  kalshi-cli markets list --all --status open --series-ticker KXHIGHNY
  kalshi-cli markets list --mve-filter exclude --status open   # hide combo markets`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}

			params := kalshi.GetMarketsParams{}
			lim := kalshi.MarketLimitQuery(listLimit)
			params.Limit = &lim

			if listCursor != "" {
				c := kalshi.CursorQuery(listCursor)
				params.Cursor = &c
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
			if listMinClose != "" {
				ts, err := parseDate(listMinClose)
				if err != nil {
					return err
				}
				mcts := kalshi.MinCloseTsQuery(ts)
				params.MinCloseTs = &mcts
			}
			if listMaxClose != "" {
				ts, err := parseDate(listMaxClose)
				if err != nil {
					return err
				}
				mcts := kalshi.MaxCloseTsQuery(ts)
				params.MaxCloseTs = &mcts
			}
			if listMveFilter != "" {
				f := kalshi.GetMarketsParamsMveFilter(listMveFilter)
				params.MveFilter = &f
			}

			search := strings.ToLower(listSearch)

			// Collect pages; loop only when --all is set.
			var allMarkets []kalshi.Market
			var nextCursor string
			for {
				resp, err := client.GetMarketsWithResponse(context.Background(), &params)
				if err != nil {
					return fmt.Errorf("request failed: %w", err)
				}
				if resp.StatusCode() != 200 {
					fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
					os.Exit(1)
				}
				page := resp.JSON200.Markets
				if search != "" {
					filtered := page[:0]
					for _, m := range page {
						if strings.Contains(strings.ToLower(m.Ticker), search) ||
							strings.Contains(strings.ToLower(m.YesSubTitle), search) ||
							strings.Contains(strings.ToLower(m.NoSubTitle), search) {
							filtered = append(filtered, m)
						}
					}
					page = filtered
				}
				allMarkets = append(allMarkets, page...)
				nextCursor = resp.JSON200.Cursor
				if !listAll || nextCursor == "" {
					break
				}
				c := kalshi.CursorQuery(nextCursor)
				params.Cursor = &c
				time.Sleep(300 * time.Millisecond) // avoid rate limiting during auto-pagination
			}

			// For structured output use the last raw response; for table render all collected markets.
			switch flagOutput {
			case "json", "yaml":
				type marketsResult struct {
					Markets []kalshi.Market `json:"markets"`
					Cursor  string          `json:"cursor,omitempty"`
				}
				return render(marketsResult{Markets: allMarkets, Cursor: nextCursor}, func(wide bool) ([]string, [][]string) {
					return marketsTable(allMarkets, wide)
				})
			default:
				headers, rows := marketsTable(allMarkets, isWide())
				if err := printTable(headers, rows); err != nil {
					return err
				}
				if nextCursor != "" {
					fmt.Fprintf(os.Stderr, "\n# %d markets shown. More available — next page: --cursor %s\n", len(allMarkets), nextCursor)
				} else {
					fmt.Fprintf(os.Stderr, "\n# %d markets\n", len(allMarkets))
				}
				return nil
			}
		},
	}
	listCmd.Flags().IntVar(&listLimit, "limit", 100, "Max markets per page (1–1000, default 100)")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Fetch all pages automatically (ignores --cursor)")
	listCmd.Flags().StringVar(&listCursor, "cursor", "", "Pagination cursor from previous response")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Market status: open, closed, settled, paused, unopened")
	listCmd.Flags().StringVar(&listSeriesTicker, "series-ticker", "", "Filter by series ticker (e.g. KXBTCD)")
	listCmd.Flags().StringVar(&listEventTicker, "event-ticker", "", "Filter by event ticker")
	listCmd.Flags().StringVar(&listTickers, "tickers", "", "Comma-separated market tickers")
	listCmd.Flags().StringVar(&listMinClose, "min-close", "", "Earliest close date: YYYY-MM-DD, 'today', or 'tomorrow'")
	listCmd.Flags().StringVar(&listMaxClose, "max-close", "", "Latest close date: YYYY-MM-DD, 'today', or 'tomorrow'")
	listCmd.Flags().StringVar(&listMveFilter, "mve-filter", "", "Multivariate event filter: exclude or only")
	listCmd.Flags().StringVar(&listSearch, "search", "", "Substring search on ticker/subtitle (current page only; combine with --all to scan all pages)")

	getCmd := &cobra.Command{
		Use:   "get <ticker>",
		Short: "Get a single market by ticker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
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
			client, err := newClient()
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
	// Default: the 7 fields most useful for scanning LP opportunities at a glance.
	// YES_BID and YES_ASK show price¢×size so you can assess depth in one column.
	// VOL_24H is the primary liquidity signal.
	headers := []string{"TICKER", "STATUS", "YES_BID", "YES_ASK", "SPREAD", "VOL_24H", "CLOSE"}
	if wide {
		// Wide adds per-side depth sizes, total open interest, liquidity $, last trade, and event grouping.
		headers = append(headers, "BID_SZ", "ASK_SZ", "OPEN_INT", "LIQUIDITY", "LAST", "EVENT")
	}
	rows := make([][]string, 0, len(markets))
	for _, m := range markets {
		row := []string{
			m.Ticker,
			string(m.Status),
			fmtCents(string(m.YesBidDollars)),
			fmtCents(string(m.YesAskDollars)),
			fmtSpread(string(m.YesBidDollars), string(m.YesAskDollars)),
			m.Volume24hFp,
			fmtTimeVal(m.CloseTime),
		}
		if wide {
			row = append(row,
				m.YesBidSizeFp,
				m.YesAskSizeFp,
				m.OpenInterestFp,
				fmtCents(string(m.LiquidityDollars)),
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
