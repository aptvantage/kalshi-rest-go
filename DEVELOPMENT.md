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
│   ├── exchange.go         #   exchange subcommands
│   ├── markets.go          #   markets subcommands
│   ├── orders.go           #   orders subcommands
│   └── portfolio.go        #   portfolio subcommands
├── kalshi.yaml             # Kalshi OpenAPI spec (patched — see below)
└── oapi-codegen.yaml       # Codegen config
```

## Building

```bash
# Build all packages
go build ./...

# Install the CLI to $GOPATH/bin
go install ./cmd/kalshi-cli/...

# Verify
kalshi-cli --help
```

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

# 4. Find an active market
kalshi-cli --env demo markets list --series-ticker KXBTCD --limit 20

# 5. Place a resting limit order (1¢ YES buy — won't fill, safe to test)
kalshi-cli --env demo orders create \
  --ticker <ticker-from-step-4> \
  --side yes --action buy --count 1 --yes-price 1 --post-only

# 6. Cancel it
kalshi-cli --env demo orders cancel <order-id-from-step-5>
```

Expected results:
- `exchange status` → `{"exchange_active": true, "trading_active": true}`
- `portfolio balance` → `{"balance": 50000, ...}` ($500 demo funds)
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
3. Call `newClient()` for authenticated endpoints or `newUnauthClient()` for public endpoints
4. Register it in the parent command's `AddCommand(...)` call
5. Use `prettyPrint(resp.JSON200)` for JSON output consistency

## Known Issues / Gotchas

- **GPG commit signing**: `git commit` in non-interactive shells (e.g., Copilot tools) times out on GPG pinentry. Workaround: `git -c commit.gpgsign=false commit`.
- **Demo credentials**: The demo API (`demo-api.kalshi.co`) is live but requires a separate account and API key — production credentials return HTTP 401.
- **Kalshi spec bug**: The upstream spec uses `x-go-type-skip-optional-pointer: true` on optional bool/string query params but the generated code still emits nil comparisons, causing compile errors. The `kalshi.yaml` in this repo already has these stripped — re-apply the `sed` command after any spec update.
- **MVE markets dominate default list**: `markets list` without filters returns multivariate/combo markets by default. Use `--series-ticker` (e.g., `KXBTCD`) to get standard single-leg markets.
