package integration

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	"codex-online/server/internal/message"

	"github.com/coder/websocket"
)

// Message is a decoded wire message for assertions.
type Message struct {
	Opcode   uint16
	SenderID uint16
	Payload  []byte
}

// TestClient is a WebSocket client for integration tests.
type TestClient struct {
	t      *testing.T
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc

	mu   sync.Mutex
	msgs []Message
	cond *sync.Cond

	// Set after JoinZone succeeds.
	PeerID uint16
	IsHost bool
}

// connect dials the gateway and starts the read loop.
func connect(t *testing.T, gatewayURL string) *TestClient {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	conn, _, err := websocket.Dial(ctx, gatewayURL, nil)
	if err != nil {
		cancel()
		t.Fatalf("dial %s: %v", gatewayURL, err)
	}

	tc := &TestClient{
		t:      t,
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
	tc.cond = sync.NewCond(&tc.mu)

	go tc.readLoop()

	t.Cleanup(func() {
		conn.Close(websocket.StatusNormalClosure, "test done")
		cancel()
	})

	return tc
}

// readLoop decodes all incoming messages into tc.msgs.
func (tc *TestClient) readLoop() {
	for {
		_, data, err := tc.conn.Read(tc.ctx)
		if err != nil {
			return
		}

		opcode, senderID, payload, err := message.Decode(data)
		if err != nil {
			continue
		}

		tc.mu.Lock()
		tc.msgs = append(tc.msgs, Message{
			Opcode:   opcode,
			SenderID: senderID,
			Payload:  append([]byte(nil), payload...),
		})
		tc.cond.Broadcast()
		tc.mu.Unlock()
	}
}

// send writes a raw binary message on the WebSocket.
func (tc *TestClient) send(data []byte) {
	tc.t.Helper()
	if err := tc.conn.Write(tc.ctx, websocket.MessageBinary, data); err != nil {
		tc.t.Fatalf("send: %v", err)
	}
}

// JoinZone sends OpJoinZone and waits for the OpZoneJoined response.
func (tc *TestClient) JoinZone(zoneID string) {
	tc.t.Helper()
	tc.send(message.Encode(message.OpJoinZone, 0, []byte(zoneID)))

	msg := tc.WaitForMessage(message.OpZoneJoined, 2*time.Second)
	if len(msg.Payload) < 3 {
		tc.t.Fatalf("ZoneJoined payload too short: %d bytes", len(msg.Payload))
	}
	tc.PeerID = binary.BigEndian.Uint16(msg.Payload[0:2])
	tc.IsHost = msg.Payload[2] == 1
}

// SendPlayerSync sends an OpPlayerSync message.
func (tc *TestClient) SendPlayerSync(payload []byte) {
	tc.t.Helper()
	tc.send(message.Encode(message.OpPlayerSync, 0, payload))
}

// SendMessage sends an arbitrary opcode+payload.
func (tc *TestClient) SendMessage(opcode uint16, payload []byte) {
	tc.t.Helper()
	tc.send(message.Encode(opcode, 0, payload))
}

// WaitForMessage blocks until a message with the given opcode arrives,
// or fails the test after timeout.
func (tc *TestClient) WaitForMessage(opcode uint16, timeout time.Duration) Message {
	tc.t.Helper()
	deadline := time.Now().Add(timeout)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	for {
		for i, m := range tc.msgs {
			if m.Opcode == opcode {
				tc.msgs = append(tc.msgs[:i], tc.msgs[i+1:]...)
				return m
			}
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			tc.t.Fatalf("timeout waiting for opcode 0x%04X (%d buffered messages)",
				opcode, len(tc.msgs))
		}

		done := make(chan struct{})
		go func() {
			timer := time.NewTimer(remaining)
			defer timer.Stop()
			select {
			case <-timer.C:
				tc.cond.Broadcast()
			case <-done:
			}
		}()

		tc.cond.Wait()
		close(done)
	}
}

// WaitForMessageFrom is like WaitForMessage but also matches on senderID.
func (tc *TestClient) WaitForMessageFrom(opcode uint16, senderID uint16, timeout time.Duration) Message {
	tc.t.Helper()
	deadline := time.Now().Add(timeout)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	for {
		for i, m := range tc.msgs {
			if m.Opcode == opcode && m.SenderID == senderID {
				tc.msgs = append(tc.msgs[:i], tc.msgs[i+1:]...)
				return m
			}
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			tc.t.Fatalf("timeout waiting for opcode 0x%04X from sender %d", opcode, senderID)
		}

		done := make(chan struct{})
		go func() {
			timer := time.NewTimer(remaining)
			defer timer.Stop()
			select {
			case <-timer.C:
				tc.cond.Broadcast()
			case <-done:
			}
		}()

		tc.cond.Wait()
		close(done)
	}
}

// ExpectNoMessage asserts that no message with the given opcode arrives
// within the specified duration.
func (tc *TestClient) ExpectNoMessage(opcode uint16, wait time.Duration) {
	tc.t.Helper()
	time.Sleep(wait)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	for _, m := range tc.msgs {
		if m.Opcode == opcode {
			tc.t.Errorf("unexpected message with opcode 0x%04X", opcode)
			return
		}
	}
}

// DrainMessages returns all currently buffered messages and clears the buffer.
func (tc *TestClient) DrainMessages() []Message {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	out := make([]Message, len(tc.msgs))
	copy(out, tc.msgs)
	tc.msgs = tc.msgs[:0]
	return out
}

// Disconnect gracefully closes the WebSocket connection.
func (tc *TestClient) Disconnect() {
	tc.conn.Close(websocket.StatusNormalClosure, "disconnect")
	tc.cancel()
}
