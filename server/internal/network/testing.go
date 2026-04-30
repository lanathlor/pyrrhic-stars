package network

import (
	"context"
	"sync"
)

// TestSpy records messages sent through a test client.
type TestSpy struct {
	mu   sync.Mutex
	msgs [][]byte
}

// Messages returns a copy of all recorded messages.
func (s *TestSpy) Messages() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.msgs))
	copy(out, s.msgs)
	return out
}

// Reset clears all recorded messages.
func (s *TestSpy) Reset() {
	s.mu.Lock()
	s.msgs = s.msgs[:0]
	s.mu.Unlock()
}

// NewTestClient creates a Client without a WebSocket connection.
// All sent messages are captured in the returned TestSpy.
// Call Close when done to stop the drain goroutine.
func NewTestClient() (*Client, *TestSpy) {
	ctx, cancel := context.WithCancel(context.Background())
	spy := &TestSpy{}
	c := &Client{
		send:   make(chan []byte, 256),
		ctx:    ctx,
		cancel: cancel,
	}
	go func() {
		for {
			select {
			case msg, ok := <-c.send:
				if !ok {
					return
				}
				cp := make([]byte, len(msg))
				copy(cp, msg)
				spy.mu.Lock()
				spy.msgs = append(spy.msgs, cp)
				spy.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	return c, spy
}
