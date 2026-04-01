package integration

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"net/http"
	"testing"
	"time"

	"codex-online/server/internal/message"
	"codex-online/server/internal/relay"

	"github.com/coder/websocket"
)

type testGateway struct {
	Relay *relay.Relay
	URL   string
	srv   *http.Server
	ln    net.Listener
}

// startGateway spins up a real HTTP+WebSocket gateway on a random port.
func startGateway(t *testing.T) *testGateway {
	t.Helper()

	r := relay.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}

		client := relay.NewClient(conn)
		defer func() {
			r.RemoveClient(req.Context(), client)
			client.Close()
			conn.CloseNow()
		}()

		for {
			data, err := client.ReadMessage()
			if err != nil {
				return
			}
			_ = r.HandleMessage(req.Context(), client, data)
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	gw := &testGateway{
		Relay: r,
		URL:   "ws://" + ln.Addr().String() + "/ws",
		srv:   srv,
		ln:    ln,
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	})

	return gw
}

func TestTwoClientsJoinSameZone(t *testing.T) {
	gw := startGateway(t)

	c1 := connect(t, gw.URL)
	c1.JoinZone("arena")

	if c1.PeerID != 1 {
		t.Errorf("c1.PeerID = %d, want 1", c1.PeerID)
	}
	if !c1.IsHost {
		t.Error("c1 should be host (first joiner)")
	}

	c2 := connect(t, gw.URL)
	c2.JoinZone("arena")

	if c2.PeerID != 2 {
		t.Errorf("c2.PeerID = %d, want 2", c2.PeerID)
	}
	if c2.IsHost {
		t.Error("c2 should not be host")
	}

	// c1 should receive OpPeerConnected for peer 2.
	peerMsg := c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	connectedPeerID := binary.BigEndian.Uint16(peerMsg.Payload)
	if connectedPeerID != 2 {
		t.Errorf("c1 got PeerConnected for peer %d, want 2", connectedPeerID)
	}

	// c2 should receive OpPeerConnected for peer 1 (existing peer notification).
	existingMsg := c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	existingPeerID := binary.BigEndian.Uint16(existingMsg.Payload)
	if existingPeerID != 1 {
		t.Errorf("c2 got PeerConnected for peer %d, want 1", existingPeerID)
	}
}

func TestPlayerSyncRelayed(t *testing.T) {
	gw := startGateway(t)

	c1 := connect(t, gw.URL)
	c1.JoinZone("arena")

	c2 := connect(t, gw.URL)
	c2.JoinZone("arena")

	// Drain peer notifications.
	c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)

	payload := []byte{0x01, 0x02, 0x03, 0x04}
	c1.SendPlayerSync(payload)

	// c2 should receive it with senderID = c1.PeerID.
	msg := c2.WaitForMessageFrom(message.OpPlayerSync, c1.PeerID, 2*time.Second)
	if !bytes.Equal(msg.Payload, payload) {
		t.Errorf("payload = %x, want %x", msg.Payload, payload)
	}

	// c1 should NOT receive its own PlayerSync (exclude-sender).
	c1.ExpectNoMessage(message.OpPlayerSync, 100*time.Millisecond)
}

func TestBroadcastIncludesSender(t *testing.T) {
	gw := startGateway(t)

	c1 := connect(t, gw.URL)
	c1.JoinZone("arena")

	c2 := connect(t, gw.URL)
	c2.JoinZone("arena")

	// Drain peer notifications.
	c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)

	// OpDamage is broadcast-include-sender (call_local).
	payload := []byte{0xAA, 0xBB}
	c1.SendMessage(message.OpDamage, payload)

	msg1 := c1.WaitForMessage(message.OpDamage, 2*time.Second)
	msg2 := c2.WaitForMessage(message.OpDamage, 2*time.Second)

	if msg1.SenderID != c1.PeerID {
		t.Errorf("c1 got senderID=%d, want %d", msg1.SenderID, c1.PeerID)
	}
	if msg2.SenderID != c1.PeerID {
		t.Errorf("c2 got senderID=%d, want %d", msg2.SenderID, c1.PeerID)
	}
}

func TestDisconnectNotifiesRemaining(t *testing.T) {
	gw := startGateway(t)

	c1 := connect(t, gw.URL)
	c1.JoinZone("arena")

	c2 := connect(t, gw.URL)
	c2.JoinZone("arena")

	// Drain peer notifications.
	c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)

	c2.Disconnect()

	dcMsg := c1.WaitForMessage(message.OpPeerDisconnected, 2*time.Second)
	dcPeerID := binary.BigEndian.Uint16(dcMsg.Payload)
	if dcPeerID != c2.PeerID {
		t.Errorf("disconnect notification peer=%d, want %d", dcPeerID, c2.PeerID)
	}
}

func TestZoneIsolation(t *testing.T) {
	gw := startGateway(t)

	c1 := connect(t, gw.URL)
	c1.JoinZone("zone-a")

	c2 := connect(t, gw.URL)
	c2.JoinZone("zone-b")

	if c1.PeerID != 1 || c2.PeerID != 1 {
		t.Errorf("peer IDs = (%d, %d), want (1, 1)", c1.PeerID, c2.PeerID)
	}

	c1.SendPlayerSync([]byte{0xFF})
	c2.ExpectNoMessage(message.OpPlayerSync, 100*time.Millisecond)
}

func TestThreeClientsRelaying(t *testing.T) {
	gw := startGateway(t)

	c1 := connect(t, gw.URL)
	c1.JoinZone("arena")
	c2 := connect(t, gw.URL)
	c2.JoinZone("arena")
	c3 := connect(t, gw.URL)
	c3.JoinZone("arena")

	// Drain all peer notifications.
	c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c3.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	c3.WaitForMessage(message.OpPeerConnected, 2*time.Second)

	c2.SendPlayerSync([]byte("movement"))

	msg1 := c1.WaitForMessageFrom(message.OpPlayerSync, c2.PeerID, 2*time.Second)
	msg3 := c3.WaitForMessageFrom(message.OpPlayerSync, c2.PeerID, 2*time.Second)

	if string(msg1.Payload) != "movement" {
		t.Errorf("c1 payload = %q, want %q", msg1.Payload, "movement")
	}
	if string(msg3.Payload) != "movement" {
		t.Errorf("c3 payload = %q, want %q", msg3.Payload, "movement")
	}

	c2.ExpectNoMessage(message.OpPlayerSync, 100*time.Millisecond)
}

func TestBroadcastBehavior(t *testing.T) {
	tests := []struct {
		name          string
		opcode        uint16
		excludeSender bool
	}{
		{"PlayerSync", message.OpPlayerSync, true},
		{"EnemySync", message.OpEnemySync, true},
		{"NetFlash", message.OpNetFlash, true},
		{"Damage", message.OpDamage, false},
		{"ProjectileSpawn", message.OpProjectileSpawn, false},
		{"ClassSelect", message.OpClassSelect, false},
		{"ReadyState", message.OpReadyState, false},
		{"PlayerInfo", message.OpPlayerInfo, false},
		{"SpawnPlayers", message.OpSpawnPlayers, false},
		{"StartFight", message.OpStartFight, false},
		{"ShowResult", message.OpShowResult, false},
		{"ResetReady", message.OpResetReady, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := startGateway(t)

			c1 := connect(t, gw.URL)
			c1.JoinZone("arena")
			c2 := connect(t, gw.URL)
			c2.JoinZone("arena")

			c1.WaitForMessage(message.OpPeerConnected, 2*time.Second)
			c2.WaitForMessage(message.OpPeerConnected, 2*time.Second)

			c1.SendMessage(tt.opcode, []byte("data"))

			// Receiver always gets the message.
			c2.WaitForMessage(tt.opcode, 2*time.Second)

			if tt.excludeSender {
				c1.ExpectNoMessage(tt.opcode, 100*time.Millisecond)
			} else {
				c1.WaitForMessage(tt.opcode, 2*time.Second)
			}
		})
	}
}
