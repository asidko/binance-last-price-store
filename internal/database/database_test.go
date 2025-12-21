package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}

func TestSymbolSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Initially empty
	settings, err := store.GetSymbolSettings()
	if err != nil {
		t.Fatalf("GetSymbolSettings failed: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("expected empty settings, got %d", len(settings))
	}
}

func TestPriceTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Create table
	if err := store.EnsurePriceTable("BTCUSDT"); err != nil {
		t.Fatalf("EnsurePriceTable failed: %v", err)
	}

	// Insert price
	if err := store.InsertPrice("BTCUSDT", 1700000000000, 42000.50); err != nil {
		t.Fatalf("InsertPrice failed: %v", err)
	}

	// Get date range
	dr, err := store.GetDateRange("BTCUSDT")
	if err != nil {
		t.Fatalf("GetDateRange failed: %v", err)
	}

	if dr.From == nil || dr.To == nil {
		t.Error("expected non-nil date range")
	}
}

func TestGetDateRange_NoTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	dr, err := store.GetDateRange("NONEXISTENT")
	if err != nil {
		t.Fatalf("GetDateRange failed: %v", err)
	}

	if dr.From != nil || dr.To != nil {
		t.Error("expected nil date range for non-existent table")
	}
}
