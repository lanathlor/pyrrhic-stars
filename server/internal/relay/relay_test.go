package relay

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"codex-online/server/internal/message"
)

var ctx = context.Background()

// mockClient creates a Client with no real connection, just a send channel.
func mockClient() *Client {
	return &Client{
		send: make(chan []byte, 64),
	}
}

func drainMessages(c *Client, timeout time.Duration) [][]byte {
	var msgs [][]byte
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return msgs
			}
			msgs = append(msgs, msg)
		case <-timer.C:
			return msgs
		}
	}
}

func TestJoinZoneCreatesZone(t *testing.T) {
	r := New()
	c := mockClient()

	peerID, isHost, err := r.JoinZone(ctx, c, "test-zone")
	if err != nil {
		t.Fatalf("JoinZone: %v", err)
	}
	if peerID != 1 {
		t.Errorf("peerID = %d, want 1", peerID)
	}
	if !isHost {
		t.Error("first joiner should be host")
	}
	if c.PeerID != 1 {
		t.Errorf("client.PeerID = %d, want 1", c.PeerID)
	}
	if c.ZoneID != "test-zone" {
		t.Errorf("client.ZoneID = %q, want %q", c.ZoneID, "test-zone")
	}

	r.mu.RLock()
	_, exists := r.zones["test-zone"]
	r.mu.RUnlock()
	if !exists {
		t.Error("zone should exist after JoinZone")
	}
}

func TestJoinZoneAssignsIncrementingIDs(t *testing.T) {
	r := New()
	c1 := mockClient()
	c2 := mockClient()
	c3 := mockClient()

	id1, host1, _ := r.JoinZone(ctx, c1, "arena")
	id2, host2, _ := r.JoinZone(ctx, c2, "arena")
	id3, host3, _ := r.JoinZone(ctx, c3, "arena")

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Errorf("IDs = (%d, %d, %d), want (1, 2, 3)", id1, id2, id3)
	}
	if !host1 || host2 || host3 {
		t.Errorf("host = (%v, %v, %v), want (true, false, false)", host1, host2, host3)
	}
}

func TestJoinZoneNotifiesPeers(t *testing.T) {
	r := New()
	c1 := mockClient()
	c2 := mockClient()

	if _, _, err := r.JoinZone(ctx, c1, "arena"); err != nil {
		t.Fatalf("JoinZone c1: %v", err)
	}
	// Drain any messages c1 got (none expected since it's first).
	drainMessages(c1, 10*time.Millisecond)

	if _, _, err := r.JoinZone(ctx, c2, "arena"); err != nil {
		t.Fatalf("JoinZone c2: %v", err)
	}

	// c1 should get PeerConnected(2).
	msgs := drainMessages(c1, 50*time.Millisecond)
	if len(msgs) != 1 {
		t.Fatalf("c1 got %d messages, want 1", len(msgs))
	}
	opcode, _, payload, _ := message.Decode(msgs[0])
	if opcode != message.OpPeerConnected {
		t.Errorf("opcode = 0x%04X, want OpPeerConnected", opcode)
	}
	peerIDNotified := binary.BigEndian.Uint16(payload)
	if peerIDNotified != 2 {
		t.Errorf("notified peer ID = %d, want 2", peerIDNotified)
	}

	// c2 should get PeerConnected(1) — existing peer notification.
	msgs2 := drainMessages(c2, 50*time.Millisecond)
	if len(msgs2) != 1 {
		t.Fatalf("c2 got %d messages, want 1", len(msgs2))
	}
	_, _, payload2, _ := message.Decode(msgs2[0])
	existingID := binary.BigEndian.Uint16(payload2)
	if existingID != 1 {
		t.Errorf("c2 got existing peer ID = %d, want 1", existingID)
	}
}

func TestRemoveClientBroadcastsDisconnect(t *testing.T) {
	r := New()
	c1 := mockClient()
	c2 := mockClient()

	if _, _, err := r.JoinZone(ctx, c1, "arena"); err != nil {
		t.Fatalf("JoinZone c1: %v", err)
	}
	if _, _, err := r.JoinZone(ctx, c2, "arena"); err != nil {
		t.Fatalf("JoinZone c2: %v", err)
	}
	drainMessages(c1, 10*time.Millisecond)
	drainMessages(c2, 10*time.Millisecond)

	r.RemoveClient(ctx, c2)

	msgs := drainMessages(c1, 50*time.Millisecond)
	if len(msgs) != 1 {
		t.Fatalf("c1 got %d messages after disconnect, want 1", len(msgs))
	}
	opcode, _, payload, _ := message.Decode(msgs[0])
	if opcode != message.OpPeerDisconnected {
		t.Errorf("opcode = 0x%04X, want OpPeerDisconnected", opcode)
	}
	dcPeer := binary.BigEndian.Uint16(payload)
	if dcPeer != 2 {
		t.Errorf("disconnected peer = %d, want 2", dcPeer)
	}
}

func TestRemoveClientCleansUpEmptyZone(t *testing.T) {
	r := New()
	c := mockClient()

	if _, _, err := r.JoinZone(ctx, c, "temp-zone"); err != nil {
		t.Fatalf("JoinZone: %v", err)
	}
	r.RemoveClient(ctx, c)

	r.mu.RLock()
	_, exists := r.zones["temp-zone"]
	r.mu.RUnlock()
	if exists {
		t.Error("empty zone should be cleaned up")
	}
}

func TestHandleMessageBroadcastExcludeSender(t *testing.T) {
	r := New()
	c1 := mockClient()
	c2 := mockClient()
	c3 := mockClient()

	if _, _, err := r.JoinZone(ctx, c1, "arena"); err != nil {
		t.Fatalf("JoinZone c1: %v", err)
	}
	if _, _, err := r.JoinZone(ctx, c2, "arena"); err != nil {
		t.Fatalf("JoinZone c2: %v", err)
	}
	if _, _, err := r.JoinZone(ctx, c3, "arena"); err != nil {
		t.Fatalf("JoinZone c3: %v", err)
	}
	drainMessages(c1, 10*time.Millisecond)
	drainMessages(c2, 10*time.Millisecond)
	drainMessages(c3, 10*time.Millisecond)

	// PlayerSync should exclude sender.
	syncMsg := message.Encode(message.OpPlayerSync, 1, []byte("pos-data"))
	if err := r.HandleMessage(ctx, c1, syncMsg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	// c1 should NOT receive it.
	msgs1 := drainMessages(c1, 50*time.Millisecond)
	if len(msgs1) != 0 {
		t.Errorf("sender got %d messages, want 0", len(msgs1))
	}

	// c2 and c3 should receive it.
	msgs2 := drainMessages(c2, 50*time.Millisecond)
	msgs3 := drainMessages(c3, 50*time.Millisecond)
	if len(msgs2) != 1 || len(msgs3) != 1 {
		t.Errorf("receivers got (%d, %d) messages, want (1, 1)", len(msgs2), len(msgs3))
	}

	// Verify the re-encoded sender ID is the server-verified one.
	opcode, senderID, _, _ := message.Decode(msgs2[0])
	if opcode != message.OpPlayerSync {
		t.Errorf("opcode = 0x%04X, want OpPlayerSync", opcode)
	}
	if senderID != 1 {
		t.Errorf("senderID = %d, want 1", senderID)
	}
}

func TestHandleMessageBroadcastIncludeSender(t *testing.T) {
	r := New()
	c1 := mockClient()
	c2 := mockClient()

	if _, _, err := r.JoinZone(ctx, c1, "arena"); err != nil {
		t.Fatalf("JoinZone c1: %v", err)
	}
	if _, _, err := r.JoinZone(ctx, c2, "arena"); err != nil {
		t.Fatalf("JoinZone c2: %v", err)
	}
	drainMessages(c1, 10*time.Millisecond)
	drainMessages(c2, 10*time.Millisecond)

	// Damage should include sender (call_local emulation).
	dmgMsg := message.Encode(message.OpDamage, 1, []byte("dmg-data"))
	if err := r.HandleMessage(ctx, c1, dmgMsg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	msgs1 := drainMessages(c1, 50*time.Millisecond)
	msgs2 := drainMessages(c2, 50*time.Millisecond)
	if len(msgs1) != 1 {
		t.Errorf("sender got %d messages, want 1 (call_local)", len(msgs1))
	}
	if len(msgs2) != 1 {
		t.Errorf("receiver got %d messages, want 1", len(msgs2))
	}
}

func TestHandleMessageJoinZone(t *testing.T) {
	r := New()
	c := mockClient()

	joinMsg := message.Encode(message.OpJoinZone, 0, []byte("my-zone"))
	err := r.HandleMessage(ctx, c, joinMsg)
	if err != nil {
		t.Fatalf("HandleMessage JoinZone: %v", err)
	}

	if c.PeerID != 1 {
		t.Errorf("peer ID = %d, want 1", c.PeerID)
	}
	if c.ZoneID != "my-zone" {
		t.Errorf("zone ID = %q, want %q", c.ZoneID, "my-zone")
	}

	// Should have received ZoneJoined.
	msgs := drainMessages(c, 50*time.Millisecond)
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1 (ZoneJoined)", len(msgs))
	}
	opcode, _, payload, _ := message.Decode(msgs[0])
	if opcode != message.OpZoneJoined {
		t.Errorf("opcode = 0x%04X, want OpZoneJoined", opcode)
	}
	assignedID := binary.BigEndian.Uint16(payload[0:2])
	isHost := payload[2]
	if assignedID != 1 || isHost != 1 {
		t.Errorf("ZoneJoined: peerID=%d isHost=%d, want 1/1", assignedID, isHost)
	}
}

func TestSeparateZonesAreIsolated(t *testing.T) {
	r := New()
	c1 := mockClient()
	c2 := mockClient()

	if _, _, err := r.JoinZone(ctx, c1, "zone-a"); err != nil {
		t.Fatalf("JoinZone c1: %v", err)
	}
	if _, _, err := r.JoinZone(ctx, c2, "zone-b"); err != nil {
		t.Fatalf("JoinZone c2: %v", err)
	}
	drainMessages(c1, 10*time.Millisecond)
	drainMessages(c2, 10*time.Millisecond)

	// Both should be peer 1 in their own zone.
	if c1.PeerID != 1 || c2.PeerID != 1 {
		t.Errorf("peer IDs = (%d, %d), want (1, 1)", c1.PeerID, c2.PeerID)
	}

	// Message from zone-a should not reach zone-b.
	syncMsg := message.Encode(message.OpPlayerSync, 1, []byte("data"))
	if err := r.HandleMessage(ctx, c1, syncMsg); err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}

	msgs := drainMessages(c2, 50*time.Millisecond)
	if len(msgs) != 0 {
		t.Errorf("cross-zone leak: c2 got %d messages", len(msgs))
	}
}
