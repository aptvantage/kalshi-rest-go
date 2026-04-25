package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
)

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

func loadBalance(client *kalshi.ClientWithResponses) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetBalanceWithResponse(context.Background(), &kalshi.GetBalanceParams{})
		if err != nil {
			return ErrMsg{Err: err}
		}
		if resp.JSON200 == nil {
			return ErrMsg{Err: fmt.Errorf("balance: HTTP %d", resp.StatusCode())}
		}
		return BalanceLoadedMsg{Balance: resp.JSON200.Balance}
	}
}

// tick schedules a periodic refresh every 30 seconds.
func tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(_ time.Time) tea.Msg {
		return TickMsg{}
	})
}
