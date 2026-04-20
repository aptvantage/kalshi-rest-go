package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

func newOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders",
		Short: "Manage orders",
	}
	cmd.AddCommand(
		newOrdersListCmd(),
		newOrdersCreateCmd(),
		newOrdersGetCmd(),
		newOrdersCancelCmd(),
	)
	return cmd
}

func newOrdersListCmd() *cobra.Command {
	var (
		ticker string
		limit  int
		status string
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your orders",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			params := kalshi.GetOrdersParams{}
			if ticker != "" {
				params.Ticker = &ticker
			}
			if limit > 0 {
				lim := kalshi.LimitQuery(limit)
				params.Limit = &lim
			}
			if status != "" {
				params.Status = &status
			}
			resp, err := client.GetOrdersWithResponse(context.Background(), &params)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return ordersTable(resp.JSON200.Orders, wide)
			})
		},
	}
	cmd.Flags().StringVar(&ticker, "ticker", "", "Filter by market ticker")
	cmd.Flags().IntVar(&limit, "limit", 50, "Max orders to return")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status: resting, canceled, executed")
	return cmd
}

func newOrdersCreateCmd() *cobra.Command {
	var (
		ticker   string
		side     string
		action   string
		count    int
		yesPrice int
		noPrice  int
		postOnly bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new order",
		Long: `Create a limit order on Kalshi.

Prices are in cents (1–99), representing the probability percentage.
Example — buy 1 YES contract at 45¢ on a market:

  kalshi-cli orders create --ticker KXBTC-25DEC-T30000 --side yes --action buy --count 1 --yes-price 45`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ticker == "" {
				return fmt.Errorf("--ticker is required")
			}
			if side != "yes" && side != "no" {
				return fmt.Errorf("--side must be 'yes' or 'no'")
			}
			if action != "buy" && action != "sell" {
				return fmt.Errorf("--action must be 'buy' or 'sell'")
			}
			if count <= 0 {
				return fmt.Errorf("--count must be >= 1")
			}
			if yesPrice == 0 && noPrice == 0 {
				return fmt.Errorf("one of --yes-price or --no-price is required")
			}

			client, err := newClient()
			if err != nil {
				return err
			}

			body := kalshi.CreateOrderRequest{
				Ticker: ticker,
				Side:   kalshi.CreateOrderRequestSide(side),
				Action: kalshi.CreateOrderRequestAction(action),
				Count:  &count,
			}
			if postOnly {
				body.PostOnly = &postOnly
			}
			if yesPrice > 0 {
				body.YesPrice = &yesPrice
			}
			if noPrice > 0 {
				body.NoPrice = &noPrice
			}

			resp, err := client.CreateOrderWithResponse(context.Background(), body)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 201 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return render(resp.JSON201, func(wide bool) ([]string, [][]string) {
				return ordersTable([]kalshi.Order{resp.JSON201.Order}, wide)
			})
		},
	}
	cmd.Flags().StringVar(&ticker, "ticker", "", "Market ticker (required)")
	cmd.Flags().StringVar(&side, "side", "yes", "Contract side: yes or no")
	cmd.Flags().StringVar(&action, "action", "buy", "Order action: buy or sell")
	cmd.Flags().IntVar(&count, "count", 1, "Number of contracts")
	cmd.Flags().IntVar(&yesPrice, "yes-price", 0, "Limit price in cents for YES side (1-99)")
	cmd.Flags().IntVar(&noPrice, "no-price", 0, "Limit price in cents for NO side (1-99)")
	cmd.Flags().BoolVar(&postOnly, "post-only", false, "Post-only: cancel if it would immediately fill (maker-only)")
	return cmd
}

func newOrdersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <order-id>",
		Short: "Get a specific order by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			resp, err := client.GetOrderWithResponse(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return ordersTable([]kalshi.Order{resp.JSON200.Order}, wide)
			})
		},
	}
}

func newOrdersCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <order-id>",
		Short: "Cancel an open order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			resp, err := client.CancelOrderWithResponse(context.Background(), args[0], &kalshi.CancelOrderParams{})
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return ordersTable([]kalshi.Order{resp.JSON200.Order}, wide)
			})
		},
	}
}

func ordersTable(orders []kalshi.Order, wide bool) ([]string, [][]string) {
	headers := []string{"ORDER_ID", "TICKER", "SIDE", "ACTION", "STATUS", "PRICE", "INIT", "REMAINING"}
	if wide {
		headers = append(headers, "FILLED", "CREATED_AT", "TYPE", "MAKER_FEE", "TAKER_FEE")
	}
	rows := make([][]string, 0, len(orders))
	for _, o := range orders {
		price := fmtCents(string(o.YesPriceDollars))
		if o.Side == "no" {
			price = fmtCents(string(o.NoPriceDollars))
		}
		row := []string{
			shortID(o.OrderId),
			o.Ticker,
			string(o.Side),
			string(o.Action),
			string(o.Status),
			price,
			o.InitialCountFp,
			o.RemainingCountFp,
		}
		if wide {
			row = append(row,
				o.FillCountFp,
				fmtTime(o.CreatedTime),
				string(o.Type),
				fmtCents(string(o.MakerFeesDollars)),
				fmtCents(string(o.TakerFeesDollars)),
			)
		}
		rows = append(rows, row)
	}
	return headers, rows
}
