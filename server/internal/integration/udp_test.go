package integration

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/zone"

	"net/http"

	"github.com/coder/websocket"
)

// udpTestGateway is a minimal gateway with real zones, a real UDP server,
// and the full association flow. Exercises the exact code path that Docker runs.
type udpTestGateway struct {
	zones     map[string]*zone.Zone
	URL       string
	UDPServer *network.UDPServer
	srv       *http.Server
	mu        sync.Mutex
	nextID    map[string]uint16
}

func startUDPTestGateway(t *testing.T) *udpTestGateway {
	t.Helper()

	udpSrv, err := network.NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("udp server: %v", err)
	}

	gw := &udpTestGateway{
		zones:     make(map[string]*zone.Zone),
		nextID:    make(map[string]uint16),
		UDPServer: udpSrv,
	}

	go udpSrv.ReadLoop(func(_ uint32, peerID, opcode uint16, payload []byte) {
		// Route player input to the correct zone.
		// For simplicity, look up zone by scanning (test has few zones).
		gw.mu.Lock()
		for _, z := range gw.zones {
			z.QueueInput(peerID, opcode, payload)
		}
		gw.mu.Unlock()
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}

		client := network.NewClient(conn)
		defer func() {
			udpSrv.RemoveClient(client)
			client.Close()
			_ = conn.CloseNow()
		}()

		var peerID uint16
		var zoneID string

		defer func() {
			if zoneID != "" {
				gw.mu.Lock()
				z := gw.zones[zoneID]
				gw.mu.Unlock()
				if z != nil {
					z.RemoveClient(peerID)
				}
			}
		}()

		for {
			data, err := client.ReadMessage()
			if err != nil {
				return
			}

			opcode, _, payload, err := message.Decode(data)
			if err != nil {
				continue
			}

			switch {
			case opcode == message.OpJoinZone:
				zoneID = string(payload)
				baseZone := zoneID
				if idx := strings.Index(zoneID, "_"); idx > 0 {
					baseZone = zoneID[:idx]
				}
				lvl, err := level.Load(baseZone)
				if err != nil {
					return
				}

				gw.mu.Lock()
				z, ok := gw.zones[zoneID]
				if !ok {
					z = zone.New(zoneID, lvl)
					gw.zones[zoneID] = z
					gw.nextID[zoneID] = 1
					go z.Run(req.Context())
				}
				peerID = gw.nextID[zoneID]
				gw.nextID[zoneID]++
				gw.mu.Unlock()

				z.AddClient(&zone.Client{
					PeerID:   peerID,
					Username: "TestPlayer",
					Send:     client.Send,
					SendUDP:  client.SendUDP,
					HasUDP:   client.HasUDP,
				})

				resp := make([]byte, 3)
				binary.BigEndian.PutUint16(resp[0:2], peerID)
				resp[2] = 0
				client.Send(message.Encode(message.OpZoneJoined, 0, resp))

				// Send UDP association token (same as real gateway).
				token := udpSrv.GenerateToken(client, 1)
				assocPayload := make([]byte, 18)
				copy(assocPayload[0:16], token[:])
				binary.BigEndian.PutUint16(assocPayload[16:18], uint16(udpSrv.Port()))
				client.Send(message.Encode(message.OpUDPAssociate, 0, assocPayload))

			case message.IsClientInput(opcode):
				if zoneID != "" {
					gw.mu.Lock()
					z := gw.zones[zoneID]
					gw.mu.Unlock()
					if z != nil {
						z.QueueInput(peerID, opcode, payload)
					}
				}
			}
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	gw.URL = "ws://" + ln.Addr().String() + "/ws"
	gw.srv = srv

	t.Cleanup(func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		_ = udpSrv.Close()
	})

	return gw
}

// TestUDP_AssociationCompletes verifies that a client can complete the full
// UDP association handshake: receive OpUDPAssociate over WS, send the ack
// over UDP, and have the server confirm the association.
func TestUDP_AssociationCompletes(t *testing.T) {
	gw := startUDPTestGateway(t)

	// Connect WS client and join zone.
	tc := connect(t, gw.URL)
	tc.JoinZone("hub")

	// Receive OpUDPAssociate from server.
	assocMsg := tc.WaitForMessage(message.OpUDPAssociate, 2*time.Second)
	if len(assocMsg.Payload) < 18 {
		t.Fatalf("OpUDPAssociate payload too short: %d", len(assocMsg.Payload))
	}
	token := assocMsg.Payload[0:16]
	port := binary.BigEndian.Uint16(assocMsg.Payload[16:18])

	// Open a real UDP socket and send the association ack.
	udpConn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: int(port),
	})
	if err != nil {
		t.Fatalf("dial udp: %v", err)
	}
	defer func() { _ = udpConn.Close() }()

	ack := make([]byte, 20)
	binary.BigEndian.PutUint16(ack[0:2], message.OpUDPAssociateAck)
	binary.BigEndian.PutUint16(ack[2:4], tc.PeerID)
	copy(ack[4:20], token)
	if _, err := udpConn.Write(ack); err != nil {
		t.Fatalf("send ack: %v", err)
	}

	// Wait for association to complete server-side.
	time.Sleep(100 * time.Millisecond)

	// Verify: server should now know this client has UDP.
	// The next world state should arrive over UDP.
	_ = udpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := udpConn.Read(buf)
	if err != nil {
		t.Fatalf("no UDP packet from server within 2s: %v (association likely failed)", err)
	}
	if n < 4 {
		t.Fatalf("UDP packet too short: %d bytes", n)
	}
	opcode := binary.BigEndian.Uint16(buf[0:2])
	if opcode != message.OpWorldState {
		t.Errorf("first UDP packet opcode = 0x%04X, want 0x%04X (OpWorldState)", opcode, message.OpWorldState)
	}
	t.Logf("UDP association confirmed: received %d-byte world state over UDP", n)
}

// TestUDP_PlayerInputReceivedOverUDP verifies that player input sent over
// UDP is received and processed by the zone tick loop.
func TestUDP_PlayerInputReceivedOverUDP(t *testing.T) {
	gw := startUDPTestGateway(t)

	tc := connect(t, gw.URL)
	tc.JoinZone("hub")

	// Complete UDP association.
	assocMsg := tc.WaitForMessage(message.OpUDPAssociate, 2*time.Second)
	token := assocMsg.Payload[0:16]
	port := binary.BigEndian.Uint16(assocMsg.Payload[16:18])

	udpConn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: int(port),
	})
	if err != nil {
		t.Fatalf("dial udp: %v", err)
	}
	defer func() { _ = udpConn.Close() }()

	ack := make([]byte, 20)
	binary.BigEndian.PutUint16(ack[0:2], message.OpUDPAssociateAck)
	binary.BigEndian.PutUint16(ack[2:4], tc.PeerID)
	copy(ack[4:20], token)
	_, _ = udpConn.Write(ack)
	time.Sleep(100 * time.Millisecond)

	// Read current position from world state first (to stay within teleport range).
	_ = udpConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	var spawnX, spawnY, spawnZ float32
	for range 10 {
		n, err := udpConn.Read(buf)
		if err != nil {
			t.Fatalf("read initial world state: %v", err)
		}
		if n < 15 {
			continue
		}
		opcode := binary.BigEndian.Uint16(buf[0:2])
		if opcode != message.OpWorldState {
			continue
		}
		payload := buf[4:n]
		if len(payload) < 15 || payload[4] < 1 {
			continue
		}
		spawnX = getF32LE(payload[7:])
		spawnY = getF32LE(payload[11:])
		spawnZ = getF32LE(payload[15:])
		break
	}
	t.Logf("player spawn at (%.1f, %.1f, %.1f)", spawnX, spawnY, spawnZ)

	// Send player input over UDP: move 0.3 units in X from spawn (within teleport range).
	targetX := spawnX + 0.3
	inputPayload := make([]byte, 4+25) // header + player input
	binary.BigEndian.PutUint16(inputPayload[0:2], message.OpPlayerInput)
	binary.BigEndian.PutUint16(inputPayload[2:4], tc.PeerID)
	putF32LE(inputPayload[4:], targetX)                 // posX
	putF32LE(inputPayload[8:], spawnY)                  // posY
	putF32LE(inputPayload[12:], spawnZ)                 // posZ
	putF32LE(inputPayload[16:], 0.0)                    // rotY
	binary.LittleEndian.PutUint32(inputPayload[20:], 1) // tick
	inputPayload[24] = 0                                // visualState
	putF32LE(inputPayload[25:], 0.0)                    // aimPitch
	_, _ = udpConn.Write(inputPayload)

	// Wait a few ticks for the zone to process the input.
	time.Sleep(200 * time.Millisecond)

	// Read world state over UDP and verify our position shifted.
	for range 10 {
		n, err := udpConn.Read(buf)
		if err != nil {
			t.Fatalf("read udp: %v", err)
		}
		if n < 15 {
			continue
		}
		opcode := binary.BigEndian.Uint16(buf[0:2])
		if opcode != message.OpWorldState {
			continue
		}
		payload := buf[4:n]
		if len(payload) < 11 || payload[4] < 1 {
			continue
		}
		posX := getF32LE(payload[7:])
		if math.Abs(float64(posX-targetX)) < 0.5 {
			t.Logf("player moved to posX=%.2f (target=%.2f) via UDP input", posX, targetX)
			return
		}
	}
	t.Error("player position did not update after UDP input")
}

// TestUDP_NoWorldStateWithoutAssociation verifies that clients without UDP
// association do NOT receive world state. UDP is mandatory.
func TestUDP_NoWorldStateWithoutAssociation(t *testing.T) {
	gw := startUDPTestGateway(t)

	tc := connect(t, gw.URL)
	tc.JoinZone("hub")

	// Receive OpUDPAssociate but DON'T send the ack (simulate NAT/firewall).
	tc.WaitForMessage(message.OpUDPAssociate, 2*time.Second)

	// World state must NOT arrive over WS; UDP is required.
	tc.ExpectNoMessage(message.OpWorldState, 500*time.Millisecond)
}

func putF32LE(b []byte, v float32) {
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
}

func getF32LE(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}
