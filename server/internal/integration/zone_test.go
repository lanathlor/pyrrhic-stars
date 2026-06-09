package integration

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/zone"

	"github.com/coder/websocket"
)

// testZoneGateway is a minimal gateway with real zone simulation (not relay).
type testZoneGateway struct {
	zones  map[string]*zone.Zone
	URL    string
	srv    *http.Server
	mu     sync.Mutex
	nextID map[string]uint16 // per-zone peer ID counter
}

// startZoneGateway spins up a gateway backed by real zones (20Hz tick loop).
func startZoneGateway(t *testing.T) *testZoneGateway {
	t.Helper()

	gw := &testZoneGateway{
		zones:  make(map[string]*zone.Zone),
		nextID: make(map[string]uint16),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()

		// Buffered send channel
		sendCh := make(chan []byte, 256)
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()

		go func() {
			for {
				select {
				case msg, ok := <-sendCh:
					if !ok {
						return
					}
					_ = conn.Write(ctx, websocket.MessageBinary, msg)
				case <-ctx.Done():
					return
				}
			}
		}()

		sendFn := func(data []byte) {
			select {
			case sendCh <- data:
			default:
			}
		}

		var peerID uint16
		var zoneID string
		var username string

		defer func() {
			if zoneID != "" {
				gw.mu.Lock()
				z := gw.zones[zoneID]
				gw.mu.Unlock()
				if z != nil {
					z.RemoveClient(peerID)
				}
			}
			close(sendCh)
		}()

		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}

			opcode, _, payload, err := message.Decode(data)
			if err != nil {
				continue
			}

			switch {
			case opcode == message.OpSetUsername:
				if len(payload) >= 1 {
					nameLen := int(payload[0])
					if len(payload) >= 1+nameLen {
						username = strings.TrimSpace(string(payload[1 : 1+nameLen]))
					}
				}

			case opcode == message.OpJoinZone:
				zoneID = string(payload)
				// Extract base zone name for level loading (e.g. "arena_test" → "arena")
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
					go z.Run(ctx)
				}
				peerID = gw.nextID[zoneID]
				gw.nextID[zoneID]++
				gw.mu.Unlock()

				if username == "" {
					username = "TestPlayer"
				}
				z.AddClient(&zone.Client{
					PeerID:   peerID,
					Username: username,
					Send:     sendFn,
					SendUDP:  sendFn,
					HasUDP:   func() bool { return true },
				})

				resp := make([]byte, 3)
				binary.BigEndian.PutUint16(resp[0:2], peerID)
				resp[2] = 0
				sendFn(message.Encode(message.OpZoneJoined, 0, resp))

				// Notify existing peers
				peerMsg := message.Encode(message.OpPeerConnected, 0, encodePeerIDBytes(peerID))
				z.Broadcast(peerMsg, peerID)

				for _, existingID := range z.GetPeerIDs() {
					if existingID == peerID {
						continue
					}
					sendFn(message.Encode(message.OpPeerConnected, 0, encodePeerIDBytes(existingID)))
				}

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
	})

	return gw
}

func encodePeerIDBytes(id uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)
	return b
}

// =============================================================================
// Integration test: Gunner fires, observer receives attack state in WorldState
// =============================================================================

func TestGunnerFireBroadcastsAttackStateIntegration(t *testing.T) {
	gw := startZoneGateway(t)

	// --- Connect two clients, join the same arena ---
	shooter := connect(t, gw.URL)
	shooter.SetUsername("Shooter")
	shooter.JoinZone("arena_test")

	observer := connect(t, gw.URL)
	observer.SetUsername("Observer")
	observer.JoinZone("arena_test")

	// Drain peer connected notifications
	shooter.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	observer.WaitForMessage(message.OpPeerConnected, 2*time.Second)

	t.Logf("shooter peer=%d, observer peer=%d", shooter.PeerID, observer.PeerID)

	// Wait for spawn grace period to expire (10 ticks @ 20Hz = 500ms)
	time.Sleep(600 * time.Millisecond)

	// Move both players into the hallway area near enemies.
	// Positions must be within 10 units of spawn (Z=48) to pass teleport check.
	aimPitch := float32(math.Atan2(-1.0, 39.0)) // aiming slightly down at enemy
	shooter.SendPlayerInput(-2, 0.1, 39, 0, 1, aimPitch)
	observer.SendPlayerInput(0, 0.1, 39, 0, 1, 0)
	time.Sleep(200 * time.Millisecond)

	// Drain any buffered world states
	observer.DrainMessages()

	// --- Shooter fires ---
	shooter.SendAbilityInput(entity.ActionShoot, aimPitch)

	// --- Observer should receive WorldState with shooter in PlayerStateAttack ---
	observer.WaitForWorldStateWithPlayerState(shooter.PeerID, uint8(entity.PlayerStateAttack), 3*time.Second)
	t.Log("OK: observer received WorldState with shooter in PlayerStateAttack")

	// --- After cooldown, state should reset to PlayerStateMove ---
	observer.WaitForWorldStateWithPlayerState(shooter.PeerID, uint8(entity.PlayerStateMove), 3*time.Second)
	t.Log("OK: shooter state reset to PlayerStateMove after cooldown")
}

// =============================================================================
// Integration test: Observer receives DamageEvent when shooter hits enemy
// =============================================================================

func TestGunnerHitBroadcastsDamageEventIntegration(t *testing.T) {
	gw := startZoneGateway(t)

	shooter := connect(t, gw.URL)
	shooter.SetUsername("Shooter")
	shooter.JoinZone("arena_hit")

	observer := connect(t, gw.URL)
	observer.SetUsername("Observer")
	observer.JoinZone("arena_hit")

	shooter.WaitForMessage(message.OpPeerConnected, 2*time.Second)
	observer.WaitForMessage(message.OpPeerConnected, 2*time.Second)

	// Wait for spawn grace period to expire (10 ticks @ 20Hz = 500ms)
	time.Sleep(600 * time.Millisecond)

	// Walk both players into the hallway (Z < ArenaEntryZ=40) near enemies,
	// using incremental steps so the server-side speed clamp accepts each move.
	var wg sync.WaitGroup
	wg.Go(func() { shooter.WalkTo(0, 0.1, 48, 0, 0.1, 39, 0.35) })
	wg.Go(func() { observer.WalkTo(0, 0.1, 48, 0, 0.1, 39, 0.35) })
	wg.Wait()

	// Drain
	shooter.DrainMessages()
	observer.DrainMessages()

	// Fire multiple shots at different yaw angles to sweep for mobs that may have
	// moved during the walk. Mobs converge on the player, so they could be in any
	// direction. Each shot is separated by enough time to clear the fire cooldown.
	for _, rotY := range []float32{0, math.Pi, math.Pi / 2, -math.Pi / 2, math.Pi / 4, -math.Pi / 4} {
		shooter.SendPlayerInput(0, 0.1, 39, rotY, 100, -0.1)
		time.Sleep(60 * time.Millisecond)
		shooter.SendAbilityInputAimed(entity.ActionShoot, -0.1, rotY)
		time.Sleep(120 * time.Millisecond)
	}

	// Look for a player-on-enemy DamageEvent (target >= 1000, source_type == 0)
	// There may also be enemy-on-player events from trash mobs, so filter.
	deadline := time.After(3 * time.Second)
	found := false
	for !found {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for player-on-enemy DamageEvent")
		default:
		}
		msg := shooter.WaitForMessage(message.OpDamageEvent, 2*time.Second)
		if len(msg.Payload) < 21 {
			continue
		}
		targetPeer := binary.LittleEndian.Uint16(msg.Payload[0:2])
		sourcePeer := binary.LittleEndian.Uint16(msg.Payload[2:4])
		amount := math.Float32frombits(binary.LittleEndian.Uint32(msg.Payload[4:8]))
		sourceType := msg.Payload[20]

		t.Logf("DamageEvent: target=%d source=%d amount=%.1f type=%d", targetPeer, sourcePeer, amount, sourceType)

		// Player-on-enemy: target is enemy ID (>= 1000), source_type == 0
		if targetPeer >= 1000 && sourceType == 0 {
			found = true
			if sourcePeer != shooter.PeerID {
				t.Errorf("source_peer = %d, want %d (shooter)", sourcePeer, shooter.PeerID)
			}
			if amount <= 0 {
				t.Errorf("damage amount = %.1f, want > 0", amount)
			}
			t.Log("OK: shooter hit an enemy")
		}
	}
}
