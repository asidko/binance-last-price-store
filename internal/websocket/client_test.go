package websocket

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockConn struct {
	messages [][]byte
	index    int
	closed   bool
}

func (m *mockConn) ReadMessage() (int, []byte, error) {
	if m.closed {
		return 0, nil, errors.New("connection closed")
	}
	if m.index >= len(m.messages) {
		// Block until closed
		time.Sleep(time.Hour)
		return 0, nil, errors.New("connection closed")
	}
	msg := m.messages[m.index]
	m.index++
	return 1, msg, nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

type mockDialer struct {
	conn *mockConn
	err  error
}

func (m *mockDialer) Dial(url string) (Conn, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.conn, nil
}

func TestParseAggTrade(t *testing.T) {
	data := []byte(`{"T":1700000000000,"p":"42000.50"}`)

	tick, err := parseAggTrade("BTCUSDT", data)
	if err != nil {
		t.Fatalf("parseAggTrade failed: %v", err)
	}

	if tick.Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", tick.Symbol)
	}
	if tick.Timestamp != 1700000000000 {
		t.Errorf("expected 1700000000000, got %d", tick.Timestamp)
	}
	if tick.Price != 42000.50 {
		t.Errorf("expected 42000.50, got %f", tick.Price)
	}
}

func TestClient_ReceivesTicks(t *testing.T) {
	conn := &mockConn{
		messages: [][]byte{
			[]byte(`{"T":1700000000000,"p":"42000.00"}`),
			[]byte(`{"T":1700000001000,"p":"42001.00"}`),
		},
	}
	dialer := &mockDialer{conn: conn}

	var ticks []Tick
	handler := func(tick Tick) {
		ticks = append(ticks, tick)
	}

	client := NewClient("BTCUSDT", dialer, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go client.Run(ctx)

	time.Sleep(30 * time.Millisecond)

	if len(ticks) != 2 {
		t.Errorf("expected 2 ticks, got %d", len(ticks))
	}
}

func TestClient_ReconnectsOnError(t *testing.T) {
	callCount := 0
	dialer := &mockDialer{err: errors.New("connection failed")}

	client := NewClient("BTCUSDT", dialer, func(tick Tick) {})

	// Override dialer to count calls
	originalDial := dialer.Dial
	dialer.Dial = func(url string) (Conn, error) {
		callCount++
		return originalDial(url)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client.Run(ctx)

	if callCount < 2 {
		t.Errorf("expected multiple reconnection attempts, got %d", callCount)
	}
}
