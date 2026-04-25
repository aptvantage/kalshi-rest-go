package tui

import "github.com/aptvantage/kalshi-rest-go/kalshi"

// Message types emitted by async commands and delivered to Update().

type (
	SeriesLoadedMsg  struct{ Series []kalshi.Series }
	EventsLoadedMsg  struct{ Events []kalshi.EventData }
	MarketsLoadedMsg struct{ Markets []kalshi.Market }
	BalanceLoadedMsg struct{ Balance int64 }
	OrderbookLoadedMsg struct {
		Ticker    string
		Orderbook kalshi.OrderbookCountFp
	}
	ErrMsg  struct{ Err error }
	TickMsg struct{} // periodic refresh tick
)
