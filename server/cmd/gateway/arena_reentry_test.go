package main

import (
	"fmt"
	"testing"
)

// TestEnterPortal_AbandonedInstanceReplacedWithFresh verifies that when a
// player re-enters a portal whose deterministic instance still exists but holds
// no live clients (e.g. a previously cleared run that was abandoned), the stale
// instance is torn down and a brand-new instance is created. Otherwise the
// player rejoins their already-cleared dungeon: graphically empty, all enemies
// dead. See gateway.leaveZone for the leave-time teardown this guards against
// failing.
func TestEnterPortal_AbandonedInstanceReplacedWithFresh(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	sess, _ := registerSession(gw, "Solo")
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")

	// First entry: creates the solo arena instance.
	gw.handleEnterPortal(sess, nil)

	instanceID := fmt.Sprintf("arena_s%d", sess.ID)
	gw.mu.Lock()
	oldZI := gw.zones[instanceID]
	gw.mu.Unlock()
	if oldZI == nil {
		t.Fatalf("first entry did not create instance %q", instanceID)
	}

	// Simulate a cleared run that was abandoned without the zone being torn
	// down: the client leaves the arena (ClientCount drops to 0) but the
	// zone lingers in the registry. Then the player walks back to the hub.
	oldZI.zone.RemoveClient(sess.PeerID)
	if oldZI.zone.ClientCount() != 0 {
		t.Fatalf("pre-condition: abandoned instance ClientCount = %d, want 0", oldZI.zone.ClientCount())
	}
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")

	// Re-entry: must NOT rejoin the stale empty instance.
	gw.handleEnterPortal(sess, nil)

	gw.mu.Lock()
	newZI := gw.zones[instanceID]
	gw.mu.Unlock()
	if newZI == nil {
		t.Fatalf("re-entry produced no instance %q", instanceID)
	}
	if newZI == oldZI {
		t.Error("re-entry rejoined the abandoned empty instance; expected a fresh one")
	}
	if newZI.zone.ClientCount() != 1 {
		t.Errorf("fresh instance ClientCount = %d, want 1", newZI.zone.ClientCount())
	}
	if sess.ZoneID != instanceID {
		t.Errorf("player ZoneID = %q, want %q", sess.ZoneID, instanceID)
	}
}
