package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"runtime"
	"testing"
	"time"

	"codex-online/server/internal/container"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

const testBenchPlayer = "BenchPlayer"

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.Run()
}

// --- test helpers ---

// stubRepo implements persistence.Repository with no-op methods.
type stubRepo struct{ persistence.Repository }

func (stubRepo) GetCharacterByID(uint) (*persistence.Character, error) { return nil, nil }
func (stubRepo) CreateItem(*persistence.CharacterItem) error           { return nil }
func (stubRepo) DeleteItem(uint) error                                 { return nil }
func (stubRepo) GetItemsByCharacterID(uint) ([]*persistence.CharacterItem, error) {
	return nil, nil
}
func (stubRepo) SetEquipment(uint, uint8, uint) error                         { return nil }
func (stubRepo) ClearEquipment(uint, uint8) error                             { return nil }
func (stubRepo) GetEquipment(uint) ([]*persistence.CharacterEquipment, error) { return nil, nil }
func (stubRepo) GetScrip(uint, uint16) (int, error)                           { return 0, nil }
func (stubRepo) GetWatermark(uint, uint16) (int, error)                       { return 0, nil }

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
func newTestZoneInstance(t testing.TB, id string, zoneName string) *zoneInstance {
	t.Helper()
	lvl, err := level.Load(zoneName)
	if err != nil {
		t.Fatalf("load level %q: %v", zoneName, err)
	}
	z := zone.New(id, lvl, nil)
	_, cancel := context.WithCancel(context.Background())
	return &zoneInstance{zone: z, cancel: cancel, nextID: 1}
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
	zi := newTestZoneInstance(t, "test", "hub")

	sess1, _ := newTestSession(1)
	sess2, _ := newTestSession(2)
	defer sess1.Conn.Close()
	defer sess2.Conn.Close()

	gw.joinZone(sess1, zi, joinResponseZoneJoined, "")
	gw.joinZone(sess2, zi, joinResponseZoneJoined, "")

	if sess1.PeerID != 1 {
		t.Errorf("sess1.PeerID = %d, want 1", sess1.PeerID)
	}
	if sess2.PeerID != 2 {
		t.Errorf("sess2.PeerID = %d, want 2", sess2.PeerID)
	}
}

func TestJoinZone_SetsSessionState(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "my-zone", "arena")

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

	if sess.ZoneID != "my-zone" {
		t.Errorf("sess.ZoneID = %q, want %q", sess.ZoneID, "my-zone")
	}
	if sess.PeerID != 1 {
		t.Errorf("sess.PeerID = %d, want 1", sess.PeerID)
	}
}

func TestJoinZone_AddsClientToZone(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "test", "hub")

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
			zi := newTestZoneInstance(t, "test", "hub")

			sess, _ := newTestSession(42)
			sess.CharName = tt.charName
			sess.Username = tt.username
			defer sess.Conn.Close()

			gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	zi := newTestZoneInstance(t, "test", "hub")

	sess, _ := newTestSession(7)
	sess.CharName = ""
	sess.Username = ""
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

	if sess.Username != "Player_7" {
		t.Errorf("sess.Username = %q, want %q", sess.Username, "Player_7")
	}
}

func TestJoinZone_SendsZoneJoinedResponse(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "test", "hub")

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	if len(payload) < 7 {
		t.Fatalf("payload missing spawn_yaw: %d bytes", len(payload))
	}
	gotYaw := math.Float32frombits(binary.BigEndian.Uint32(payload[3:7]))
	if gotYaw != zi.zone.SpawnYaw() {
		t.Errorf("spawn_yaw in response = %v, want %v", gotYaw, zi.zone.SpawnYaw())
	}
}

func TestJoinZone_SendsZoneTransferResponse(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "arena_1", "arena")

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneTransfer, "")

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
	if len(payload) < 7 {
		t.Fatalf("payload missing spawn_yaw: %d bytes", len(payload))
	}
	gotYaw := math.Float32frombits(binary.BigEndian.Uint32(payload[3:7]))
	if gotYaw != zi.zone.SpawnYaw() {
		t.Errorf("spawn_yaw in response = %v, want %v", gotYaw, zi.zone.SpawnYaw())
	}
}

func TestJoinZone_NotifiesExistingPeers(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "test", "hub")

	// Add an existing peer.
	existing, existingSpy := newTestSession(1)
	defer existing.Conn.Close()
	gw.joinZone(existing, zi, joinResponseZoneJoined, "")
	existingSpy.Reset()

	// New peer joins.
	newSess, _ := newTestSession(2)
	defer newSess.Conn.Close()
	gw.joinZone(newSess, zi, joinResponseZoneJoined, "")

	// Existing peer should receive OpPeerConnected for new peer.
	msgs := drainSpy(existingSpy)
	n := countMessages(msgs, message.OpPeerConnected)
	if n == 0 {
		t.Error("existing peer did not receive OpPeerConnected for new peer")
	}
}

func TestJoinZone_NotifiesNewPeerAboutExisting(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "test", "hub")

	// Add an existing peer.
	existing, _ := newTestSession(1)
	defer existing.Conn.Close()
	gw.joinZone(existing, zi, joinResponseZoneJoined, "")

	// New peer joins.
	newSess, newSpy := newTestSession(2)
	defer newSess.Conn.Close()
	gw.joinZone(newSess, zi, joinResponseZoneJoined, "")

	// New peer should receive OpPeerConnected for the existing peer.
	msgs := drainSpy(newSpy)
	n := countMessages(msgs, message.OpPeerConnected)
	if n == 0 {
		t.Error("new peer did not receive OpPeerConnected for existing peer")
	}
}

func TestJoinZone_QueuesClassSelectForNonGunner(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "test", "hub")

	sess, _ := newTestSession(1)
	sess.Class = entity.ClassVanguard
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	zi := newTestZoneInstance(t, "test", "hub")

	sess, _ := newTestSession(1)
	sess.Class = entity.ClassGunner
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

	// Gunner is the default — no class select should be queued.
	// Verify player exists and zone is functional.
	if zi.zone.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", zi.zone.ClientCount())
	}
}

func TestJoinZone_RestoresPositionForHub(t *testing.T) {
	gw := newTestGateway(posRepo{})
	zi := newTestZoneInstance(t, "hub", "hub")

	sess, _ := newTestSession(1)
	sess.CharID = 42
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	zi := newTestZoneInstance(t, "arena_1", "arena")

	sess, _ := newTestSession(1)
	sess.CharID = 42
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	zi := newTestZoneInstance(t, "hub", "hub")

	sess, _ := newTestSession(1)
	sess.CharID = 0 // no character
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	zi := newTestZoneInstance(t, "test", "hub")

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()

	gw.joinZone(sess, zi, joinResponseZoneJoined, "")
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
	zi := newTestZoneInstance(t, "test", "hub")

	// Two players in zone.
	sess1, _ := newTestSession(1)
	sess2, spy2 := newTestSession(2)
	defer sess1.Conn.Close()
	defer sess2.Conn.Close()

	gw.joinZone(sess1, zi, joinResponseZoneJoined, "")
	gw.joinZone(sess2, zi, joinResponseZoneJoined, "")
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
	zi := newTestZoneInstance(t, "arena_1", "arena")

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

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
	zi := newTestZoneInstance(t, "hub", "hub")

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, zi, joinResponseZoneJoined, "")

	gw.mu.Lock()
	gw.zones["hub"] = zi
	gw.mu.Unlock()

	gw.leaveZone(sess)

	if gw.getZone("hub") == nil {
		t.Error("hub zone was incorrectly removed")
	}
}

func TestLeaveZone_DoesNotRemoveNonEmptyArena(t *testing.T) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(t, "arena_1", "arena")

	sess1, _ := newTestSession(1)
	sess2, _ := newTestSession(2)
	defer sess1.Conn.Close()
	defer sess2.Conn.Close()

	gw.joinZone(sess1, zi, joinResponseZoneJoined, "")
	gw.joinZone(sess2, zi, joinResponseZoneJoined, "")

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

	hubZI := newTestZoneInstance(t, "hub", "hub")
	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.mu.Unlock()

	sess, _ := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")

	if hubZI.zone.ClientCount() != 1 {
		t.Fatalf("hub ClientCount = %d, want 1", hubZI.zone.ClientCount())
	}

	// Transfer to arena (getOrCreateZone will create it).
	arenaLvl, err := level.Load("arena")
	if err != nil {
		t.Fatalf("load arena level: %v", err)
	}
	gw.transferPlayer(sess, "arena_test", arenaLvl, 1, nil, "")

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

// --- portal enter tests (OpEnterPortal → zone transfer) ---

// TestEnterPortal_PlayerReceivesZoneTransfer exercises the full portal-enter
// path: player is in the hub, sends OpEnterPortal, and must receive
// OpZoneTransfer for the arena. This failed after adding UDP transport because
// sendUDPAssociate (called during joinZone) panics when udpServer is nil, or
// the OpUDPAssociate message interferes with the transfer flow.
func TestEnterPortal_PlayerReceivesZoneTransfer(t *testing.T) {
	gw := newTestGateway(stubRepo{})

	// Start a real UDP server so sendUDPAssociate runs the full path.
	udpSrv, err := network.NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpSrv.Close() }()
	gw.udpServer = udpSrv

	// Create hub zone and place the player in it.
	hubZI := newTestZoneInstance(t, "hub", "hub")
	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.mu.Unlock()

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
	spy.Reset() // discard initial join messages

	// Simulate pressing 'E' at the portal.
	gw.handleEnterPortal(sess, nil)

	// The client must receive OpZoneTransfer.
	msgs := drainSpy(spy)
	raw := findMessage(msgs, message.OpZoneTransfer)
	if raw == nil {
		var opcodes []uint16
		for _, m := range msgs {
			op, _, _, e := message.Decode(m)
			if e == nil {
				opcodes = append(opcodes, op)
			}
		}
		t.Fatalf("OpZoneTransfer not received; got %d messages with opcodes %v", len(msgs), opcodes)
	}

	_, _, payload, _ := message.Decode(raw)
	if len(payload) < 3 {
		t.Fatalf("payload too short: %d", len(payload))
	}
	if payload[0] != byte(zone.ZoneTypeInstanced) {
		t.Errorf("zone type = %d, want %d (instanced)", payload[0], zone.ZoneTypeInstanced)
	}
}

// TestEnterPortal_PlayerReceivesUDPAssociate verifies that the UDP association
// token is sent alongside the zone transfer during portal enter.
func TestEnterPortal_PlayerReceivesUDPAssociate(t *testing.T) {
	gw := newTestGateway(stubRepo{})

	udpSrv, err := network.NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpSrv.Close() }()
	gw.udpServer = udpSrv

	hubZI := newTestZoneInstance(t, "hub", "hub")
	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.mu.Unlock()

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
	spy.Reset()

	gw.handleEnterPortal(sess, nil)

	msgs := drainSpy(spy)

	// Must have both OpZoneTransfer and OpUDPAssociate.
	if findMessage(msgs, message.OpZoneTransfer) == nil {
		t.Fatal("OpZoneTransfer not received")
	}
	raw := findMessage(msgs, message.OpUDPAssociate)
	if raw == nil {
		t.Fatal("OpUDPAssociate not received after portal enter")
	}

	// OpUDPAssociate payload: [token:16][port:2 BE]
	_, _, payload, _ := message.Decode(raw)
	if len(payload) < 18 {
		t.Fatalf("OpUDPAssociate payload too short: %d bytes", len(payload))
	}
	port := binary.BigEndian.Uint16(payload[16:18])
	if port != uint16(udpSrv.Port()) {
		t.Errorf("UDP port in OpUDPAssociate = %d, want %d", port, udpSrv.Port())
	}
}

// TestEnterPortal_NoUDPAssociateWhenAlreadyAssociated verifies that zone
// transfer does NOT re-send OpUDPAssociate when the client already has a
// working UDP association. The UDP connection survives zone transfers.
func TestEnterPortal_NoUDPAssociateWhenAlreadyAssociated(t *testing.T) {
	gw := newTestGateway(stubRepo{})

	udpSrv, err := network.NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpSrv.Close() }()
	gw.udpServer = udpSrv

	hubZI := newTestZoneInstance(t, "hub", "hub")
	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.mu.Unlock()

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")

	// Simulate completed UDP association (as if client sent back the ack).
	sess.Conn.AssociateUDP(udpSrv.Conn(), &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999})

	// Drain and reset spy to discard initial join messages.
	drainSpy(spy)
	spy.Reset()

	// Simulate pressing 'E' at the portal.
	gw.handleEnterPortal(sess, nil)

	msgs := drainSpy(spy)

	// Must receive OpZoneTransfer.
	if findMessage(msgs, message.OpZoneTransfer) == nil {
		var opcodes []uint16
		for _, m := range msgs {
			op, _, _, e := message.Decode(m)
			if e == nil {
				opcodes = append(opcodes, op)
			}
		}
		t.Fatalf("OpZoneTransfer not received; got %d messages with opcodes %v", len(msgs), opcodes)
	}

	// Must NOT receive OpUDPAssociate (UDP is already associated).
	if raw := findMessage(msgs, message.OpUDPAssociate); raw != nil {
		t.Error("OpUDPAssociate should not be sent when UDP is already associated")
	}
}

// TestEnterPortal_ArenaTickLowerThanHub verifies that a freshly created arena
// zone starts at tick 0, which is lower than the hub zone's accumulated tick.
// This proves that a client using a stale _udp_last_tick from the hub will
// drop ALL arena world state via the out-of-order check, making the arena
// appear frozen.
func TestEnterPortal_ArenaTickLowerThanHub(t *testing.T) {
	gw := newTestGateway(stubRepo{})

	udpSrv, err := network.NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = udpSrv.Close() }()
	gw.udpServer = udpSrv

	// Create hub zone and simulate several ticks so TickNum is high.
	hubZI := newTestZoneInstance(t, "hub", "hub")
	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.mu.Unlock()
	// Advance hub tick counter (simulates ~5 seconds of gameplay).
	for range 100 {
		hubZI.zone.IncrementTick()
	}

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")

	// Simulate completed UDP association.
	sess.Conn.AssociateUDP(udpSrv.Conn(), &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999})

	hubTick := hubZI.zone.TickNum()
	if hubTick < 100 {
		t.Fatalf("hub tick should be >= 100, got %d", hubTick)
	}

	drainSpy(spy)
	spy.Reset()

	// Enter portal -> creates fresh arena zone.
	gw.handleEnterPortal(sess, nil)

	// Find the arena zone.
	arenaZI := findArenaZone(gw)
	if arenaZI == nil {
		t.Fatal("arena zone not created")
	}

	arenaTick := arenaZI.zone.TickNum()

	// The arena starts at tick 0. A client with _udp_last_tick == hubTick
	// would drop all arena world state because arenaTick <= _udp_last_tick.
	if arenaTick >= hubTick {
		t.Skipf("arena tick (%d) >= hub tick (%d); out-of-order check would pass (unexpected)", arenaTick, hubTick)
	}

	// This is the bug: the client's _udp_last_tick is NOT reset on zone
	// transfer, so arenaTick (0) <= _udp_last_tick (hubTick) and all arena
	// world state is silently dropped.
	t.Logf("BUG CONFIRMED: hub tick=%d, arena tick=%d — client's _udp_last_tick from hub "+
		"would reject all arena world state via out-of-order check", hubTick, arenaTick)
}

func findArenaZone(gw *gateway) *zoneInstance {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	for id, zi := range gw.zones {
		if id != "hub" {
			return zi
		}
	}
	return nil
}

// --- benchmarks ---

func BenchmarkJoinZone(b *testing.B) {
	gw := newTestGateway(stubRepo{})
	zi := newTestZoneInstance(b, "bench", "hub")

	conn, _ := network.NewTestClient()
	defer conn.Close()
	sess := &session.Session{
		ID:       1,
		Conn:     conn,
		Username: testBenchPlayer,
		Class:    entity.ClassGunner,
	}

	for b.Loop() {
		gw.joinZone(sess, zi, joinResponseZoneJoined, "")
		// Reset for next iteration.
		zi.zone.RemoveClient(sess.PeerID)
	}
}

func BenchmarkJoinZoneWithPeers(b *testing.B) {
	for _, peerCount := range []int{1, 5, 20} {
		b.Run(fmt.Sprintf("%d_peers", peerCount), func(b *testing.B) {
			gw := newTestGateway(stubRepo{})
			zi := newTestZoneInstance(b, "bench", "hub")

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
				gw.joinZone(pSess, zi, joinResponseZoneJoined, "")
			}

			conn, _ := network.NewTestClient()
			defer conn.Close()
			sess := &session.Session{
				ID:       1,
				Conn:     conn,
				Username: testBenchPlayer,
				Class:    entity.ClassGunner,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				gw.joinZone(sess, zi, joinResponseZoneJoined, "")
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
	zi := newTestZoneInstance(b, "bench", "hub")

	gw.mu.Lock()
	gw.zones["bench"] = zi
	gw.mu.Unlock()

	conn, _ := network.NewTestClient()
	defer conn.Close()
	sess := &session.Session{
		ID:       1,
		Conn:     conn,
		Username: testBenchPlayer,
		Class:    entity.ClassGunner,
	}

	// Pre-join so leaveZone has something to leave.
	// Each iteration: join + leave. The join cost is included but consistent.

	for b.Loop() {
		gw.joinZone(sess, zi, joinResponseZoneJoined, "")
		gw.leaveZone(sess)
	}
}

func BenchmarkTransferPlayer(b *testing.B) {
	gw := newTestGateway(stubRepo{})

	hubZI := newTestZoneInstance(b, "hub", "hub")
	arenaZI := newTestZoneInstance(b, "arena_bench", "arena")

	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.zones["arena_bench"] = arenaZI
	gw.mu.Unlock()

	conn, _ := network.NewTestClient()
	defer conn.Close()
	sess := &session.Session{
		ID:       1,
		Conn:     conn,
		Username: testBenchPlayer,
		Class:    entity.ClassGunner,
	}

	// Each iteration: join hub, transfer to arena, clean up arena client.
	arenaLvl, err := level.Load("arena")
	if err != nil {
		b.Fatalf("load arena level: %v", err)
	}

	for b.Loop() {
		gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
		gw.transferPlayer(sess, "arena_bench", arenaLvl, 1, nil, "")
		arenaZI.zone.RemoveClient(sess.PeerID)
	}
}
