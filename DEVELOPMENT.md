# Developer Guide

## Prerequisites

- **Go 1.26.2+** — `go version` to check
- **oapi-codegen v2** — for SDK regeneration (optional unless updating the spec)

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

## Project Structure

```
kalshi-rest-go/
├── auth/                   # RSA-PSS signing middleware
│   └── auth.go             #   Signer, Transport, NewClient
├── kalshi/
│   └── kalshi.gen.go       # Generated API client — DO NOT EDIT by hand
├── cmd/kalshi-cli/         # CLI entry point
│   ├── main.go             #   Root command, client factory, env routing
│   ├── output.go           #   render(), printTable(), printJSON(), printYAML(), format helpers
│   ├── exchange.go         #   exchange subcommands
│   ├── series.go           #   series subcommands (list, get, categories)
│   ├── events.go           #   events subcommands (list, get)
│   ├── markets.go          #   markets subcommands (list, get, orderbook)
│   ├── orders.go           #   orders subcommands
│   ├── portfolio.go        #   portfolio subcommands
│   └── tui/                #   Interactive terminal UI (Bubble Tea)
│       ├── model.go        #     Model struct, New(), Init(), table builders
│       ├── update.go       #     Update() dispatch, navigation, orderbook renderer
│       ├── view.go         #     View() layout: header / content / status / help
│       ├── commands.go     #     Async tea.Cmd wrappers for each API call
│       ├── messages.go     #     Typed Msg structs (SeriesLoadedMsg, etc.)
│       ├── keys.go         #     Key binding definitions
│       ├── styles.go       #     Lip Gloss style palette + tableStyles()
│       └── format.go       #     Display helpers (fmtCents, fmtSpread, …)
├── kalshi-cli              # Pre-built binary (rebuild with: go build -o kalshi-cli ./cmd/kalshi-cli)
├── kalshi.yaml             # Kalshi OpenAPI spec (patched — see below)
└── oapi-codegen.yaml       # Codegen config
```

## Building

```bash
# Build all packages (validates everything compiles)
go build ./...

# Build the CLI binary to the repo root (run as ./kalshi-cli)
go build -o kalshi-cli ./cmd/kalshi-cli

# Install the CLI globally to $GOPATH/bin (run as kalshi-cli from anywhere)
go install ./cmd/kalshi-cli/...

# Verify
./kalshi-cli --help
```

> **Tip:** After any code change, re-run `go build -o kalshi-cli ./cmd/kalshi-cli` to refresh the local binary before testing. The binary in the repo root is not updated automatically.

## Running the CLI

### Credentials

The CLI reads credentials from environment variables. Both forms are supported:

```bash
# Option A: PEM string (useful when key is stored in a secrets manager)
export KALSHI_KEY_ID=your-api-key-id
export KALSHI_PRIVATE_KEY="$(cat ~/.secrets/kalshi-key.pem)"

# Option B: file path
export KALSHI_KEY_ID=your-api-key-id
export KALSHI_KEY_FILE=~/.secrets/kalshi-key.pem
```

### Environments

Pass `--env demo` to use the sandbox API instead of production:

```bash
# Production (default)
kalshi-cli portfolio balance

# Demo / sandbox
kalshi-cli --env demo portfolio balance
```

| Environment | Base URL |
|---|---|
| `prod` | `https://api.elections.kalshi.com/trade-api/v2` |
| `demo` | `https://demo-api.kalshi.co/trade-api/v2` |

Demo credentials are separate from production — you need a distinct API key created on the demo platform.

## Testing

There are no automated unit tests yet. Until a mock server or recorded fixtures are added, testing is done manually against the demo environment.

### Manual smoke test

```bash
# Source demo credentials
source ~/.secrets/kalshi-demo-test-key

# 1. Exchange status (no auth required)
kalshi-cli --env demo exchange status

# 2. Authenticated balance
kalshi-cli --env demo portfolio balance

# 3. API rate limits
kalshi-cli --env demo exchange limits

# 4. Browse categories, then narrow to a series
kalshi-cli series categories
kalshi-cli series list --tags "Daily temperature" --include-volume -o wide

# 5. Find open events and markets for a known series
kalshi-cli events list --series-ticker KXHIGHNY --status open
kalshi-cli events get KXHIGHNY-26APR25     # replace with a date returned above
kalshi-cli markets list --status open --series-ticker KXHIGHNY -o wide

# 6. Orderbook for a specific market
kalshi-cli markets orderbook KXHIGHNY-26APR25-T51   # replace ticker as needed

# 7. Place a resting limit order (1¢ YES buy — won't fill, safe to test)
kalshi-cli --env demo orders create \
  --ticker <ticker-from-step-5> \
  --side yes --action buy --count 1 --yes-price 1 --post-only

# 8. Cancel it
kalshi-cli --env demo orders cancel <order-id-from-step-7>
```

Expected results:
- `exchange status` → `{"exchange_active": true, "trading_active": true}`
- `portfolio balance` → `{"balance": 50000, ...}` ($500 demo funds)
- `series categories` → table of 14 categories with tags
- `orders create` → order with `"status": "resting"`
- `orders cancel` → same order with `"status": "canceled"`

## Regenerating the SDK

The `kalshi/kalshi.gen.go` file is generated from `kalshi.yaml` using `oapi-codegen`. Commit the generated file — consumers need it to be present for `go get` to work without running codegen themselves.

```bash
# 1. Download the latest spec
curl -sSL https://docs.kalshi.com/openapi.yaml -o kalshi.yaml

# 2. Remove the x-go-type-skip-optional-pointer extension
#    This extension causes oapi-codegen to emit invalid nil comparisons
#    for optional bool/string params. Stripping it restores pointer types.
sed -i '' '/x-go-type-skip-optional-pointer/d' kalshi.yaml

# 3. Regenerate
oapi-codegen -config oapi-codegen.yaml kalshi.yaml

# 4. Verify it compiles
go build ./...
```

### Codegen config (`oapi-codegen.yaml`)

| Setting | Value | Why |
|---|---|---|
| `generate.client` | `true` | HTTP client with `WithResponse` methods |
| `generate.models` | `true` | Request/response structs |
| `response-type-suffix` | `HTTPResponse` | Avoids name collision with Kalshi's own `*Response` schema types |

## Adding a New CLI Command

1. Create or edit the relevant `cmd/kalshi-cli/<group>.go` file
2. Define a `new<Group><Action>Cmd()` function returning `*cobra.Command`
3. Call `newAuthClient()` for authenticated endpoints or `newUnauthClient()` for public endpoints
4. Register it in the parent command's `AddCommand(...)` call
5. Use `render(resp.JSON200, tableFunc)` for output — it dispatches to table/wide/json/yaml based on `-o`

Output helpers in `output.go`:
- `render(data, tableFunc)` — dispatch to json/yaml/table based on `-o` flag
- `printTable(headers, rows)` — tabwriter-aligned table
- `fmtCents(fixedPoint)` — converts `"0.4500"` → `"45¢"`
- `fmtTimeVal(t)` / `fmtTime(*t)` — formats timestamps as `MM/DD HH:MMZ`
- `truncate(s, max)` — truncates strings for table columns

## Known Issues / Gotchas

- **GPG commit signing**: `git commit` in non-interactive shells (e.g., Copilot tools) times out on GPG pinentry. Workaround: `git -c commit.gpgsign=false commit`.
- **Demo credentials**: The demo API (`demo-api.kalshi.co`) is live but requires a separate account and API key — production credentials return HTTP 401.
- **Kalshi spec bug**: The upstream spec uses `x-go-type-skip-optional-pointer: true` on optional bool/string query params but the generated code still emits nil comparisons, causing compile errors. The `kalshi.yaml` in this repo already has these stripped — re-apply the `sed` command after any spec update.
- **MVE markets dominate default list**: `markets list` without filters returns multivariate/combo markets by default. Use `--mve-filter exclude` or `--series-ticker` (e.g., `KXHIGHNY`) to get standard single-leg markets.
