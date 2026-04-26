package main_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newMockServer starts an httptest.Server with realistic fixture responses for all
// Kalshi API endpoints used by kalshi-cli commands.
func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	handle := func(path, body string) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, body)
		})
	}

	handle("/trade-api/v2/exchange/status", `{
		"exchange_active": true,
		"trading_active": true,
		"exchange_estimated_resume_time": null
	}`)

	handle("/trade-api/v2/account/limits", `{
		"usage_tier": "standard",
		"read_limit": 10,
		"write_limit": 5
	}`)

	handle("/trade-api/v2/portfolio/balance", `{
		"balance": 1000000,
		"portfolio_value": 500000
	}`)

	handle("/trade-api/v2/portfolio/positions", `{
		"market_positions": [{
			"ticker": "KXHIGHNY-26APR25-T51",
			"position_fp": "5.00",
			"market_exposure_dollars": "0.250000",
			"realized_pnl_dollars": "0.100000",
			"fees_paid_dollars": "0.010000",
			"total_traded_dollars": "0.360000",
			"last_updated_ts": "2025-04-25T10:00:00Z"
		}],
		"event_positions": [],
		"series_positions": []
	}`)

	handle("/trade-api/v2/portfolio/fills", `{
		"fills": [{
			"ticker": "KXHIGHNY-26APR25-T51",
			"side": "yes",
			"action": "buy",
			"yes_price_dollars": "0.450000",
			"no_price_dollars": "0.550000",
			"count_fp": "5.00",
			"created_time": "2025-04-25T10:00:00Z",
			"fill_id": "fill-abc-000001",
			"order_id": "order-abc-000001",
			"fee_cost": "0.000010",
			"is_taker": true
		}],
		"cursor": ""
	}`)

	handle("/trade-api/v2/series", `{
		"series": [
			{
				"ticker": "KXHIGHNY",
				"title": "NYC High Temperature",
				"category": "weather",
				"fee_type": "quadratic",
				"fee_multiplier": 1.0,
				"frequency": "daily",
				"tags": ["temperature", "new york"],
				"settlement_sources": [],
				"additional_prohibitions": []
			},
			{
				"ticker": "KXNFLWINS",
				"title": "NFL Team Wins",
				"category": "sports",
				"fee_type": "flat",
				"fee_multiplier": 0.5,
				"frequency": "weekly",
				"tags": ["football", "nfl"],
				"settlement_sources": [],
				"additional_prohibitions": []
			}
		]
	}`)

	handle("/trade-api/v2/series/KXHIGHNY", `{
		"series": {
			"ticker": "KXHIGHNY",
			"title": "NYC High Temperature",
			"category": "weather",
			"fee_type": "quadratic",
			"fee_multiplier": 1.0,
			"frequency": "daily",
			"tags": ["temperature", "new york"],
			"settlement_sources": [],
			"additional_prohibitions": []
		}
	}`)

	handle("/trade-api/v2/search/tags_by_categories", `{
		"tags_by_categories": {
			"weather": ["temperature", "precipitation", "new york", "chicago"],
			"sports": ["football", "nfl", "basketball", "nba"]
		}
	}`)

	handle("/trade-api/v2/events", `{
		"events": [{
			"event_ticker": "KXHIGHNY-26APR25",
			"series_ticker": "KXHIGHNY",
			"title": "NYC High Temperature Apr 26, 2025",
			"sub_title": "NYC High Apr 26",
			"mutually_exclusive": false,
			"collateral_return_type": "NO_WIN",
			"strike_date": "2025-04-26T00:00:00Z"
		}],
		"cursor": ""
	}`)

	handle("/trade-api/v2/events/KXHIGHNY-26APR25", `{
		"event": {
			"event_ticker": "KXHIGHNY-26APR25",
			"series_ticker": "KXHIGHNY",
			"title": "NYC High Temperature Apr 26, 2025",
			"sub_title": "NYC High Apr 26",
			"mutually_exclusive": false,
			"collateral_return_type": "NO_WIN",
			"strike_date": "2025-04-26T00:00:00Z",
			"markets": []
		}
	}`)

	handle("/trade-api/v2/markets", `{
		"markets": [{
			"ticker": "KXHIGHNY-26APR25-T51",
			"event_ticker": "KXHIGHNY-26APR25",
			"series_ticker": "KXHIGHNY",
			"title": "NYC High Temp above 51F on Apr 26?",
			"status": "open",
			"yes_bid_dollars": "0.450000",
			"yes_ask_dollars": "0.470000",
			"no_bid_dollars": "0.530000",
			"no_ask_dollars": "0.550000",
			"volume_fp": "1000.00",
			"open_interest_fp": "200.00",
			"close_time": "2025-04-26T23:59:00Z",
			"last_price_fp": "0.46",
			"prev_yes_bid_dollars": "0.440000",
			"prev_yes_ask_dollars": "0.460000",
			"prev_no_bid_dollars": "0.540000",
			"prev_no_ask_dollars": "0.560000",
			"volume24h_fp": "150.00",
			"liquidity_fp": "50.00",
			"open_time": "2025-04-25T00:00:00Z"
		}],
		"cursor": ""
	}`)

	handle("/trade-api/v2/markets/KXHIGHNY-26APR25-T51/orderbook", `{
		"orderbook": {
			"yes": [[45, 10], [44, 5]],
			"no": [[53, 8], [52, 3]]
		}
	}`)

	// /portfolio/orders handles both GET (list) and POST (create)
	mux.HandleFunc("/trade-api/v2/portfolio/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{
				"order": {
					"order_id": "new-order-xyz-000001",
					"ticker": "KXHIGHNY-26APR25-T51",
					"side": "yes",
					"action": "buy",
					"status": "resting",
					"created_time": "2025-04-25T10:00:00Z",
					"type": "limit",
					"yes_price_dollars": "0.450000",
					"no_price_dollars": "0.550000",
					"initial_count_fp": "1.00",
					"remaining_count_fp": "1.00",
					"fill_count_fp": "0.00",
					"maker_fees_dollars": "0.000000",
					"taker_fees_dollars": "0.000000",
					"maker_fill_cost_dollars": "0.000000",
					"taker_fill_cost_dollars": "0.000000",
					"client_order_id": "",
					"user_id": ""
				}
			}`)
			return
		}
		// GET
		fmt.Fprint(w, `{
			"orders": [{
				"order_id": "order-abc-000001",
				"ticker": "KXHIGHNY-26APR25-T51",
				"side": "yes",
				"action": "buy",
				"status": "resting",
				"created_time": "2025-04-25T10:00:00Z",
				"type": "limit",
				"yes_price_dollars": "0.450000",
				"no_price_dollars": "0.550000",
				"initial_count_fp": "5.00",
				"remaining_count_fp": "5.00",
				"fill_count_fp": "0.00",
				"maker_fees_dollars": "0.000000",
				"taker_fees_dollars": "0.000000",
				"maker_fill_cost_dollars": "0.000000",
				"taker_fill_cost_dollars": "0.000000",
				"client_order_id": "",
				"user_id": ""
			}],
			"cursor": ""
		}`)
	})

	// GET and DELETE /portfolio/orders/{id}
	mux.HandleFunc("/trade-api/v2/portfolio/orders/order-abc-000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			fmt.Fprint(w, `{
				"order": {
					"order_id": "order-abc-000001",
					"ticker": "KXHIGHNY-26APR25-T51",
					"side": "yes",
					"action": "buy",
					"status": "resting",
					"created_time": "2025-04-25T10:00:00Z",
					"type": "limit",
					"yes_price_dollars": "0.450000",
					"no_price_dollars": "0.550000",
					"initial_count_fp": "5.00",
					"remaining_count_fp": "5.00",
					"fill_count_fp": "0.00",
					"maker_fees_dollars": "0.000000",
					"taker_fees_dollars": "0.000000",
					"maker_fill_cost_dollars": "0.000000",
					"taker_fill_cost_dollars": "0.000000",
					"client_order_id": "",
					"user_id": ""
				}
			}`)
		case http.MethodDelete:
			fmt.Fprint(w, `{
				"order": {
					"order_id": "order-abc-000001",
					"ticker": "KXHIGHNY-26APR25-T51",
					"side": "yes",
					"action": "buy",
					"status": "canceled",
					"created_time": "2025-04-25T10:00:00Z",
					"type": "limit",
					"yes_price_dollars": "0.450000",
					"no_price_dollars": "0.550000",
					"initial_count_fp": "5.00",
					"remaining_count_fp": "0.00",
					"fill_count_fp": "0.00",
					"maker_fees_dollars": "0.000000",
					"taker_fees_dollars": "0.000000",
					"maker_fill_cost_dollars": "0.000000",
					"taker_fill_cost_dollars": "0.000000",
					"client_order_id": "",
					"user_id": ""
				}
			}`)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newErrorServer returns an httptest.Server that responds to all requests with
// the given HTTP status and body.
func newErrorServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}
