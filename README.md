# Binance Last Price Store

Captures real-time price ticks from Binance USD-M Futures WebSocket streams and stores them in SQLite. Solves the problem that Binance provides kline history but not individual tick data.

## Quick Start

```bash
make run                # Build and start with Docker
make enable BTCUSDT     # Add symbol to track
make status             # Check app status
```

Wait ~60 seconds for the watcher to pick up the new symbol. Check status again:

```
Binance Last Price Store

Status:     running
Uptime:     2m

BTCUSDT     on      1985      2025-12-21 19:51:58  ->  2025-12-21 19:53:13
ETHUSDT     on      802       2025-12-21 19:51:58  ->  2025-12-21 19:53:13
```

## Commands

```bash
make run              # Start with Docker
make stop             # Stop container
make status           # Show app status
make logs             # Tail container logs
make enable BTCUSDT   # Start tracking symbol
make disable BTCUSDT  # Stop tracking symbol
make build            # Build binary locally
make test             # Run tests
```

## Running with Docker

Everything already included in a container. No other dependencies or configuration needed.

1. Start the container:
   ```bash
   docker compose up -d
   ```

2. Enable symbols to track:
   ```bash
   docker compose exec binance-last-price-store make enable BTCUSDT
   docker compose exec binance-last-price-store make enable ETHUSDT
   ```

3. Check status (wait ~60s for watcher to pick up changes):
   ```bash
   docker compose exec binance-last-price-store make status
   ```

4. (Optional) Query database if needed:
   ```bash
   docker compose exec binance-last-price-store sqlite3 /app/.data/ticks.db \
     "SELECT * FROM prices_BTCUSDT ORDER BY id DESC LIMIT 5;"
   ```

## How It Works

1. **Settings watcher** polls the database every 60 seconds for symbol configuration changes
2. For each enabled symbol, a **WebSocket client** connects to `wss://fstream.binance.com/ws/<symbol>@aggTrade`
3. Incoming ticks are parsed and stored in per-symbol SQLite tables (`prices_BTCUSDT`, `prices_ETHUSDT`, etc.)
4. On connection failure, clients **auto-reconnect** with exponential backoff (1s → 30s max)

## Accessing Data

```bash
# Query ticks directly
sqlite3 .data/ticks.db "SELECT * FROM prices_BTCUSDT ORDER BY id DESC LIMIT 5;"

# Output: id|timestamp|price
# 1986|1766346793828|88474.8
# 1985|1766346793639|88474.9
# 1984|1766346792903|88474.9
# ...
```

Timestamps are Unix milliseconds (Binance trade time).

## Development

Prerequisites: Go 1.23+, SQLite3

```bash
make build
./bin/server

# In another terminal
make enable BTCUSDT
make status
```

### Environment Variables

- `DB_PATH` - SQLite database path (default: `./.data/ticks.db`)
- `HTTP_PORT` - HTTP server port (default: `8080`)
- `LOG_LEVEL` - DEBUG, INFO, WARN, ERROR (default: `INFO`)
