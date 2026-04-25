# kalshi-rest-go

> **Developer docs:** See [DEVELOPMENT.md](./DEVELOPMENT.md) for build instructions, smoke tests, and SDK regeneration steps.

Go SDK and CLI for the [Kalshi Trade REST API](https://docs.kalshi.com), generated from the official OpenAPI spec.

## Packages

| Package | Description |
|---|---|
| `github.com/aptvantage/kalshi-rest-go/kalshi` | Generated API client (models + HTTP client) |
| `github.com/aptvantage/kalshi-rest-go/auth` | RSA-PSS request signing middleware |
| `github.com/aptvantage/kalshi-rest-go/cmd/kalshi-cli` | Command-line interface |

## Authentication

Kalshi uses RSA-PSS signing. Each request is signed with your RSA private key:

```
KALSHI-ACCESS-KEY: <key-id>
KALSHI-ACCESS-TIMESTAMP: <unix-ms>
KALSHI-ACCESS-SIGNATURE: base64(RSA-PSS-SHA256(timestamp + METHOD + path))
```

Set credentials via environment variables:

```bash
export KALSHI_KEY_ID=your-key-id-here
export KALSHI_PRIVATE_KEY="$(cat ~/.secrets/kalshi-key.pem)"
# OR
export KALSHI_KEY_FILE=~/.secrets/kalshi-key.pem
```

## CLI Usage

### Install

```bash
go install github.com/aptvantage/kalshi-rest-go/cmd/kalshi-cli@latest
```

### Commands

```
kalshi-cli [--env prod|demo] [-o table|wide|json|yaml] <command>

exchange:
  exchange status          Exchange up/down status (no auth required)
  exchange limits          Your API rate limit tier

series:                    Browse the contract template hierarchy
  series categories        List all categories and their tags
  series list              List series (--category, --tags, --include-volume)
  series get <ticker>      Get a single series with volume

events:                    Browse dated instances of a series
  events list              List events (--series-ticker, --status, --min-close,
                             --with-markets, --all, --cursor, --limit)
  events get <ticker>      Get a single event with its markets inline

markets:                   Browse individual binary contracts
  markets list             List markets (--series-ticker, --status, --min-close, --max-close,
                             --search, --mve-filter, --limit, --cursor, --all)
  markets get <ticker>     Get a single market
  markets orderbook <ticker>  Get the order book

portfolio:
  portfolio balance        Current balance
  portfolio positions      Open positions
  portfolio fills          Trade fills

orders:
  orders list              List orders (--ticker, --status, --limit)
  orders create            Place a limit order
  orders get <id>          Get order by ID
  orders cancel <id>       Cancel an open order
```

The three-level hierarchy:
```
Series (KXHIGHNY)  →  Event (KXHIGHNY-26APR25)  →  Market (KXHIGHNY-26APR25-T51)
```

### Browse & discover markets

```bash
# 1. Find categories and their tags
kalshi-cli series categories -o wide

# 2. Find series within a category, ranked by volume
kalshi-cli series list --tags "Daily temperature" --include-volume -o wide

# 3. See open events for a series
kalshi-cli events list --series-ticker KXHIGHNY --status open

# 4. See all markets inside an event
kalshi-cli events get KXHIGHNY-26APR25

# 5. Scan open markets across a series for spread/volume
kalshi-cli markets list --status open --series-ticker KXHIGHNY -o wide

# 6. Drill into a specific market's orderbook
kalshi-cli markets orderbook KXHIGHNY-26APR25-T51
```

### Example: Place and cancel a limit order

```bash
# Find an active market
kalshi-cli markets list --series-ticker KXHIGHNY --status open

# Place a resting limit order (YES at 1¢ — far from market, won't fill)
kalshi-cli orders create \
  --ticker KXHIGHNY-26APR25-T51 \
  --side yes \
  --action buy \
  --count 1 \
  --yes-price 1 \
  --post-only

# Cancel it
kalshi-cli orders cancel <order-id>
```

## Interactive TUI

Running `kalshi-cli` with no subcommand launches a full-screen interactive terminal UI for browsing markets, scanning order books, and placing orders.

```bash
kalshi-cli              # launch TUI (production)
kalshi-cli --env demo   # launch against the demo/sandbox environment
```

> **Debug mode:** add `--debug /tmp/kalshi.log` to write verbose API and navigation logs to a file without corrupting the terminal.

### Navigation hierarchy

```
Categories  →  Series (filtered)  →  Events  →  Markets  →  Orderbook
```

Each screen narrows the data; `esc` / `backspace` steps back. The current path is shown in the breadcrumb at the top of the screen.

### Key bindings

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `⏎ Enter` | Go deeper (open selected row) |
| `esc` / `⌫` | Go back one level |
| `/` | Open filter bar |
| `o` | New order (Markets screen only) |
| `q` / `ctrl+c` | Quit |

**In the filter bar:**

| Key | Action |
|---|---|
| _type_ | Narrow results live |
| `⏎ Enter` | Apply filter and navigate (on Categories) / close bar |
| `esc` | Clear filter and close bar |

**In the order entry form:**

| Key | Action |
|---|---|
| `tab` / `↓` | Next field |
| `shift+tab` / `↑` | Previous field |
| `space` | Toggle side / action / post-only |
| `ctrl+s` / `⏎` on Submit | Submit order |
| `esc` | Cancel / go back |

### Filter syntax

The filter bar (press `/`) is available on every screen. Filters apply live as you type.

| Syntax | Meaning |
|---|---|
| `hockey` | Substring match across all fields |
| `hockey*` | Prefix match |
| `*` | Match everything (same as no filter) |
| `a \| b` | OR — match either term |
| `category:sports` | Match only the category name field |
| `tag:nhl` | Match only tag fields |
| `category:sp*` | Field qualifier + wildcard combined |

**Example:** `/tag:temperature | tag:weather` shows all categories/series tagged with temperature or weather topics.

### Typical workflow

```
1. Launch: kalshi-cli
2. (Optional) Press / and type a tag or category filter
   e.g.  "hockey | basketball"  or  "category:sports"
3. Press Enter → lands on Series screen filtered to matching series
4. Navigate with ↑↓, Enter to open a series → Events screen
5. Enter on an event → Markets screen (shows bid/ask/spread/volume)
6. Enter on a market → Orderbook (scrollable with ↑↓)
7. Press esc to back out, or o on a market row to open order entry
```

### Order entry

Press `o` on any row in the Markets screen to open the order form. Fields:

| Field | Description |
|---|---|
| side | `yes` or `no` |
| action | `buy` or `sell` |
| count | Number of contracts |
| price | Limit price in cents (1–99) |
| post-only | Only rest; reject if it would immediately fill |

The header always shows your current balance, which refreshes after each order.

## SDK Usage

```go
import (
    "github.com/aptvantage/kalshi-rest-go/auth"
    "github.com/aptvantage/kalshi-rest-go/kalshi"
)

signer, _ := auth.NewSignerFromPEM(keyID, pemBytes)
client, _ := kalshi.NewClientWithResponses(
    "https://api.elections.kalshi.com/trade-api/v2",
    kalshi.WithHTTPClient(auth.NewClient(signer)),
)

// Get balance
resp, _ := client.GetBalanceWithResponse(ctx, &kalshi.GetBalanceParams{})
fmt.Println(resp.JSON200.Balance)

// Place order
count := 1
yesPrice := 45
resp, _ := client.CreateOrderWithResponse(ctx, kalshi.CreateOrderRequest{
    Ticker:   "KXBTCD-26APR2117-T79499.99",
    Side:     kalshi.CreateOrderRequestSideYes,
    Action:   kalshi.CreateOrderRequestActionBuy,
    Count:    &count,
    YesPrice: &yesPrice,
})
```

## Regenerating the SDK

```bash
# Update the spec
curl -sSL https://docs.kalshi.com/openapi.yaml -o kalshi.yaml

# Remove the x-go-type-skip-optional-pointer extension (causes nil comparison errors)
sed -i '' '/x-go-type-skip-optional-pointer/d' kalshi.yaml

# Regenerate
oapi-codegen -config oapi-codegen.yaml kalshi.yaml
```

## API Rate Limits

| Tier | Reads/sec | Writes/sec | How to get |
|---|---|---|---|
| Basic | 20 | 10 | Default after signup |
| Advanced | 30 | 30 | [Apply via Typeform](https://kalshi.typeform.com/advanced-api) |
| Premier | 100 | 100 | 3.75% of monthly exchange volume |
| Prime | 400 | 400 | 7.5% of monthly exchange volume |

Rate-limited endpoints: `CreateOrder`, `CancelOrder`, `AmendOrder`, `DecreaseOrder`, `BatchCreateOrders`, `BatchCancelOrders`.
