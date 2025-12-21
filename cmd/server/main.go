package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"binance-tick-store/internal/config"
	"binance-tick-store/internal/database"
	httpHandler "binance-tick-store/internal/http"
	"binance-tick-store/internal/settings"
	"binance-tick-store/internal/websocket"
)

func main() {
	slog.Info("starting binance tick store")

	cfg := config.Load()
	slog.Info("config loaded", "db_path", cfg.DBPath, "http_port", cfg.HTTPPort)

	store, err := database.Open(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := newApp(store)

	// Start settings watcher
	watcher := settings.New(store, 60*time.Second)
	changes := watcher.Start(ctx)

	// Process settings changes
	go app.handleChanges(ctx, changes)

	// Start HTTP server
	handler := httpHandler.NewHandler(store, app)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: handler,
	}

	go func() {
		slog.Info("http server starting", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	server.Shutdown(shutdownCtx)
	app.stopAll()

	slog.Info("shutdown complete")
}

// app manages WebSocket clients for symbols.
type app struct {
	store   database.Store
	dialer  websocket.Dialer
	clients map[string]context.CancelFunc
	mu      sync.RWMutex
}

func newApp(store database.Store) *app {
	return &app{
		store:   store,
		dialer:  &websocket.DefaultDialer{},
		clients: make(map[string]context.CancelFunc),
	}
}

// GetActiveSymbols returns currently connected symbols.
func (a *app) GetActiveSymbols() map[string]bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	active := make(map[string]bool)
	for symbol := range a.clients {
		active[symbol] = true
	}
	return active
}

func (a *app) handleChanges(ctx context.Context, changes <-chan settings.SymbolChange) {
	for {
		select {
		case <-ctx.Done():
			return
		case change, ok := <-changes:
			if !ok {
				return
			}
			if change.Enabled {
				a.startClient(ctx, change.Symbol)
			} else {
				a.stopClient(change.Symbol)
			}
		}
	}
}

func (a *app) startClient(ctx context.Context, symbol string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.clients[symbol]; exists {
		return
	}

	if err := a.store.EnsurePriceTable(symbol); err != nil {
		slog.Error("failed to create price table", "symbol", symbol, "error", err)
		return
	}

	clientCtx, cancel := context.WithCancel(ctx)
	a.clients[symbol] = cancel

	handler := func(tick websocket.Tick) {
		if err := a.store.InsertPrice(tick.Symbol, tick.Timestamp, tick.Price); err != nil {
			slog.Error("failed to insert price", "symbol", tick.Symbol, "error", err)
		}
	}

	client := websocket.NewClient(symbol, a.dialer, handler)
	go client.Run(clientCtx)

	slog.Info("client started", "symbol", symbol)
}

func (a *app) stopClient(symbol string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if cancel, exists := a.clients[symbol]; exists {
		cancel()
		delete(a.clients, symbol)
		slog.Info("client stopped", "symbol", symbol)
	}
}

func (a *app) stopAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for symbol, cancel := range a.clients {
		cancel()
		slog.Info("client stopped", "symbol", symbol)
	}
	a.clients = make(map[string]context.CancelFunc)
}
