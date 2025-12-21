# Project Guidelines

- Git: conventional commits, compact messages (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`)
- Keep CLAUDE.md minimal

# Overview

Binance Last Price Store - a Go service that captures real-time price ticks from Binance USD-M Futures via WebSocket (`@aggTrade` streams) and stores them in SQLite with WAL mode. Symbols are configured dynamically in the database and watched every 60 seconds for changes, allowing zero-downtime reconfiguration. Each symbol gets its own WebSocket connection with exponential backoff reconnection (1s→30s) and a dedicated price table with timestamp indexing. The app exposes an HTTP `/status` endpoint for health checks, handles graceful shutdown, and runs in Docker with persistent volume storage. Pure Go with no CGO dependencies.
