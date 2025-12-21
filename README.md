# Binance Symbol Tick Store

Captures real-time price ticks from Binance USD-M Futures and stores them in SQLite.

## Quick Start

```bash
make run      # Build and start
make status   # Check app status
```

## Configuration

Add symbols via SQLite:
```sql
INSERT INTO symbol_settings (symbol) VALUES ('BTCUSDT');
```

See `spec.txt` for full documentation.
