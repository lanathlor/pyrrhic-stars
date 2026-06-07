package network

import (
	"context"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/coder/websocket"
)

// Client wraps a WebSocket connection with a buffered send channel.
// UDP fields are set lazily after the client completes the UDP association
// handshake. Zone tick goroutines read udpAddr atomically (lock-free).
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc

	// UDP transport (set once during association, read from zone tick goroutines).
	udpConn *net.UDPConn                // shared server UDP socket
	udpAddr atomic.Pointer[net.UDPAddr] // client's confirmed UDP address
}

// NewClient creates a Client and starts its write pump.
func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		ctx:    ctx,
		cancel: cancel,
	}
	go c.writePump()
	return c
}

func (c *Client) writePump() {
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.Write(c.ctx, websocket.MessageBinary, msg); err != nil {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Send enqueues a message for writing. Drops if the buffer is full
// or the client has been closed.
func (c *Client) Send(data []byte) {
	select {
	case <-c.ctx.Done():
		return
	default:
	}
	select {
	case c.send <- data:
	case <-c.ctx.Done():
	default:
		slog.Warn("send buffer full, dropping message")
	}
}

// ReadMessage blocks until a binary message arrives or the context is cancelled.
func (c *Client) ReadMessage() ([]byte, error) {
	_, data, err := c.conn.Read(c.ctx)
	return data, err
}

// Close cancels the context and drains the send channel.
func (c *Client) Close() {
	c.udpAddr.Store(nil) // signal HasUDP() = false
	c.cancel()
	close(c.send)
}

// CloseNow immediately closes the underlying WebSocket connection.
func (c *Client) CloseNow() {
	_ = c.conn.CloseNow()
}

// AssociateUDP sets the UDP transport fields. Called once from the UDP read
// loop after token validation. Subsequent SendUDP calls use these fields.
func (c *Client) AssociateUDP(conn *net.UDPConn, addr *net.UDPAddr) {
	c.udpConn = conn
	c.udpAddr.Store(addr)
}

// SendUDP writes data directly to the client's UDP address using the shared
// server socket. Synchronous, zero-allocation (no channel, no copy).
// No-op if the client has not completed UDP association.
func (c *Client) SendUDP(data []byte) {
	addr := c.udpAddr.Load()
	if addr == nil || c.udpConn == nil {
		return
	}
	_, _ = c.udpConn.WriteToUDP(data, addr)
}

// HasUDP returns true if the client has completed UDP association.
func (c *Client) HasUDP() bool {
	return c.udpAddr.Load() != nil
}

// UDPAddrString returns the string form of the client's UDP address,
// or empty string if not associated. Used as a map key for cleanup.
func (c *Client) UDPAddrString() string {
	addr := c.udpAddr.Load()
	if addr == nil {
		return ""
	}
	return addr.String()
}
