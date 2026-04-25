package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Browse events (dated instances of a series)",
		Long: `Events are specific instances of a series (e.g. KXHIGHNY-26APR25 = "NYC High Temp Apr 26").
Each event groups all strike markets for that date/period.

Hierarchy:  Series → Event(s) → Market(s)
Example:    KXHIGHNY → KXHIGHNY-26APR25 → KXHIGHNY-26APR25-T51

Use --with-markets to see the individual markets inside each event.`,
	}

	var (
		listSeriesTicker  string
		listStatus        string
		listMinClose      string
		listWithMarkets   bool
		listAll           bool
		listCursor        string
		listLimit         int
	)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		Long: `List events, optionally filtered by series or status.

Examples:
  kalshi-cli events list --series-ticker KXHIGHNY --status open
  kalshi-cli events list --series-ticker KXBTCD --status open --with-markets
  kalshi-cli events list --series-ticker KXHIGHNY --min-close today
  kalshi-cli events list --series-ticker KXHIGHNY --status open -o wide`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			params := kalshi.GetEventsParams{}
			if listSeriesTicker != "" {
				st := kalshi.SeriesTickerQuery(listSeriesTicker)
				params.SeriesTicker = &st
			}
			if listStatus != "" {
				s := kalshi.GetEventsParamsStatus(listStatus)
				params.Status = &s
			}
			if listMinClose != "" {
				ts, err := parseDate(listMinClose)
				if err != nil {
					return err
				}
				params.MinCloseTs = &ts
			}
			if listWithMarkets {
				t := true
				params.WithNestedMarkets = &t
			}
			if listLimit > 0 {
				params.Limit = &listLimit
			}
			if listCursor != "" {
				params.Cursor = &listCursor
			}

			var allEvents []kalshi.EventData
			var nextCursor string
			for {
				resp, err := client.GetEventsWithResponse(context.Background(), &params)
				if err != nil {
					return fmt.Errorf("request failed: %w", err)
				}
				if resp.StatusCode() != 200 {
					fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
					os.Exit(1)
				}
				allEvents = append(allEvents, resp.JSON200.Events...)
				nextCursor = resp.JSON200.Cursor
				if !listAll || nextCursor == "" {
					break
				}
				params.Cursor = &nextCursor
				time.Sleep(300 * time.Millisecond)
			}

			switch flagOutput {
			case "json", "yaml":
				type eventsResult struct {
					Events []kalshi.EventData `json:"events"`
					Cursor string             `json:"cursor,omitempty"`
				}
				return render(eventsResult{Events: allEvents, Cursor: nextCursor}, func(wide bool) ([]string, [][]string) {
					return eventsTable(allEvents, wide)
				})
			default:
				headers, rows := eventsTable(allEvents, isWide())
				if err := printTable(headers, rows); err != nil {
					return err
				}
				if nextCursor != "" {
					fmt.Fprintf(os.Stderr, "\n# %d events shown. More available — next page: --cursor %s\n", len(allEvents), nextCursor)
				} else {
					fmt.Fprintf(os.Stderr, "\n# %d events\n", len(allEvents))
				}
				// If --with-markets, also print nested market tables per event.
				if listWithMarkets && flagOutput != "json" && flagOutput != "yaml" {
					for _, e := range allEvents {
						if e.Markets == nil || len(*e.Markets) == 0 {
							continue
						}
						fmt.Printf("\n--- %s: %s ---\n", e.EventTicker, e.SubTitle)
						mh, mr := marketsTable(*e.Markets, isWide())
						if err := printTable(mh, mr); err != nil {
							return err
						}
					}
				}
				return nil
			}
		},
	}
	listCmd.Flags().StringVar(&listSeriesTicker, "series-ticker", "", "Filter by series ticker (e.g. KXHIGHNY)")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status: open, closed, settled, unopened")
	listCmd.Flags().StringVar(&listMinClose, "min-close", "", "Events with a market closing after: YYYY-MM-DD, 'today', 'tomorrow'")
	listCmd.Flags().BoolVar(&listWithMarkets, "with-markets", false, "Include nested market table for each event")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Fetch all pages automatically")
	listCmd.Flags().StringVar(&listCursor, "cursor", "", "Pagination cursor from previous response")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Max events per page (default: API default of 200)")

	getCmd := &cobra.Command{
		Use:   "get <ticker>",
		Short: "Get details for a single event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			t := true
			resp, err := client.GetEventWithResponse(context.Background(), args[0], &kalshi.GetEventParams{
				WithNestedMarkets: &t,
			})
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			ed := resp.JSON200.Event
			if err := render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return eventsTable([]kalshi.EventData{ed}, wide)
			}); err != nil {
				return err
			}
			// Always show nested markets for a single event get.
			if flagOutput != "json" && flagOutput != "yaml" && ed.Markets != nil && len(*ed.Markets) > 0 {
				fmt.Printf("\nMarkets:\n")
				mh, mr := marketsTable(*ed.Markets, isWide())
				return printTable(mh, mr)
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}

func eventsTable(events []kalshi.EventData, wide bool) ([]string, [][]string) {
	// Default: enough to find events and understand their structure.
	headers := []string{"EVENT_TICKER", "SERIES", "STATUS?", "SUBTITLE", "STRIKE_DATE"}
	if wide {
		headers = append(headers, "MUTUAL_EXCL", "COLLATERAL", "TITLE")
	}
	rows := make([][]string, 0, len(events))
	for _, e := range events {
		strikeDate := "-"
		if e.StrikeDate != nil {
			strikeDate = e.StrikeDate.UTC().Format("2006-01-02")
		} else if e.StrikePeriod != nil {
			strikeDate = *e.StrikePeriod
		}
		// Events don't carry a status field directly; infer from ticker suffix convention.
		row := []string{
			e.EventTicker,
			e.SeriesTicker,
			"-",
			truncate(e.SubTitle, 35),
			strikeDate,
		}
		if wide {
			row = append(row,
				fmtBool(e.MutuallyExclusive),
				e.CollateralReturnType,
				truncate(e.Title, 50),
			)
		}
		rows = append(rows, row)
	}
	return headers, rows
}
