package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var validSymbol = regexp.MustCompile(`^[A-Z0-9]+$`)

// ValidateSymbol checks if symbol contains only alphanumeric characters.
func ValidateSymbol(symbol string) error {
	s := strings.ToUpper(symbol)
	if !validSymbol.MatchString(s) {
		return fmt.Errorf("invalid symbol %q: must be alphanumeric", symbol)
	}
	return nil
}

// SymbolSettings represents a symbol configuration.
type SymbolSettings struct {
	Symbol  string
	Enabled bool
}

// DateRange represents min/max timestamps for a symbol.
type DateRange struct {
	From *time.Time
	To   *time.Time
}

// Store defines database operations.
type Store interface {
	Close() error
	GetSymbolSettings() ([]SymbolSettings, error)
	EnsurePriceTable(symbol string) error
	InsertPrice(symbol string, timestamp int64, price float64) error
	GetDateRange(symbol string) (DateRange, error)
}

type store struct {
	db    *sql.DB
	mu    sync.Mutex
	stmts map[string]*sql.Stmt // prepared statements cache
}

// Open creates a new database connection with WAL mode.
func Open(path string) (Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := createSettingsTable(db); err != nil {
		db.Close()
		return nil, err
	}

	return &store{
		db:    db,
		stmts: make(map[string]*sql.Stmt),
	}, nil
}

func createSettingsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS symbol_settings (
			symbol  TEXT PRIMARY KEY,
			enabled INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		return fmt.Errorf("create symbol_settings table: %w", err)
	}
	return nil
}

func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, stmt := range s.stmts {
		stmt.Close()
	}
	return s.db.Close()
}

func (s *store) GetSymbolSettings() ([]SymbolSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT symbol, enabled FROM symbol_settings")
	if err != nil {
		return nil, fmt.Errorf("query symbol_settings: %w", err)
	}
	defer rows.Close()

	var settings []SymbolSettings
	for rows.Next() {
		var ss SymbolSettings
		var enabled int
		if err := rows.Scan(&ss.Symbol, &enabled); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		ss.Enabled = enabled == 1
		settings = append(settings, ss)
	}
	return settings, rows.Err()
}

func (s *store) EnsurePriceTable(symbol string) error {
	if err := ValidateSymbol(symbol); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := priceTableName(symbol)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER NOT NULL,
			price     REAL NOT NULL
		)
	`, table)

	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("create price table %s: %w", table, err)
	}

	idx := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s(timestamp)", table, table)
	if _, err := s.db.Exec(idx); err != nil {
		return fmt.Errorf("create index on %s: %w", table, err)
	}

	// Prepare insert statement for this symbol
	insertSQL := fmt.Sprintf("INSERT INTO %s (timestamp, price) VALUES (?, ?)", table)
	stmt, err := s.db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("prepare insert statement: %w", err)
	}
	s.stmts[symbol] = stmt

	return nil
}

func (s *store) InsertPrice(symbol string, timestamp int64, price float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stmt, ok := s.stmts[symbol]
	if !ok {
		return fmt.Errorf("no prepared statement for symbol %s", symbol)
	}

	_, err := stmt.Exec(timestamp, price)
	if err != nil {
		return fmt.Errorf("insert price: %w", err)
	}
	return nil
}

func (s *store) GetDateRange(symbol string) (DateRange, error) {
	if err := ValidateSymbol(symbol); err != nil {
		return DateRange{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	table := priceTableName(symbol)

	// Check if table exists
	var exists int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name=?
	`, table).Scan(&exists)
	if err != nil || exists == 0 {
		return DateRange{}, nil
	}

	var minTS, maxTS sql.NullInt64
	query := fmt.Sprintf("SELECT MIN(timestamp), MAX(timestamp) FROM %s", table)
	if err := s.db.QueryRow(query).Scan(&minTS, &maxTS); err != nil {
		return DateRange{}, fmt.Errorf("query date range: %w", err)
	}

	var dr DateRange
	if minTS.Valid {
		t := time.UnixMilli(minTS.Int64).UTC()
		dr.From = &t
	}
	if maxTS.Valid {
		t := time.UnixMilli(maxTS.Int64).UTC()
		dr.To = &t
	}
	return dr, nil
}

func priceTableName(symbol string) string {
	return "prices_" + strings.ToUpper(symbol)
}
