package settings

import (
	"context"
	"log/slog"
	"time"

	"binance-tick-store/internal/database"
)

// SymbolChange represents a change in symbol state.
type SymbolChange struct {
	Symbol  string
	Enabled bool
}

// Watcher monitors symbol_settings for changes.
type Watcher interface {
	Start(ctx context.Context) <-chan SymbolChange
}

type watcher struct {
	store    database.Store
	interval time.Duration
	known    map[string]bool
}

// New creates a new settings watcher.
func New(store database.Store, interval time.Duration) Watcher {
	return &watcher{
		store:    store,
		interval: interval,
		known:    make(map[string]bool),
	}
}

func (w *watcher) Start(ctx context.Context) <-chan SymbolChange {
	ch := make(chan SymbolChange, 16)

	go func() {
		defer close(ch)

		// Initial load
		w.check(ch)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.check(ch)
			}
		}
	}()

	return ch
}

func (w *watcher) check(ch chan<- SymbolChange) {
	settings, err := w.store.GetSymbolSettings()
	if err != nil {
		slog.Error("failed to get symbol settings", "error", err)
		return
	}

	current := make(map[string]bool)
	for _, s := range settings {
		current[s.Symbol] = s.Enabled
	}

	// Detect new or changed symbols
	for symbol, enabled := range current {
		prev, exists := w.known[symbol]
		if !exists || prev != enabled {
			slog.Info("symbol settings changed", "symbol", symbol, "enabled", enabled)
			ch <- SymbolChange{Symbol: symbol, Enabled: enabled}
		}
	}

	// Detect removed symbols
	for symbol := range w.known {
		if _, exists := current[symbol]; !exists {
			slog.Info("symbol removed", "symbol", symbol)
			ch <- SymbolChange{Symbol: symbol, Enabled: false}
		}
	}

	w.known = current
}
