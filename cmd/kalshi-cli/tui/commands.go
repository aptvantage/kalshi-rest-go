package tui

import (
"context"
"fmt"
"time"

tea "github.com/charmbracelet/bubbletea"

"github.com/aptvantage/kalshi-rest-go/kalshi"
)

// verifyAuth calls the balance endpoint to confirm credentials are valid.
// On success it returns AuthOKMsg (with the balance so we don't need a second call).
// On HTTP 401 or any network error it returns AuthFailedMsg.
func verifyAuth(client *kalshi.ClientWithResponses) tea.Cmd {
return func() tea.Msg {
resp, err := client.GetBalanceWithResponse(context.Background(), &kalshi.GetBalanceParams{})
if err != nil {
return AuthFailedMsg{Err: fmt.Errorf("could not reach API: %w", err)}
}
if resp.StatusCode() == 401 {
return AuthFailedMsg{Err: fmt.Errorf("invalid credentials (HTTP 401) — check KALSHI_KEY_ID and KALSHI_KEY_FILE")}
}
if resp.JSON200 == nil {
return AuthFailedMsg{Err: fmt.Errorf("unexpected auth response: HTTP %d", resp.StatusCode())}
}
return AuthOKMsg{Balance: resp.JSON200.Balance}
}
}

// loadBalance refreshes the account balance (used after order placement).
func loadBalance(client *kalshi.ClientWithResponses) tea.Cmd {
return func() tea.Msg {
resp, err := client.GetBalanceWithResponse(context.Background(), &kalshi.GetBalanceParams{})
if err != nil {
return ErrMsg{Err: err}
}
if resp.StatusCode() == 401 {
return AuthFailedMsg{Err: fmt.Errorf("session expired (HTTP 401) — please restart")}
}
if resp.JSON200 == nil {
return ErrMsg{Err: fmt.Errorf("balance: HTTP %d", resp.StatusCode())}
}
return BalanceLoadedMsg{Balance: resp.JSON200.Balance}
}
}

func loadSeries(client *kalshi.ClientWithResponses) tea.Cmd {
return func() tea.Msg {
incVol := true
resp, err := client.GetSeriesListWithResponse(context.Background(), &kalshi.GetSeriesListParams{
IncludeVolume: &incVol,
})
if err != nil {
return ErrMsg{Err: err}
}
if resp.JSON200 == nil {
return ErrMsg{Err: fmt.Errorf("series list: HTTP %d", resp.StatusCode())}
}
return SeriesLoadedMsg{Series: resp.JSON200.Series}
}
}

func loadEvents(client *kalshi.ClientWithResponses, seriesTicker string) tea.Cmd {
return func() tea.Msg {
st := kalshi.SeriesTickerQuery(seriesTicker)
status := kalshi.GetEventsParamsStatus("open")
resp, err := client.GetEventsWithResponse(context.Background(), &kalshi.GetEventsParams{
SeriesTicker: &st,
Status:       &status,
})
if err != nil {
return ErrMsg{Err: err}
}
if resp.JSON200 == nil {
return ErrMsg{Err: fmt.Errorf("events list: HTTP %d", resp.StatusCode())}
}
return EventsLoadedMsg{Events: resp.JSON200.Events}
}
}

func loadMarkets(client *kalshi.ClientWithResponses, eventTicker string) tea.Cmd {
return func() tea.Msg {
status := kalshi.GetMarketsParamsStatus("open")
resp, err := client.GetMarketsWithResponse(context.Background(), &kalshi.GetMarketsParams{
EventTicker: &eventTicker,
Status:      &status,
})
if err != nil {
return ErrMsg{Err: err}
}
if resp.JSON200 == nil {
return ErrMsg{Err: fmt.Errorf("markets list: HTTP %d", resp.StatusCode())}
}
return MarketsLoadedMsg{Markets: resp.JSON200.Markets}
}
}

func loadOrderbook(client *kalshi.ClientWithResponses, ticker string) tea.Cmd {
return func() tea.Msg {
resp, err := client.GetMarketOrderbookWithResponse(context.Background(), ticker, &kalshi.GetMarketOrderbookParams{})
if err != nil {
return ErrMsg{Err: err}
}
if resp.JSON200 == nil {
return ErrMsg{Err: fmt.Errorf("orderbook: HTTP %d", resp.StatusCode())}
}
return OrderbookLoadedMsg{Ticker: ticker, Orderbook: resp.JSON200.OrderbookFp}
}
}

// createOrder submits a limit order and returns OrderCreatedMsg or ErrMsg.
func createOrder(client *kalshi.ClientWithResponses, ticker, side, action string, count, price int, postOnly bool) tea.Cmd {
return func() tea.Msg {
body := kalshi.CreateOrderRequest{
Ticker: ticker,
Side:   kalshi.CreateOrderRequestSide(side),
Action: kalshi.CreateOrderRequestAction(action),
Count:  &count,
}
if postOnly {
body.PostOnly = &postOnly
}
if side == "yes" {
body.YesPrice = &price
} else {
body.NoPrice = &price
}
resp, err := client.CreateOrderWithResponse(context.Background(), body)
if err != nil {
return ErrMsg{Err: fmt.Errorf("create order: %w", err)}
}
if resp.StatusCode() != 201 {
return ErrMsg{Err: fmt.Errorf("create order HTTP %d: %s", resp.StatusCode(), string(resp.Body))}
}
return OrderCreatedMsg{Order: resp.JSON201.Order}
}
}

// tick schedules a periodic refresh every 30 seconds.
func tick() tea.Cmd {
return tea.Tick(30*time.Second, func(_ time.Time) tea.Msg {
return TickMsg{}
})
}
