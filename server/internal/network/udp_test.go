package network

import (
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"
)

func TestUDPServer_GenerateToken_Unique(t *testing.T) {
	srv, err := NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	c1, _ := NewTestClient()
	defer c1.Close()
	c2, _ := NewTestClient()
	defer c2.Close()

	tok1 := srv.GenerateToken(c1, 1)
	tok2 := srv.GenerateToken(c2, 2)

	if tok1 == tok2 {
		t.Error("tokens should be unique")
	}
}

func TestUDPServer_Association(t *testing.T) {
	srv, err := NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	dispatched := make(chan struct{}, 1)
	go srv.ReadLoop(func(sessID uint32, _, opcode uint16, _ []byte) {
		if sessID == 1 && opcode == 0x0030 {
			dispatched <- struct{}{}
		}
	})

	client, _ := NewTestClient()
	defer client.Close()
	token := srv.GenerateToken(client, 1)

	// Open a UDP socket and send the association packet.
	udpConn, err := net.DialUDP("udp", nil, srv.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpConn.Close() }()

	// Build OpUDPAssociateAck: [opcode:2 BE][sender:2 BE][token:16]
	assoc := make([]byte, 20)
	binary.BigEndian.PutUint16(assoc[0:2], 0xFF11) // OpUDPAssociateAck
	binary.BigEndian.PutUint16(assoc[2:4], 1)      // peer ID
	copy(assoc[4:20], token[:])
	if _, err := udpConn.Write(assoc); err != nil {
		t.Fatal(err)
	}

	// Wait for association to complete.
	time.Sleep(50 * time.Millisecond)

	if !client.HasUDP() {
		t.Fatal("client should have UDP after association")
	}

	// Send a player input packet and verify dispatch.
	input := make([]byte, 8)
	binary.BigEndian.PutUint16(input[0:2], 0x0030) // OpPlayerInput
	binary.BigEndian.PutUint16(input[2:4], 1)      // peer ID
	if _, err := udpConn.Write(input); err != nil {
		t.Fatal(err)
	}

	select {
	case <-dispatched:
		// success
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for dispatched input")
	}
}

func TestUDPServer_RejectsUnknownSource(t *testing.T) {
	srv, err := NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	dispatched := make(chan struct{}, 1)
	go srv.ReadLoop(func(_ uint32, _, _ uint16, _ []byte) {
		dispatched <- struct{}{}
	})

	// Send a packet without associating first.
	udpConn, err := net.DialUDP("udp", nil, srv.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpConn.Close() }()

	input := make([]byte, 8)
	binary.BigEndian.PutUint16(input[0:2], 0x0030)
	binary.BigEndian.PutUint16(input[2:4], 1)
	if _, err := udpConn.Write(input); err != nil {
		t.Fatal(err)
	}

	select {
	case <-dispatched:
		t.Fatal("should not dispatch from unknown source")
	case <-time.After(100 * time.Millisecond):
		// success: packet was dropped
	}
}

func TestUDPServer_InvalidTokenIgnored(t *testing.T) {
	srv, err := NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	client, _ := NewTestClient()
	defer client.Close()
	_ = srv.GenerateToken(client, 1)

	go srv.ReadLoop(func(_ uint32, _, _ uint16, _ []byte) {})

	udpConn, err := net.DialUDP("udp", nil, srv.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpConn.Close() }()

	// Send association with wrong token.
	assoc := make([]byte, 20)
	binary.BigEndian.PutUint16(assoc[0:2], 0xFF11)
	binary.BigEndian.PutUint16(assoc[2:4], 1)
	// token bytes are all zero, which won't match

	if _, err := udpConn.Write(assoc); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	if client.HasUDP() {
		t.Fatal("client should NOT have UDP with wrong token")
	}
}

func TestUDPServer_RemoveClient(t *testing.T) {
	srv, err := NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	go srv.ReadLoop(func(_ uint32, _, _ uint16, _ []byte) {})

	client, _ := NewTestClient()
	defer client.Close()
	token := srv.GenerateToken(client, 1)

	udpConn, err := net.DialUDP("udp", nil, srv.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpConn.Close() }()

	// Associate.
	assoc := make([]byte, 20)
	binary.BigEndian.PutUint16(assoc[0:2], 0xFF11)
	binary.BigEndian.PutUint16(assoc[2:4], 1)
	copy(assoc[4:20], token[:])
	if _, err := udpConn.Write(assoc); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	if !client.HasUDP() {
		t.Fatal("expected UDP association")
	}

	srv.RemoveClient(client)

	// Route should be cleared.
	srv.mu.Lock()
	routeCount := len(srv.routes)
	srv.mu.Unlock()
	if routeCount != 0 {
		t.Errorf("routes should be empty after RemoveClient, got %d", routeCount)
	}
}

// TestUDPServer_PayloadNotCorruptedByNextRead verifies that the payload
// passed to the dispatch callback is safe to store and read later, even
// after the next UDP packet arrives. This reproduces the real QueueInput
// pattern: the callback stores the raw payload slice (no copy), then a
// later goroutine reads it. If the read loop reuses a shared buffer
// without copying, the stored slice points into overwritten memory.
func TestUDPServer_PayloadNotCorruptedByNextRead(t *testing.T) {
	srv, err := NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	// Simulate QueueInput: store raw payload slices without copying.
	var mu sync.Mutex
	var stored [][]byte
	done := make(chan struct{})

	go srv.ReadLoop(func(_ uint32, _, _ uint16, payload []byte) {
		mu.Lock()
		// Store the raw slice — exactly what QueueInput does.
		stored = append(stored, payload)
		if len(stored) == 2 {
			close(done)
		}
		mu.Unlock()
	})

	// Associate a client.
	client, _ := NewTestClient()
	defer client.Close()
	token := srv.GenerateToken(client, 1)

	udpConn, err := net.DialUDP("udp", nil, srv.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpConn.Close() }()

	assoc := make([]byte, 20)
	binary.BigEndian.PutUint16(assoc[0:2], 0xFF11)
	binary.BigEndian.PutUint16(assoc[2:4], 1)
	copy(assoc[4:20], token[:])
	_, _ = udpConn.Write(assoc)
	time.Sleep(50 * time.Millisecond)

	// Send two packets with distinct payloads.
	pkt1 := make([]byte, 8)
	binary.BigEndian.PutUint16(pkt1[0:2], 0x0030) // OpPlayerInput
	binary.BigEndian.PutUint16(pkt1[2:4], 1)
	pkt1[4], pkt1[5], pkt1[6], pkt1[7] = 0xAA, 0xBB, 0xCC, 0xDD

	pkt2 := make([]byte, 8)
	binary.BigEndian.PutUint16(pkt2[0:2], 0x0030)
	binary.BigEndian.PutUint16(pkt2[2:4], 1)
	pkt2[4], pkt2[5], pkt2[6], pkt2[7] = 0x11, 0x22, 0x33, 0x44

	_, _ = udpConn.Write(pkt1)
	time.Sleep(10 * time.Millisecond) // ensure ordering
	_, _ = udpConn.Write(pkt2)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for 2 dispatches")
	}

	mu.Lock()
	defer mu.Unlock()

	// After both packets are dispatched, read the FIRST stored payload.
	// If the read loop reuses a shared buffer, stored[0] now contains
	// pkt2's data (0x11) instead of pkt1's data (0xAA).
	if stored[0][0] != 0xAA {
		t.Errorf("first payload corrupted by second read: got 0x%02X, want 0xAA "+
			"(shared buffer not copied before dispatch)", stored[0][0])
	}
	if stored[1][0] != 0x11 {
		t.Errorf("second payload wrong: got 0x%02X, want 0x11", stored[1][0])
	}
}

func TestClient_SendUDP_NoAssociation(t *testing.T) {
	client, _ := NewTestClient()
	defer client.Close()

	// Should not panic when UDP is not associated.
	client.SendUDP([]byte{1, 2, 3})

	if client.HasUDP() {
		t.Error("HasUDP should be false without association")
	}
}
