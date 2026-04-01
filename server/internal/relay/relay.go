package relay

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"

	"codex-online/server/internal/message"
	"codex-online/server/internal/telemetry"

	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Client represents a connected player.
type Client struct {
	PeerID uint16
	ZoneID string
	conn   *websocket.Conn
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
}

// Zone represents a game zone instance (arena, dungeon, etc.).
type Zone struct {
	ID         string
	clients    map[uint16]*Client
	hostID     uint16
	nextPeerID uint16
	mu         sync.RWMutex
}

// Relay manages zones and routes messages between clients.
type Relay struct {
	zones map[string]*Zone
	mu    sync.RWMutex

	// Metrics instruments.
	connGauge      metric.Int64UpDownCounter
	msgCounter     metric.Int64Counter
	zoneGauge      metric.Int64UpDownCounter
	msgSizeHisto   metric.Int64Histogram
}

// New creates a new Relay with OTel metric instruments.
func New() *Relay {
	m := telemetry.Meter()
	connGauge, _ := m.Int64UpDownCounter("gateway.connections",
		metric.WithDescription("Current active WebSocket connections"))
	msgCounter, _ := m.Int64Counter("gateway.messages_relayed",
		metric.WithDescription("Total messages relayed"))
	zoneGauge, _ := m.Int64UpDownCounter("gateway.zones_active",
		metric.WithDescription("Current active zones"))
	msgSizeHisto, _ := m.Int64Histogram("gateway.message_bytes",
		metric.WithDescription("Message size distribution in bytes"))

	return &Relay{
		zones:        make(map[string]*Zone),
		connGauge:    connGauge,
		msgCounter:   msgCounter,
		zoneGauge:    zoneGauge,
		msgSizeHisto: msgSizeHisto,
	}
}

// NewClient wraps a WebSocket connection into a Client and starts its write pump.
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

// writePump serializes all outgoing writes through a single goroutine.
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

// Send queues a message for delivery. Non-blocking; drops if buffer full.
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		slog.Warn("send buffer full, dropping message", "peer_id", c.PeerID)
	}
}

// Close shuts down the client's write pump.
func (c *Client) Close() {
	c.cancel()
	close(c.send)
}

// ReadMessage reads the next binary WebSocket message from the client.
func (c *Client) ReadMessage() ([]byte, error) {
	_, data, err := c.conn.Read(c.ctx)
	return data, err
}

// JoinZone adds a client to a zone, creating the zone if it doesn't exist.
// Returns the assigned peer ID and whether the client is the host.
func (r *Relay) JoinZone(ctx context.Context, client *Client, zoneID string) (peerID uint16, isHost bool, err error) {
	ctx, span := telemetry.Tracer().Start(ctx, "JoinZone",
		trace.WithAttributes(attribute.String("zone.id", zoneID)))
	defer span.End()

	r.mu.Lock()
	zone, exists := r.zones[zoneID]
	if !exists {
		zone = &Zone{
			ID:         zoneID,
			clients:    make(map[uint16]*Client),
			nextPeerID: 1,
		}
		r.zones[zoneID] = zone
		r.zoneGauge.Add(ctx, 1)
	}
	r.mu.Unlock()

	zone.mu.Lock()
	defer zone.mu.Unlock()

	peerID = zone.nextPeerID
	zone.nextPeerID++

	client.PeerID = peerID
	client.ZoneID = zoneID
	zone.clients[peerID] = client

	if peerID == 1 {
		zone.hostID = peerID
		isHost = true
	}

	r.connGauge.Add(ctx, 1)
	span.SetAttributes(
		attribute.Int("peer.id", int(peerID)),
		attribute.Bool("is_host", isHost),
	)

	// Notify the new client about all existing peers.
	for existingID := range zone.clients {
		if existingID == peerID {
			continue
		}
		peerMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(existingID))
		client.Send(peerMsg)
	}

	// Notify existing clients about the new peer.
	joinMsg := message.Encode(message.OpPeerConnected, 0, encodePeerID(peerID))
	for id, c := range zone.clients {
		if id == peerID {
			continue
		}
		c.Send(joinMsg)
	}

	return peerID, isHost, nil
}

// RemoveClient removes a client from its zone and cleans up empty zones.
func (r *Relay) RemoveClient(ctx context.Context, client *Client) {
	if client.ZoneID == "" {
		return
	}

	ctx, span := telemetry.Tracer().Start(ctx, "RemoveClient",
		trace.WithAttributes(
			attribute.String("zone.id", client.ZoneID),
			attribute.Int("peer.id", int(client.PeerID)),
		))
	defer span.End()

	r.mu.RLock()
	zone, exists := r.zones[client.ZoneID]
	r.mu.RUnlock()
	if !exists {
		return
	}

	zone.mu.Lock()
	delete(zone.clients, client.PeerID)
	remaining := len(zone.clients)
	zone.mu.Unlock()

	r.connGauge.Add(ctx, -1)

	// Broadcast disconnect to remaining clients.
	if remaining > 0 {
		msg := message.Encode(message.OpPeerDisconnected, 0, encodePeerID(client.PeerID))
		zone.mu.RLock()
		for _, c := range zone.clients {
			c.Send(msg)
		}
		zone.mu.RUnlock()
	}

	// Clean up empty zones.
	if remaining == 0 {
		r.mu.Lock()
		delete(r.zones, client.ZoneID)
		r.mu.Unlock()
		r.zoneGauge.Add(ctx, -1)
		slog.Info("zone removed (empty)", "zone_id", client.ZoneID)
	}
}

// HandleMessage processes an incoming message from a client and routes it.
func (r *Relay) HandleMessage(ctx context.Context, client *Client, raw []byte) error {
	opcode, _, payload, err := message.Decode(raw)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if message.IsServerHandled(opcode) {
		return r.handleServerMessage(ctx, client, opcode, payload)
	}

	// Re-encode with the server-verified sender ID.
	outMsg := message.Encode(opcode, client.PeerID, payload)

	r.mu.RLock()
	zone, exists := r.zones[client.ZoneID]
	r.mu.RUnlock()
	if !exists {
		return fmt.Errorf("client %d not in a zone", client.PeerID)
	}

	// Record metrics for relayed messages.
	attrs := metric.WithAttributes(attribute.Int("opcode", int(opcode)))
	r.msgCounter.Add(ctx, 1, attrs)
	r.msgSizeHisto.Record(ctx, int64(len(raw)), attrs)

	excludeSender := message.BroadcastExcludeSender(opcode)

	zone.mu.RLock()
	defer zone.mu.RUnlock()
	for id, c := range zone.clients {
		if excludeSender && id == client.PeerID {
			continue
		}
		c.Send(outMsg)
	}

	return nil
}

func (r *Relay) handleServerMessage(ctx context.Context, client *Client, opcode uint16, payload []byte) error {
	_, span := telemetry.Tracer().Start(ctx, "handleServerMessage",
		trace.WithAttributes(
			attribute.Int("opcode", int(opcode)),
			attribute.Int("peer.id", int(client.PeerID)),
		))
	defer span.End()

	switch opcode {
	case message.OpJoinZone:
		zoneID := string(payload)
		if zoneID == "" {
			zoneID = "arena"
		}

		peerID, isHost, err := r.JoinZone(ctx, client, zoneID)
		if err != nil {
			return fmt.Errorf("join zone: %w", err)
		}

		// Send ZoneJoined response.
		resp := make([]byte, 3)
		binary.BigEndian.PutUint16(resp[0:2], peerID)
		if isHost {
			resp[2] = 1
		}
		client.Send(message.Encode(message.OpZoneJoined, 0, resp))
		slog.Info("peer joined zone", "zone_id", zoneID, "peer_id", peerID, "host", isHost)
		return nil

	default:
		return fmt.Errorf("unknown server opcode: 0x%04X", opcode)
	}
}

func encodePeerID(id uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)
	return b
}
