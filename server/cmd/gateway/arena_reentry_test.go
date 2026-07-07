package main

import (
	"fmt"
	"testing"

	"codex-online/server/internal/message"
)

// TestEnterPortal_MemberRejoinsRunAfterDisconnect covers the playtest bug: a
// group member disconnects mid-run (which removes them from the group, so the
// deterministic arena_g{id} no longer resolves for them), reconnects with the
// same character, and walks into the portal. They must rejoin the run their
// character is a member of, not spawn a fresh solo instance while their
// partner is still fighting inside.
func TestEnterPortal_MemberRejoinsRunAfterDisconnect(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	leader, _ := registerSession(gw, "Leader")
	defer leader.Conn.Close()
	leader.CharID = 11
	member, _ := registerSession(gw, "Member")
	member.CharID = 22
	gw.joinZone(leader, hubZI, joinResponseZoneJoined, "")
	gw.joinZone(member, hubZI, joinResponseZoneJoined, "")

	grp, err := gw.groups.CreateGroup(leader.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.groups.InvitePlayer(leader.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.groups.AcceptInvite(member.ID, grp.ID); err != nil {
		t.Fatal(err)
	}

	// Both enter the run.
	gw.handleEnterPortal(leader, nil)
	gw.handleEnterPortal(member, nil)
	instanceID := fmt.Sprintf("arena_g%d", grp.ID)
	if member.ZoneID != instanceID {
		t.Fatalf("setup: member ZoneID = %q, want %q", member.ZoneID, instanceID)
	}

	// Member disconnects mid-fight: connection cleanup removes group
	// membership and leaves the zone. The leader keeps fighting inside.
	gw.groups.LeaveGroup(member.ID)
	gw.leaveZone(member)
	member.Conn.Close()

	// Reconnect: a brand-new session (new session ID, no group), same character.
	member2, _ := registerSession(gw, "Member")
	defer member2.Conn.Close()
	member2.CharID = 22
	gw.joinZone(member2, hubZI, joinResponseZoneJoined, "")

	gw.handleEnterPortal(member2, nil)

	if member2.ZoneID != instanceID {
		t.Errorf("after reconnect, ZoneID = %q, want %q (must rejoin the ongoing run)", member2.ZoneID, instanceID)
	}
	zi := gw.getZone(instanceID)
	if zi == nil {
		t.Fatal("ongoing run vanished")
	}
	if n := zi.zone.ClientCount(); n != 2 {
		t.Errorf("run ClientCount = %d, want 2 (leader + rejoined member)", n)
	}
}

// TestReturnToHub_MidRun_SendsRejoinPrompt covers the death-screen path: a
// player who returns to the hub while their run is ongoing must receive
// OpInstanceJoinPrompt again. The client's portal join-vs-create decision
// rests on that prompt (pending_instance_zone is wiped on arena entry), so
// without a re-prompt the portal only offers the create-instance panel.
func TestReturnToHub_MidRun_SendsRejoinPrompt(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	sess, spy := registerSession(gw, "Diver")
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
	gw.handleEnterPortal(sess, nil)
	instanceID := fmt.Sprintf("arena_s%d", sess.ID)
	if sess.ZoneID != instanceID {
		t.Fatalf("setup: ZoneID = %q, want %q", sess.ZoneID, instanceID)
	}
	spy.Reset()

	// Death screen "return to hub" routes through handlePlayerReturnToOpenWorld.
	gw.handlePlayerReturnToOpenWorld(instanceID, sess.PeerID)
	if sess.ZoneID != defaultOpenWorldZone {
		t.Fatalf("after return, ZoneID = %q, want %q", sess.ZoneID, defaultOpenWorldZone)
	}

	msgs := drainSpy(spy)
	raw := findMessage(msgs, message.OpInstanceJoinPrompt)
	if raw == nil {
		t.Fatal("no OpInstanceJoinPrompt after returning to hub mid-run")
	}
	_, _, payload, _ := message.Decode(raw)
	if len(payload) < 1 || int(payload[0]) > len(payload)-1 {
		t.Fatalf("malformed prompt payload: %v", payload)
	}
	if zoneName := string(payload[1 : 1+payload[0]]); zoneName != "arena" {
		t.Errorf("prompt zone = %q, want %q", zoneName, "arena")
	}
}

// TestInstanceJoinReply_RejoinsRunWithoutGroup verifies that accepting a
// rejoin prompt works via run membership even when the player has no group
// (solo runs, or a group that churned while they were away).
func TestInstanceJoinReply_RejoinsRunWithoutGroup(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	sess, _ := registerSession(gw, "Diver")
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
	gw.handleEnterPortal(sess, nil)
	instanceID := fmt.Sprintf("arena_s%d", sess.ID)

	gw.handlePlayerReturnToOpenWorld(instanceID, sess.PeerID)

	// Accept the rejoin prompt.
	gw.handleInstanceJoinReply(sess, []byte{1})

	if sess.ZoneID != instanceID {
		t.Errorf("after accept, ZoneID = %q, want %q", sess.ZoneID, instanceID)
	}
}

// TestEnterPortal_AbandonedInstanceReplacedWithFresh verifies that when a
// player re-enters a portal whose deterministic instance still exists but holds
// no live clients (e.g. a previously cleared run that was abandoned), the stale
// instance is torn down and a brand-new instance is created. Otherwise the
// player rejoins their already-cleared dungeon: graphically empty, all enemies
// dead. See gateway.leaveZone for the leave-time teardown this guards against
// failing.
// TestEnterPortal_OngoingInstanceRejoinedAfterHubReturn covers the playtest
// bug: a player who returns to the hub mid-run (portal exit by accident) must
// be able to walk back into the portal and rejoin the ongoing instance, not
// have it destroyed and replaced with a fresh one.
func TestEnterPortal_OngoingInstanceRejoinedAfterHubReturn(t *testing.T) {
	gw, _, sess, instanceID := setupDungeonExitGateway(t)

	oldZI := gw.getZone(instanceID)
	if oldZI == nil {
		t.Fatalf("setup: instance %q not found", instanceID)
	}
	if oldZI.zone.RunCompleted() {
		t.Fatal("setup: fresh instance should not read as completed")
	}

	// Accidental exit: walk into the arena's exit portal back to the hub while
	// the run is still ongoing (boss alive).
	gw.handleEnterPortal(sess, nil)
	if sess.ZoneID != defaultOpenWorldZone {
		t.Fatalf("after exit portal, ZoneID = %q, want %q", sess.ZoneID, defaultOpenWorldZone)
	}

	if gw.getZone(instanceID) == nil {
		t.Fatal("ongoing instance was destroyed when its last player returned to hub")
	}

	// Re-entering the portal must rejoin the same run, not create a fresh one.
	gw.handleEnterPortal(sess, nil)
	newZI := gw.getZone(instanceID)
	if newZI == nil {
		t.Fatalf("re-entry produced no instance %q", instanceID)
	}
	if newZI != oldZI {
		t.Error("portal re-entry created a fresh instance instead of rejoining the ongoing run")
	}
	if sess.ZoneID != instanceID {
		t.Errorf("player ZoneID = %q, want %q", sess.ZoneID, instanceID)
	}
	if newZI.zone.ClientCount() != 1 {
		t.Errorf("rejoined instance ClientCount = %d, want 1", newZI.zone.ClientCount())
	}
}

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
	// down: the boss is dead, the client leaves the arena (ClientCount drops
	// to 0) but the zone lingers in the registry. Then the player walks back
	// to the hub.
	oldZI.zone.MarkRunCompleted()
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
