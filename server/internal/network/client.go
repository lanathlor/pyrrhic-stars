package network

import (
	"context"
	"log/slog"

	"github.com/coder/websocket"
)

// Client wraps a WebSocket connection with a buffered send channel.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
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

// Send enqueues a message for writing. Drops if the buffer is full.
func (c *Client) Send(data []byte) {
	select {
	case c.send <- data:
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
	c.cancel()
	close(c.send)
}

// CloseNow immediately closes the underlying WebSocket connection.
func (c *Client) CloseNow() {
	_ = c.conn.CloseNow()
}
