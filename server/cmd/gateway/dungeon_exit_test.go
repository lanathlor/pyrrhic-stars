package main

import (
	"fmt"
	"math"
	"testing"

	"codex-online/server/internal/container"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
)

// savedPosRepo returns a fixed saved character position so tests can prove that
// returning from a dungeon overrides the last-saved position with the entrance.
type savedPosRepo struct {
	stubRepo
	char *persistence.Character
}

func (r savedPosRepo) GetCharacterByID(uint) (*persistence.Character, error) {
	return r.char, nil
}

// setupDungeonExitGateway builds a gateway whose hub is the real hub level, with
// a character whose saved open-world position is far from the arena portal so a
// failure to apply the entrance is detectable. Returns the gateway, hub zone,
// the session (already in the arena instance), and that instance's ID.
func setupDungeonExitGateway(t *testing.T) (*gateway, *zoneInstance, *session.Session, string) {
	t.Helper()
	// Saved position far from the hub arena portal (33, 102, 5.5).
	saved := &persistence.Character{ID: 7, PosX: -100, PosY: 0.02, PosZ: -63.55, RotY: 1.0}
	gw := newGateway(container.New(savedPosRepo{char: saved}))

	udpSrv, err := network.NewUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = udpSrv.Close() })
	gw.udpServer = udpSrv

	hubZI := newTestZoneInstance(t, "hub", defaultOpenWorldZone)
	gw.mu.Lock()
	gw.zones["hub"] = hubZI
	gw.mu.Unlock()

	sess, _ := registerSession(gw, "Diver")
	t.Cleanup(func() { sess.Conn.Close() })
	sess.CharID = 7
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")

	// Enter the portal: creates and transfers into the solo arena instance.
	gw.handleEnterPortal(sess, nil)
	instanceID := fmt.Sprintf("arena_s%d", sess.ID)
	if sess.ZoneID != instanceID {
		t.Fatalf("after portal, ZoneID = %q, want %q", sess.ZoneID, instanceID)
	}
	return gw, hubZI, sess, instanceID
}

// assertAtHubEntrance checks the player spawned on the hub arena-portal pad and
// not at the seeded saved position.
func assertAtHubEntrance(t *testing.T, hubZI *zoneInstance, peerID uint16) {
	t.Helper()
	p := hubZI.zone.GetPlayer(peerID)
	if p == nil {
		t.Fatal("player not found in hub after return")
	}
	// Hub arena portal lives at (33, 102, 5.5); the player spawns on the pad.
	const portalX, portalY, portalZ = 33.0, 102.0, 5.5
	dx := float64(p.Position.X) - portalX
	dy := float64(p.Position.Y) - portalY
	dz := float64(p.Position.Z) - portalZ
	if d := math.Sqrt(dx*dx + dy*dy + dz*dz); d > 0.5 {
		t.Errorf("spawned at (%.2f, %.2f, %.2f), want portal (%.1f, %.1f, %.1f) (dist=%.2f)",
			p.Position.X, p.Position.Y, p.Position.Z, portalX, portalY, portalZ, d)
	}
}

// TestReturnFromDungeon_PortalExit_SpawnsAtHubEntrance covers the common path:
// the player walks into the arena's exit portal (target "hub"), which routes
// through handleEnterPortal.
func TestReturnFromDungeon_PortalExit_SpawnsAtHubEntrance(t *testing.T) {
	gw, hubZI, sess, _ := setupDungeonExitGateway(t)

	// Walk into the arena's exit portal back to the hub.
	gw.handleEnterPortal(sess, nil)
	if sess.ZoneID != defaultOpenWorldZone {
		t.Fatalf("after exit portal, ZoneID = %q, want %q", sess.ZoneID, defaultOpenWorldZone)
	}

	assertAtHubEntrance(t, hubZI, sess.PeerID)
}

// TestReturnFromDungeon_BossExit_SpawnsAtHubEntrance covers the boss-defeated /
// respawn path, which routes through handlePlayerReturnToOpenWorld.
func TestReturnFromDungeon_BossExit_SpawnsAtHubEntrance(t *testing.T) {
	gw, hubZI, sess, instanceID := setupDungeonExitGateway(t)

	gw.handlePlayerReturnToOpenWorld(instanceID, sess.PeerID)
	if sess.ZoneID != defaultOpenWorldZone {
		t.Fatalf("after return, ZoneID = %q, want %q", sess.ZoneID, defaultOpenWorldZone)
	}

	assertAtHubEntrance(t, hubZI, sess.PeerID)
}
