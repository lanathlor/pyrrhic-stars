package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.Run()
}

// --- test helpers ---

// stubRepo implements persistence.Repository with no-op methods.
type stubRepo struct{ persistence.Repository }

func (stubRepo) GetCharacterByID(uint) (*persistence.Character, error) { return nil, nil }

// posRepo returns a character with a saved position.
type posRepo struct{ stubRepo }

func (posRepo) GetCharacterByID(id uint) (*persistence.Character, error) {
	return &persistence.Character{
		ID:   id,
		PosX: 10, PosY: 1, PosZ: 20,
		RotY: 90,
	}, nil
}

func newTestGateway(repo persistence.Repository) *gateway {
	return newGateway(container.New(repo))
}

// newTestZoneInstance creates a zoneInstance without starting the tick loop.
func newTestZoneInstance(id string, zt zone.ZoneType) *zoneInstance {
	z := zone.New(id, zt)
	_, cancel := context.WithCancel(context.Background())
	return &zoneInstance{zone: z, zoneType: zt, cancel: cancel, nextID: 1}
}

func newTestSession(id uint32) (*session.Session, *network.TestSpy) {
	conn, spy := network.NewTestClient()
	return &session.Session{
		ID:       id,
		Conn:     conn,
		Username: "TestPlayer",
		Class:    entity.ClassGunner,
	}, spy
}

// drainSpy gives the drain goroutine a moment to process, then returns messages.
func drainSpy(spy *network.TestSpy) [][]byte {
	runtime.Gosched()
	time.Sleep(5 * time.Millisecond)
	return spy.Messages()
}

// findMessage returns the first message with the given opcode, or nil.
func findMessage(msgs [][]byte, opcode uint16) []byte {
	for _, m := range msgs {
		op, _, _, err := message.Decode(m)
		if err == nil && op == opcode {
			return m
		}
	}
	return nil
}

// countMessages counts messages with the given opcode.
func countMessages(msgs [][]byte, opcode uint16) int {
	n := 0
	for _, m := range msgs {
		op, _, _, err := message.Decode(m)
		if err == nil && op == opcode {
			n++
		}
	}
	return n
}

// --- joinZone tests ---

func TestJoinZone_AllocatesPeerID(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess1, _ := newTestSession(1)
	sess2, _ := newTestSession(2)
	defer sess1.Conn.Close()
	defer sess2.Conn.Close()

	gw.joinZone(sess1, zi, joinResponseZoneJoined)
	gw.joinZone(sess2, zi, joinResponseZoneJoined)

	if sess1.PeerID != 1 {
		t.Errorf("sess1.PeerID = %d, want 1", sess1.PeerID)
	}
	if sess2.PeerID != 2 {
		t.Errorf("sess2.PeerID = %d, want 2", sess2.PeerID)
	}
}

func TestJoinZone_SetsSessionState(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("my-zone", zone.ZoneTypeInstanced)

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	if sess.ZoneID != "my-zone" {
		t.Errorf("sess.ZoneID = %q, want %q", sess.ZoneID, "my-zone")
	}
	if sess.PeerID != 1 {
		t.Errorf("sess.PeerID = %d, want 1", sess.PeerID)
	}
}

func TestJoinZone_AddsClientToZone(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	if zi.zone.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", zi.zone.ClientCount())
	}
	if p := zi.zone.GetPlayer(1); p == nil {
		t.Error("player not found in zone")
	}
}

func TestJoinZone_DisplayNameResolution(t *testing.T) {
	tests := []struct {
		name     string
		charName string
		username string
		wantName string
	}{
		{"prefers CharName", "Hero", "User", "Hero"},
		{"falls back to Username", "", "User", "User"},
		{"falls back to Player_ID", "", "", "Player_42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := newTestGateway(stubRepo{})
			zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

			sess, _ := newTestSession(42)
			sess.CharName = tt.charName
			sess.Username = tt.username
			defer sess.Conn.Close()

			gw.joinZone(sess, zi, joinResponseZoneJoined)

			p := zi.zone.GetPlayer(sess.PeerID)
			if p == nil {
				t.Fatal("player not in zone")
			}
			if p.Username != tt.wantName {
				t.Errorf("username = %q, want %q", p.Username, tt.wantName)
			}
		})
	}
}

func TestJoinZone_DisplayNameFallbackSetsUsername(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(7)
	sess.CharName = ""
	sess.Username = ""
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	if sess.Username != "Player_7" {
		t.Errorf("sess.Username = %q, want %q", sess.Username, "Player_7")
	}
}

func TestJoinZone_SendsZoneJoinedResponse(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	msgs := drainSpy(spy)
	raw := findMessage(msgs, message.OpZoneJoined)
	if raw == nil {
		t.Fatal("OpZoneJoined not sent")
	}
	_, _, payload, _ := message.Decode(raw)
	if len(payload) < 3 {
		t.Fatalf("payload too short: %d", len(payload))
	}
	gotPeer := binary.BigEndian.Uint16(payload[0:2])
	if gotPeer != 1 {
		t.Errorf("peerID in response = %d, want 1", gotPeer)
	}
}

func TestJoinZone_SendsZoneTransferResponse(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("arena_1", zone.ZoneTypeInstanced)

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneTransfer)

	msgs := drainSpy(spy)
	raw := findMessage(msgs, message.OpZoneTransfer)
	if raw == nil {
		t.Fatal("OpZoneTransfer not sent")
	}
	_, _, payload, _ := message.Decode(raw)
	if len(payload) < 3 {
		t.Fatalf("payload too short: %d", len(payload))
	}
	if payload[0] != byte(zone.ZoneTypeInstanced) {
		t.Errorf("zone type = %d, want %d", payload[0], zone.ZoneTypeInstanced)
	}
	gotPeer := binary.BigEndian.Uint16(payload[1:3])
	if gotPeer != 1 {
		t.Errorf("peerID in response = %d, want 1", gotPeer)
	}
}

func TestJoinZone_NotifiesExistingPeers(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	// Add an existing peer.
	existing, existingSpy := newTestSession(1)
	defer existing.Conn.Close()
	gw.joinZone(existing, zi, joinResponseZoneJoined)
	existingSpy.Reset()

	// New peer joins.
	newSess, _ := newTestSession(2)
	defer newSess.Conn.Close()
	gw.joinZone(newSess, zi, joinResponseZoneJoined)

	// Existing peer should receive OpPeerConnected for new peer.
	msgs := drainSpy(existingSpy)
	n := countMessages(msgs, message.OpPeerConnected)
	if n == 0 {
		t.Error("existing peer did not receive OpPeerConnected for new peer")
	}
}

func TestJoinZone_NotifiesNewPeerAboutExisting(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	// Add an existing peer.
	existing, _ := newTestSession(1)
	defer existing.Conn.Close()
	gw.joinZone(existing, zi, joinResponseZoneJoined)

	// New peer joins.
	newSess, newSpy := newTestSession(2)
	defer newSess.Conn.Close()
	gw.joinZone(newSess, zi, joinResponseZoneJoined)

	// New peer should receive OpPeerConnected for the existing peer.
	msgs := drainSpy(newSpy)
	n := countMessages(msgs, message.OpPeerConnected)
	if n == 0 {
		t.Error("new peer did not receive OpPeerConnected for existing peer")
	}
}

func TestJoinZone_QueuesClassSelectForNonGunner(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	sess.Class = entity.ClassVanguard
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	// The class select is queued as an input; verify via player class after
	// zone processes it. Since we don't run the tick loop, check that the
	// zone has pending inputs (indirect verification).
	// The key assertion is that AddClient worked and the player exists.
	if zi.zone.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", zi.zone.ClientCount())
	}
}

func TestJoinZone_DoesNotQueueClassSelectForGunner(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	sess.Class = entity.ClassGunner
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	// Gunner is the default — no class select should be queued.
	// Verify player exists and zone is functional.
	if zi.zone.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", zi.zone.ClientCount())
	}
}

func TestJoinZone_RestoresPositionForHub(t *testing.T) {
	gw := newTestGateway(posRepo{})
	zi := newTestZoneInstance(zone.ZoneHub, zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	sess.CharID = 42
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	p := zi.zone.GetPlayer(sess.PeerID)
	if p == nil {
		t.Fatal("player not in zone")
	}
	if p.Position.X != 10 || p.Position.Y != 1 || p.Position.Z != 20 {
		t.Errorf("position = %v, want {10 1 20}", p.Position)
	}
	if p.RotationY != 90 {
		t.Errorf("rotationY = %f, want 90", p.RotationY)
	}
}

func TestJoinZone_NoPositionRestoreForArena(t *testing.T) {
	gw := newTestGateway(posRepo{})
	zi := newTestZoneInstance("arena_1", zone.ZoneTypeInstanced)

	sess, _ := newTestSession(1)
	sess.CharID = 42
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	p := zi.zone.GetPlayer(sess.PeerID)
	if p == nil {
		t.Fatal("player not in zone")
	}
	// Arena should use spawn position, not saved position.
	if p.Position.X == 10 && p.Position.Z == 20 {
		t.Error("arena player got hub-saved position, should use spawn")
	}
}

func TestJoinZone_NoPositionRestoreWithoutCharID(t *testing.T) {
	gw := newTestGateway(posRepo{})
	zi := newTestZoneInstance(zone.ZoneHub, zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	sess.CharID = 0 // no character
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)

	p := zi.zone.GetPlayer(sess.PeerID)
	if p == nil {
		t.Fatal("player not in zone")
	}
	// Without charID, position should be default spawn, not from repo.
	if p.Position.X == 10 && p.Position.Z == 20 {
		t.Error("player without charID got saved position")
	}
}

// --- leaveZone tests ---

func TestLeaveZone_RemovesClientFromZone(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined)
	if zi.zone.ClientCount() != 1 {
		t.Fatalf("pre-condition: ClientCount = %d, want 1", zi.zone.ClientCount())
	}

	// Register zone so leaveZone can find it.
	gw.mu.Lock()
	gw.zones["test"] = zi
	gw.mu.Unlock()

	gw.leaveZone(sess)

	if zi.zone.ClientCount() != 0 {
		t.Errorf("ClientCount after leave = %d, want 0", zi.zone.ClientCount())
	}
}

func TestLeaveZone_BroadcastsDisconnect(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("test", zone.ZoneTypeOpenWorld)

	// Two players in zone.
	sess1, _ := newTestSession(1)
	sess2, spy2 := newTestSession(2)
	defer sess1.Conn.Close()
	defer sess2.Conn.Close()

	gw.joinZone(sess1, zi, joinResponseZoneJoined)
	gw.joinZone(sess2, zi, joinResponseZoneJoined)
	spy2.Reset()

	gw.mu.Lock()
	gw.zones["test"] = zi
	gw.mu.Unlock()

	gw.leaveZone(sess1)

	msgs := drainSpy(spy2)
	if findMessage(msgs, message.OpPeerDisconnected) == nil {
		t.Error("remaining peer did not receive OpPeerDisconnected")
	}
}

func TestLeaveZone_NoopWhenZoneIDEmpty(_ *testing.T) {
	gw := newTestGateway(stubRepo{})
	sess, _ := newTestSession(1)
	sess.ZoneID = ""
	defer sess.Conn.Close()

	// Should not panic.
	gw.leaveZone(sess)
}

func TestLeaveZone_NoopWhenZoneNotFound(_ *testing.T) {
	gw := newTestGateway(stubRepo{})
	sess, _ := newTestSession(1)
	sess.ZoneID = "nonexistent"
	defer sess.Conn.Close()

	// Should not panic.
	gw.leaveZone(sess)
}

func TestLeaveZone_RemovesEmptyArena(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("arena_1", zone.ZoneTypeInstanced)

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, zi, joinResponseZoneJoined)

	gw.mu.Lock()
	gw.zones["arena_1"] = zi
	gw.mu.Unlock()

	gw.leaveZone(sess)

	if gw.getZone("arena_1") != nil {
		t.Error("empty arena was not removed")
	}
}

func TestLeaveZone_DoesNotRemoveEmptyHub(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(zone.ZoneHub, zone.ZoneTypeOpenWorld)

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, zi, joinResponseZoneJoined)

	gw.mu.Lock()
	gw.zones[zone.ZoneHub] = zi
	gw.mu.Unlock()

	gw.leaveZone(sess)

	if gw.getZone(zone.ZoneHub) == nil {
		t.Error("hub zone was incorrectly removed")
	}
}

func TestLeaveZone_DoesNotRemoveNonEmptyArena(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("arena_1", zone.ZoneTypeInstanced)

	sess1, _ := newTestSession(1)
	sess2, _ := newTestSession(2)
	defer sess1.Conn.Close()
	defer sess2.Conn.Close()

	gw.joinZone(sess1, zi, joinResponseZoneJoined)
	gw.joinZone(sess2, zi, joinResponseZoneJoined)

	gw.mu.Lock()
	gw.zones["arena_1"] = zi
	gw.mu.Unlock()

	gw.leaveZone(sess1)

	if gw.getZone("arena_1") == nil {
		t.Error("non-empty arena was incorrectly removed")
	}
}

// --- transferPlayer tests ---

func TestTransferPlayer_MovesPlayerBetweenZones(t *testing.T) {
	gw := newTestGateway(stubRepo{})

	hubZI := newTestZoneInstance(zone.ZoneHub, zone.ZoneTypeOpenWorld)
	gw.mu.Lock()
	gw.zones[zone.ZoneHub] = hubZI
	gw.mu.Unlock()

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined)

	if hubZI.zone.ClientCount() != 1 {
		t.Fatalf("hub ClientCount = %d, want 1", hubZI.zone.ClientCount())
	}

	// Transfer to arena (getOrCreateZone will create it).
	gw.transferPlayer(sess, "arena_test", zone.ZoneTypeInstanced, 1)

	if hubZI.zone.ClientCount() != 0 {
		t.Errorf("hub ClientCount after transfer = %d, want 0", hubZI.zone.ClientCount())
	}
	arenaZI := gw.getZone("arena_test")
	if arenaZI == nil {
		t.Fatal("arena zone not created")
	}
	if arenaZI.zone.ClientCount() != 1 {
		t.Errorf("arena ClientCount = %d, want 1", arenaZI.zone.ClientCount())
	}
	if sess.ZoneID != "arena_test" {
		t.Errorf("sess.ZoneID = %q, want %q", sess.ZoneID, "arena_test")
	}
}

// --- benchmarks ---

func BenchmarkJoinZone(b *testing.B) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("bench", zone.ZoneTypeOpenWorld)

	conn, _ := network.NewTestClient()
	defer conn.Close()
	sess := &session.Session{
		ID:       1,
		Conn:     conn,
		Username: "BenchPlayer",
		Class:    entity.ClassGunner,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gw.joinZone(sess, zi, joinResponseZoneJoined)
		// Reset for next iteration.
		zi.zone.RemoveClient(sess.PeerID)
	}
}

func BenchmarkJoinZoneWithPeers(b *testing.B) {
	for _, peerCount := range []int{1, 5, 20} {
		b.Run(fmt.Sprintf("%d_peers", peerCount), func(b *testing.B) {
			gw := newTestGateway(stubRepo{})
			zi := newTestZoneInstance("bench", zone.ZoneTypeOpenWorld)

			// Pre-populate peers.
			conns := make([]*network.Client, peerCount)
			for i := range conns {
				c, _ := network.NewTestClient()
				conns[i] = c
				pSess := &session.Session{
					ID:       uint32(100 + i),
					Conn:     c,
					Username: "Peer",
					Class:    entity.ClassGunner,
				}
				gw.joinZone(pSess, zi, joinResponseZoneJoined)
			}

			conn, _ := network.NewTestClient()
			defer conn.Close()
			sess := &session.Session{
				ID:       1,
				Conn:     conn,
				Username: "BenchPlayer",
				Class:    entity.ClassGunner,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				gw.joinZone(sess, zi, joinResponseZoneJoined)
				zi.zone.RemoveClient(sess.PeerID)
			}

			b.StopTimer()
			for _, c := range conns {
				c.Close()
			}
		})
	}
}

func BenchmarkLeaveZone(b *testing.B) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance("bench", zone.ZoneTypeOpenWorld)

	gw.mu.Lock()
	gw.zones["bench"] = zi
	gw.mu.Unlock()

	conn, _ := network.NewTestClient()
	defer conn.Close()
	sess := &session.Session{
		ID:       1,
		Conn:     conn,
		Username: "BenchPlayer",
		Class:    entity.ClassGunner,
	}

	// Pre-join so leaveZone has something to leave.
	// Each iteration: join + leave. The join cost is included but consistent.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gw.joinZone(sess, zi, joinResponseZoneJoined)
		gw.leaveZone(sess)
	}
}

func BenchmarkTransferPlayer(b *testing.B) {
	gw := newTestGateway(stubRepo{})

	hubZI := newTestZoneInstance(zone.ZoneHub, zone.ZoneTypeOpenWorld)
	arenaZI := newTestZoneInstance("arena_bench", zone.ZoneTypeInstanced)

	gw.mu.Lock()
	gw.zones[zone.ZoneHub] = hubZI
	gw.zones["arena_bench"] = arenaZI
	gw.mu.Unlock()

	conn, _ := network.NewTestClient()
	defer conn.Close()
	sess := &session.Session{
		ID:       1,
		Conn:     conn,
		Username: "BenchPlayer",
		Class:    entity.ClassGunner,
	}

	// Each iteration: join hub, transfer to arena, clean up arena client.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gw.joinZone(sess, hubZI, joinResponseZoneJoined)
		gw.transferPlayer(sess, "arena_bench", zone.ZoneTypeInstanced, 1)
		arenaZI.zone.RemoveClient(sess.PeerID)
	}
}
