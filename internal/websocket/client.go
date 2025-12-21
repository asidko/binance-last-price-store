package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	binanceWSURL = "wss://fstream.binance.com/ws/%s@aggTrade"
	maxBackoff   = 30 * time.Second
)

// Tick represents a price update.
type Tick struct {
	Symbol    string
	Timestamp int64
	Price     float64
}

// TickHandler processes incoming ticks.
type TickHandler func(Tick)

// Dialer abstracts WebSocket connection creation (for testing).
type Dialer interface {
	Dial(url string) (Conn, error)
}

// Conn abstracts a WebSocket connection (for testing).
type Conn interface {
	ReadMessage() (int, []byte, error)
	Close() error
}

// Client manages a WebSocket connection for a symbol.
type Client interface {
	Run(ctx context.Context)
}

type client struct {
	symbol  string
	dialer  Dialer
	handler TickHandler
}

// NewClient creates a new WebSocket client for a symbol.
func NewClient(symbol string, dialer Dialer, handler TickHandler) Client {
	return &client{
		symbol:  symbol,
		dialer:  dialer,
		handler: handler,
	}
}

func (c *client) Run(ctx context.Context) {
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := c.connect(ctx)
		if err != nil {
			slog.Error("websocket error", "symbol", c.symbol, "error", err)
		} else {
			backoff = time.Second // Reset backoff on clean disconnect
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

func (c *client) connect(ctx context.Context) error {
	url := fmt.Sprintf(binanceWSURL, strings.ToLower(c.symbol))

	conn, err := c.dialer.Dial(url)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// Close connection when context is cancelled
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	slog.Info("websocket connected", "symbol", c.symbol)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil // Clean shutdown
			}
			return fmt.Errorf("read: %w", err)
		}

		tick, err := parseAggTrade(c.symbol, msg)
		if err != nil {
			slog.Warn("parse error", "symbol", c.symbol, "error", err)
			continue
		}

		c.handler(tick)
	}
}

// aggTrade represents Binance aggTrade message.
type aggTrade struct {
	TradeTime int64  `json:"T"`
	Price     string `json:"p"`
}

func parseAggTrade(symbol string, data []byte) (Tick, error) {
	var at aggTrade
	if err := json.Unmarshal(data, &at); err != nil {
		return Tick{}, err
	}

	var price float64
	if _, err := fmt.Sscanf(at.Price, "%f", &price); err != nil {
		return Tick{}, fmt.Errorf("parse price: %w", err)
	}

	return Tick{
		Symbol:    symbol,
		Timestamp: at.TradeTime,
		Price:     price,
	}, nil
}

// DefaultDialer implements Dialer using gorilla/websocket.
type DefaultDialer struct{}

func (d *DefaultDialer) Dial(url string) (Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &connWrapper{conn}, nil
}

type connWrapper struct {
	*websocket.Conn
}

func (c *connWrapper) ReadMessage() (int, []byte, error) {
	return c.Conn.ReadMessage()
}

func (c *connWrapper) Close() error {
	return c.Conn.Close()
}
