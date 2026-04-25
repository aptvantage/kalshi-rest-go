package tui

import "github.com/aptvantage/kalshi-rest-go/kalshi"

// Message types emitted by async commands and delivered to Update().

type (
// AuthOKMsg is returned by verifyAuth when credentials are valid.
AuthOKMsg struct{ Balance int64 }
// AuthFailedMsg is returned when authentication fails or a 401 is received mid-session.
AuthFailedMsg struct{ Err error }

SeriesLoadedMsg  struct{ Series []kalshi.Series }
EventsLoadedMsg  struct{ Events []kalshi.EventData }
MarketsLoadedMsg struct{ Markets []kalshi.Market }
BalanceLoadedMsg struct{ Balance int64 }
OrderbookLoadedMsg struct {
Ticker    string
Orderbook kalshi.OrderbookCountFp
}
OrderCreatedMsg struct{ Order kalshi.Order }
ErrMsg          struct{ Err error }
TickMsg         struct{} // periodic refresh tick
)
