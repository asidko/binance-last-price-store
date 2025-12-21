package settings

import (
	"context"
	"testing"
	"time"

	"binance-tick-store/internal/database"
)

type mockStore struct {
	settings []database.SymbolSettings
}

func (m *mockStore) Close() error { return nil }
func (m *mockStore) GetSymbolSettings() ([]database.SymbolSettings, error) {
	return m.settings, nil
}
func (m *mockStore) EnsurePriceTable(symbol string) error                         { return nil }
func (m *mockStore) InsertPrice(symbol string, timestamp int64, price float64) error { return nil }
func (m *mockStore) GetDateRange(symbol string) (database.DateRange, error) {
	return database.DateRange{}, nil
}

func TestWatcher_InitialLoad(t *testing.T) {
	store := &mockStore{
		settings: []database.SymbolSettings{
			{Symbol: "BTCUSDT", Enabled: true},
			{Symbol: "ETHUSDT", Enabled: false},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	watcher := New(store, time.Hour) // Long interval, we only care about initial load
	changes := watcher.Start(ctx)

	received := make(map[string]bool)
	for change := range changes {
		received[change.Symbol] = change.Enabled
	}

	if !received["BTCUSDT"] {
		t.Error("expected BTCUSDT to be enabled")
	}
	if received["ETHUSDT"] {
		t.Error("expected ETHUSDT to be disabled")
	}
}

func TestWatcher_DetectsChanges(t *testing.T) {
	store := &mockStore{
		settings: []database.SymbolSettings{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := New(store, 10*time.Millisecond)
	changes := watcher.Start(ctx)

	// Wait for initial (empty) check
	time.Sleep(20 * time.Millisecond)

	// Add a symbol
	store.settings = []database.SymbolSettings{
		{Symbol: "BTCUSDT", Enabled: true},
	}

	// Wait for next check
	select {
	case change := <-changes:
		if change.Symbol != "BTCUSDT" || !change.Enabled {
			t.Errorf("unexpected change: %+v", change)
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("expected change event")
	}
}
