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
	closeCh  chan struct{}
}

func newMockConn(messages [][]byte) *mockConn {
	return &mockConn{
		messages: messages,
		closeCh:  make(chan struct{}),
	}
}

func (m *mockConn) ReadMessage() (int, []byte, error) {
	if m.index >= len(m.messages) {
		<-m.closeCh
		return 0, nil, errors.New("connection closed")
	}
	msg := m.messages[m.index]
	m.index++
	return 1, msg, nil
}

func (m *mockConn) Close() error {
	select {
	case <-m.closeCh:
	default:
		close(m.closeCh)
	}
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
	conn := newMockConn([][]byte{
		[]byte(`{"T":1700000000000,"p":"42000.00"}`),
		[]byte(`{"T":1700000001000,"p":"42001.00"}`),
	})
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

type countingDialer struct {
	err       error
	callCount int
}

func (d *countingDialer) Dial(url string) (Conn, error) {
	d.callCount++
	return nil, d.err
}

func TestClient_ReconnectsOnError(t *testing.T) {
	dialer := &countingDialer{err: errors.New("connection failed")}

	client := NewClient("BTCUSDT", dialer, func(tick Tick) {})

	// Backoff starts at 1s, so we need >1s for 2 attempts
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	client.Run(ctx)

	if dialer.callCount < 2 {
		t.Errorf("expected multiple reconnection attempts, got %d", dialer.callCount)
	}
}
