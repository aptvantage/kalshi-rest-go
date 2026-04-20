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
				b := resp.JSON200
				return render(b, func(wide bool) ([]string, [][]string) {
					return []string{"BALANCE", "PORTFOLIO_VALUE"},
						[][]string{{fmtDollars(b.Balance), fmtDollars(b.PortfolioValue)}}
				})
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
				return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
					return positionsTable(resp.JSON200.MarketPositions, wide)
				})
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
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return fillsTable(resp.JSON200.Fills, wide)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "Max fills to return")
	return cmd
}

func positionsTable(positions []kalshi.MarketPosition, wide bool) ([]string, [][]string) {
	headers := []string{"TICKER", "POSITION", "EXPOSURE", "REALIZED_PNL"}
	if wide {
		headers = append(headers, "FEES_PAID", "TOTAL_TRADED", "LAST_UPDATED")
	}
	rows := make([][]string, 0, len(positions))
	for _, p := range positions {
		row := []string{
			p.Ticker,
			p.PositionFp,
			fmtCents(string(p.MarketExposureDollars)),
			fmtCents(string(p.RealizedPnlDollars)),
		}
		if wide {
			row = append(row,
				fmtCents(string(p.FeesPaidDollars)),
				fmtCents(string(p.TotalTradedDollars)),
				fmtTimeVal(p.LastUpdatedTs),
			)
		}
		rows = append(rows, row)
	}
	return headers, rows
}

func fillsTable(fills []kalshi.Fill, wide bool) ([]string, [][]string) {
	headers := []string{"TICKER", "SIDE", "ACTION", "PRICE", "COUNT", "CREATED_AT"}
	if wide {
		headers = append(headers, "FILL_ID", "ORDER_ID", "FEE_COST", "IS_TAKER")
	}
	rows := make([][]string, 0, len(fills))
	for _, f := range fills {
		price := fmtCents(string(f.YesPriceDollars))
		if f.Side == "no" {
			price = fmtCents(string(f.NoPriceDollars))
		}
		row := []string{
			f.Ticker,
			string(f.Side),
			string(f.Action),
			price,
			f.CountFp,
			fmtTime(f.CreatedTime),
		}
		if wide {
			row = append(row,
				shortID(f.FillId),
				shortID(f.OrderId),
				fmtCents(string(f.FeeCost)),
				fmtBool(f.IsTaker),
			)
		}
		rows = append(rows, row)
	}
	return headers, rows
}
