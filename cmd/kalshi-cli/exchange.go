package main

import (
	"context"
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
				s := resp.JSON200
				return render(s, func(wide bool) ([]string, [][]string) {
					if wide {
						return []string{"EXCHANGE", "TRADING", "EST_RESUME"},
							[][]string{{fmtBool(s.ExchangeActive), fmtBool(s.TradingActive), fmtTime(s.ExchangeEstimatedResumeTime)}}
					}
					return []string{"EXCHANGE", "TRADING"},
						[][]string{{fmtBool(s.ExchangeActive), fmtBool(s.TradingActive)}}
				})
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
				l := resp.JSON200
				return render(l, func(wide bool) ([]string, [][]string) {
					return []string{"TIER", "READ/s", "WRITE/s"},
						[][]string{{l.UsageTier, fmt.Sprintf("%d", l.ReadLimit), fmt.Sprintf("%d", l.WriteLimit)}}
				})
			},
		},
	)
	return cmd
}
