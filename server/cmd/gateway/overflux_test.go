package main

import (
	"encoding/binary"
	"fmt"
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

// encodeConditionsPayload builds the OpEnterPortal wire payload for a set of
// overflux conditions. Mirrors the DecodeConditions wire format:
// [count:u8][per: id_len:u8 + id:bytes + rank:u8]
func encodeConditionsPayload(conditions []overflux.ActiveCondition) []byte {
	size := 1
	for _, c := range conditions {
		size += 1 + len(c.ID) + 1
	}
	buf := make([]byte, size)
	buf[0] = byte(len(conditions))
	off := 1
	for _, c := range conditions {
		buf[off] = byte(len(c.ID))
		off++
		copy(buf[off:], c.ID)
		off += len(c.ID)
		buf[off] = byte(c.Rank)
		off++
	}
	return buf
}

// registerSession registers a client connection in the gateway's session
// registry and returns the session (with auto-assigned ID) and its spy.
// Use this for tests that need GetByID lookups (group prompt broadcasts, etc.).
func registerSession(gw *gateway, username string) (*session.Session, *network.TestSpy) {
	conn, spy := network.NewTestClient()
	sess := gw.sessions.Register(conn)
	sess.Username = username
	sess.Class = entity.ClassGunner
	return sess, spy
}

// setupPortalGateway creates a gateway with a hub zone and a live UDP server
// ready for portal-enter tests. The hub zone is stored in gw.zones["hub"].
func setupPortalGateway(t *testing.T) (*gateway, *zoneInstance) {
	t.Helper()
	gw := newTestGateway(stubRepo{})

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

	return gw, hubZI
}

// --- TestEnterPortal_WithOverfluxConditions ---

// TestEnterPortal_WithOverfluxConditions verifies that when a player sends
// OpEnterPortal with a conditions payload the created zone stores the overflux
// state and the player receives OpZoneTransfer for an instanced zone.
func TestEnterPortal_WithOverfluxConditions(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
	spy.Reset()

	conditions := []overflux.ActiveCondition{
		{ID: overflux.CondEnemyHP, Rank: 3},
	}
	payload := encodeConditionsPayload(conditions)
	wantScore := overflux.ComputeScore(conditions) // 4 * 3 = 12

	gw.handleEnterPortal(sess, payload)

	msgs := drainSpy(spy)

	// Player must receive OpZoneTransfer.
	raw := findMessage(msgs, message.OpZoneTransfer)
	if raw == nil {
		t.Fatal("OpZoneTransfer not received")
	}
	_, _, xferPayload, _ := message.Decode(raw)
	if len(xferPayload) < 1 {
		t.Fatalf("OpZoneTransfer payload too short: %d", len(xferPayload))
	}
	if xferPayload[0] != byte(zone.ZoneTypeInstanced) {
		t.Errorf("zone type = %d, want %d (instanced)", xferPayload[0], zone.ZoneTypeInstanced)
	}

	// The created instance must carry the overflux state.
	instanceID := fmt.Sprintf("arena_s%d", sess.ID)
	gw.mu.Lock()
	zi := gw.zones[instanceID]
	gw.mu.Unlock()
	if zi == nil {
		t.Fatalf("instance zone %q not found", instanceID)
	}
	oflx := zi.zone.OverfluxState()
	if oflx == nil {
		t.Fatal("OverfluxState is nil on created instance")
	}
	if oflx.TotalScore != wantScore {
		t.Errorf("TotalScore = %d, want %d", oflx.TotalScore, wantScore)
	}
	if len(oflx.Conditions) != 1 {
		t.Fatalf("len(Conditions) = %d, want 1", len(oflx.Conditions))
	}
	if oflx.Conditions[0].ID != overflux.CondEnemyHP {
		t.Errorf("Conditions[0].ID = %q, want %q", oflx.Conditions[0].ID, overflux.CondEnemyHP)
	}
	if oflx.Conditions[0].Rank != 3 {
		t.Errorf("Conditions[0].Rank = %d, want 3", oflx.Conditions[0].Rank)
	}
}

// --- TestEnterPortal_JoinsExistingInstance ---

// TestEnterPortal_JoinsExistingInstance verifies that a player whose group
// already has an active instance joins it rather than creating a new zone.
func TestEnterPortal_JoinsExistingInstance(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	// Use registered sessions so group broadcasts via GetByID work.
	leaderSess, leaderSpy := registerSession(gw, "Leader")
	defer leaderSess.Conn.Close()
	gw.joinZone(leaderSess, hubZI, joinResponseZoneJoined, "")

	memberSess, memberSpy := registerSession(gw, "Member")
	defer memberSess.Conn.Close()
	gw.joinZone(memberSess, hubZI, joinResponseZoneJoined, "")

	// Build a group: leader invites member, member accepts.
	grp, err := gw.groups.CreateGroup(leaderSess.ID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	invite, err := gw.groups.InvitePlayer(leaderSess.ID, memberSess.ID)
	if err != nil {
		t.Fatalf("InvitePlayer: %v", err)
	}
	if _, err := gw.groups.AcceptInvite(memberSess.ID, invite.GroupID); err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}

	leaderSpy.Reset()
	memberSpy.Reset()

	// Leader enters portal — creates a new instance.
	gw.handleEnterPortal(leaderSess, nil)
	drainSpy(leaderSpy)

	// Count zones before the second enter.
	gw.mu.Lock()
	zonesBefore := len(gw.zones)
	gw.mu.Unlock()

	// Member enters portal — must join the existing instance, not create one.
	gw.handleEnterPortal(memberSess, nil)
	msgs := drainSpy(memberSpy)

	if findMessage(msgs, message.OpZoneTransfer) == nil {
		t.Fatal("member did not receive OpZoneTransfer")
	}

	gw.mu.Lock()
	zonesAfter := len(gw.zones)
	gw.mu.Unlock()

	if zonesAfter != zonesBefore {
		t.Errorf("zone count changed from %d to %d; expected no new zone", zonesBefore, zonesAfter)
	}

	expectedInstanceID := fmt.Sprintf("arena_g%d", grp.ID)
	if memberSess.ZoneID != expectedInstanceID {
		t.Errorf("member ZoneID = %q, want %q", memberSess.ZoneID, expectedInstanceID)
	}
}

// --- TestEnterPortal_GroupMembersReceiveJoinPrompt ---

// TestEnterPortal_GroupMembersReceiveJoinPrompt verifies that when the group
// leader enters a portal, hub-dwelling members receive OpInstanceJoinPrompt
// while members already in an instanced zone do not.
func TestEnterPortal_GroupMembersReceiveJoinPrompt(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	// All sessions must be registered so GetByID works in the broadcast loop.
	leaderSess, leaderSpy := registerSession(gw, "Leader")
	leaderSess.Username = "Leader"
	defer leaderSess.Conn.Close()
	gw.joinZone(leaderSess, hubZI, joinResponseZoneJoined, "")

	// Hub member — should receive the prompt.
	hubMemberSess, hubMemberSpy := registerSession(gw, "HubMember")
	defer hubMemberSess.Conn.Close()
	gw.joinZone(hubMemberSess, hubZI, joinResponseZoneJoined, "")

	// Create an arena zone and place a third member inside it.
	arenaZI := newTestZoneInstance(t, "arena_pre", "arena")
	gw.mu.Lock()
	gw.zones["arena_pre"] = arenaZI
	gw.mu.Unlock()
	instancedMemberSess, instancedMemberSpy := registerSession(gw, "InstancedMember")
	defer instancedMemberSess.Conn.Close()
	gw.joinZone(instancedMemberSess, arenaZI, joinResponseZoneJoined, "")

	// Build group with all three members.
	if _, err := gw.groups.CreateGroup(leaderSess.ID); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	invite2, err := gw.groups.InvitePlayer(leaderSess.ID, hubMemberSess.ID)
	if err != nil {
		t.Fatalf("InvitePlayer hub member: %v", err)
	}
	if _, err := gw.groups.AcceptInvite(hubMemberSess.ID, invite2.GroupID); err != nil {
		t.Fatalf("AcceptInvite hub member: %v", err)
	}
	invite3, err := gw.groups.InvitePlayer(leaderSess.ID, instancedMemberSess.ID)
	if err != nil {
		t.Fatalf("InvitePlayer instanced member: %v", err)
	}
	if _, err := gw.groups.AcceptInvite(instancedMemberSess.ID, invite3.GroupID); err != nil {
		t.Fatalf("AcceptInvite instanced member: %v", err)
	}

	leaderSpy.Reset()
	hubMemberSpy.Reset()
	instancedMemberSpy.Reset()

	conditions := []overflux.ActiveCondition{
		{ID: overflux.CondEnemyHP, Rank: 2},
	}
	payload := encodeConditionsPayload(conditions)
	gw.handleEnterPortal(leaderSess, payload)
	drainSpy(leaderSpy)

	hubMemberMsgs := drainSpy(hubMemberSpy)
	instancedMemberMsgs := drainSpy(instancedMemberSpy)

	// Hub member must receive OpInstanceJoinPrompt.
	promptRaw := findMessage(hubMemberMsgs, message.OpInstanceJoinPrompt)
	if promptRaw == nil {
		t.Fatal("hub member did not receive OpInstanceJoinPrompt")
	}

	// Verify prompt payload: [zone:str8][leader:str8][score:u16 LE][count:u8]...
	_, _, promptPayload, _ := message.Decode(promptRaw)
	if len(promptPayload) < 1 {
		t.Fatal("prompt payload empty")
	}
	// Decode zone name.
	off := 0
	zoneNameLen := int(promptPayload[off])
	off++
	if off+zoneNameLen > len(promptPayload) {
		t.Fatal("prompt payload truncated at zone name")
	}
	zoneName := string(promptPayload[off : off+zoneNameLen])
	off += zoneNameLen
	if zoneName != "arena" {
		t.Errorf("prompt zone name = %q, want %q", zoneName, "arena")
	}
	// Decode leader name.
	if off >= len(promptPayload) {
		t.Fatal("prompt payload truncated at leader name length")
	}
	leaderNameLen := int(promptPayload[off])
	off++
	if off+leaderNameLen > len(promptPayload) {
		t.Fatal("prompt payload truncated at leader name")
	}
	leaderName := string(promptPayload[off : off+leaderNameLen])
	off += leaderNameLen
	if leaderName != leaderSess.Username {
		t.Errorf("prompt leader name = %q, want %q", leaderName, leaderSess.Username)
	}
	// Decode score.
	if off+2 > len(promptPayload) {
		t.Fatal("prompt payload truncated at score")
	}
	score := int(binary.LittleEndian.Uint16(promptPayload[off : off+2]))
	wantScore := overflux.ComputeScore(conditions)
	if score != wantScore {
		t.Errorf("prompt score = %d, want %d", score, wantScore)
	}

	// Member already in an instance must NOT receive OpInstanceJoinPrompt.
	if findMessage(instancedMemberMsgs, message.OpInstanceJoinPrompt) != nil {
		t.Error("instanced member should not receive OpInstanceJoinPrompt")
	}
}

// --- TestInstanceJoinReply_Accept ---

// TestInstanceJoinReply_Accept verifies that a member who accepts a join prompt
// receives OpZoneTransfer to the group's instance.
func TestInstanceJoinReply_Accept(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	leaderSess, leaderSpy := registerSession(gw, "Leader")
	defer leaderSess.Conn.Close()
	gw.joinZone(leaderSess, hubZI, joinResponseZoneJoined, "")

	memberSess, memberSpy := registerSession(gw, "Member")
	defer memberSess.Conn.Close()
	gw.joinZone(memberSess, hubZI, joinResponseZoneJoined, "")

	grp, err := gw.groups.CreateGroup(leaderSess.ID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	invite, err := gw.groups.InvitePlayer(leaderSess.ID, memberSess.ID)
	if err != nil {
		t.Fatalf("InvitePlayer: %v", err)
	}
	if _, err := gw.groups.AcceptInvite(memberSess.ID, invite.GroupID); err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}

	// Leader creates the instance.
	leaderSpy.Reset()
	gw.handleEnterPortal(leaderSess, nil)
	drainSpy(leaderSpy)

	memberSpy.Reset()

	// Member sends accept reply [accept:1].
	gw.handleInstanceJoinReply(memberSess, []byte{1})
	msgs := drainSpy(memberSpy)

	xfer := findMessage(msgs, message.OpZoneTransfer)
	if xfer == nil {
		t.Fatal("member did not receive OpZoneTransfer after accepting join prompt")
	}

	expectedInstanceID := fmt.Sprintf("arena_g%d", grp.ID)
	if memberSess.ZoneID != expectedInstanceID {
		t.Errorf("member ZoneID = %q, want %q", memberSess.ZoneID, expectedInstanceID)
	}
}

// --- TestInstanceJoinReply_Decline ---

// TestInstanceJoinReply_Decline verifies that a member who declines the join
// prompt stays in the hub and does not receive OpZoneTransfer.
func TestInstanceJoinReply_Decline(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	leaderSess, leaderSpy := registerSession(gw, "Leader")
	defer leaderSess.Conn.Close()
	gw.joinZone(leaderSess, hubZI, joinResponseZoneJoined, "")

	memberSess, memberSpy := registerSession(gw, "Member")
	defer memberSess.Conn.Close()
	gw.joinZone(memberSess, hubZI, joinResponseZoneJoined, "")

	if _, err := gw.groups.CreateGroup(leaderSess.ID); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	invite, err := gw.groups.InvitePlayer(leaderSess.ID, memberSess.ID)
	if err != nil {
		t.Fatalf("InvitePlayer: %v", err)
	}
	if _, err := gw.groups.AcceptInvite(memberSess.ID, invite.GroupID); err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}

	leaderSpy.Reset()
	gw.handleEnterPortal(leaderSess, nil)
	drainSpy(leaderSpy)

	memberSpy.Reset()

	// Member sends decline reply [accept:0].
	gw.handleInstanceJoinReply(memberSess, []byte{0})
	msgs := drainSpy(memberSpy)

	if findMessage(msgs, message.OpZoneTransfer) != nil {
		t.Error("member should not receive OpZoneTransfer after declining join prompt")
	}
	if memberSess.ZoneID != defaultOpenWorldZone {
		t.Errorf("member ZoneID = %q, want %q (should remain in hub)", memberSess.ZoneID, defaultOpenWorldZone)
	}
}

// --- TestInstanceReset_LeaderOnly ---

// TestInstanceReset_LeaderOnly verifies that only the group leader can reset the
// instance. A non-leader receives an error and the zone is unchanged. When the
// leader resets, the instance zone is removed and players inside it are
// transferred back to hub.
func TestInstanceReset_LeaderOnly(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	// Registered sessions are required so that ResolveZonePeer + GetByID work
	// inside handleInstanceReset's player-transfer loop.
	leaderSess, leaderSpy := registerSession(gw, "Leader")
	defer leaderSess.Conn.Close()
	gw.joinZone(leaderSess, hubZI, joinResponseZoneJoined, "")

	memberSess, memberSpy := registerSession(gw, "Member")
	defer memberSess.Conn.Close()
	gw.joinZone(memberSess, hubZI, joinResponseZoneJoined, "")

	grp, err := gw.groups.CreateGroup(leaderSess.ID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	invite, err := gw.groups.InvitePlayer(leaderSess.ID, memberSess.ID)
	if err != nil {
		t.Fatalf("InvitePlayer: %v", err)
	}
	if _, err := gw.groups.AcceptInvite(memberSess.ID, invite.GroupID); err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}

	// Both leader and member enter the instance.
	leaderSpy.Reset()
	gw.handleEnterPortal(leaderSess, nil)
	drainSpy(leaderSpy)

	gw.handleEnterPortal(memberSess, nil)
	drainSpy(memberSpy)

	instanceID := fmt.Sprintf("arena_g%d", grp.ID)
	if gw.getZone(instanceID) == nil {
		t.Fatalf("instance zone %q not found before reset", instanceID)
	}

	// Non-leader tries to reset — must receive error, zone must remain.
	memberSpy.Reset()
	gw.handleInstanceReset(memberSess)
	memberMsgs := drainSpy(memberSpy)
	if findMessage(memberMsgs, message.OpGroupError) == nil {
		t.Error("non-leader should receive OpGroupError when attempting reset")
	}
	if gw.getZone(instanceID) == nil {
		t.Error("instance zone was removed by non-leader — should not happen")
	}

	// Leader resets — instance zone must be removed and players transferred to hub.
	leaderSpy.Reset()
	memberSpy.Reset()
	gw.handleInstanceReset(leaderSess)
	drainSpy(leaderSpy)
	drainSpy(memberSpy)

	if gw.getZone(instanceID) != nil {
		t.Errorf("instance zone %q still exists after leader reset", instanceID)
	}
	if leaderSess.ZoneID != defaultOpenWorldZone {
		t.Errorf("leader ZoneID = %q, want %q after reset", leaderSess.ZoneID, defaultOpenWorldZone)
	}
	if memberSess.ZoneID != defaultOpenWorldZone {
		t.Errorf("member ZoneID = %q, want %q after reset", memberSess.ZoneID, defaultOpenWorldZone)
	}
}

// --- TestOverfluxState_BroadcastOnJoin ---

// TestOverfluxState_BroadcastOnJoin verifies that when a player joins an
// instanced zone that carries overflux conditions they receive OpOverfluxState
// with the correct score and condition data.
func TestOverfluxState_BroadcastOnJoin(t *testing.T) {
	gw, hubZI := setupPortalGateway(t)

	sess, spy := newTestSession(1)
	defer sess.Conn.Close()
	gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
	spy.Reset()

	conditions := []overflux.ActiveCondition{
		{ID: overflux.CondEnemyHP, Rank: 5},
	}
	payload := encodeConditionsPayload(conditions)
	wantScore := overflux.ComputeScore(conditions) // 4 * 5 = 20

	gw.handleEnterPortal(sess, payload)
	msgs := drainSpy(spy)

	oflxRaw := findMessage(msgs, message.OpOverfluxState)
	if oflxRaw == nil {
		t.Fatal("OpOverfluxState not received on join to overflux instance")
	}

	_, _, oflxPayload, _ := message.Decode(oflxRaw)
	// Wire: [score:u16 LE][count:u8][per: id_len:u8 + id:bytes + rank:u8]
	if len(oflxPayload) < 3 {
		t.Fatalf("OpOverfluxState payload too short: %d bytes", len(oflxPayload))
	}
	score := int(binary.LittleEndian.Uint16(oflxPayload[0:2]))
	if score != wantScore {
		t.Errorf("OpOverfluxState score = %d, want %d", score, wantScore)
	}
	count := int(oflxPayload[2])
	if count != 1 {
		t.Errorf("OpOverfluxState condition count = %d, want 1", count)
	}
	off := 3
	if off >= len(oflxPayload) {
		t.Fatal("OpOverfluxState payload truncated before id_len")
	}
	idLen := int(oflxPayload[off])
	off++
	if off+idLen+1 > len(oflxPayload) {
		t.Fatalf("OpOverfluxState payload truncated (off=%d idLen=%d len=%d)", off, idLen, len(oflxPayload))
	}
	id := overflux.ConditionID(oflxPayload[off : off+idLen])
	off += idLen
	rank := int(oflxPayload[off])
	if id != overflux.CondEnemyHP {
		t.Errorf("condition ID = %q, want %q", id, overflux.CondEnemyHP)
	}
	if rank != 5 {
		t.Errorf("condition rank = %d, want 5", rank)
	}
}

// --- TestEnterPortal_EmptyPayload_BackwardCompat ---

// TestEnterPortal_EmptyPayload_BackwardCompat verifies that nil or empty portal
// payload is accepted (backward-compatible path) and results in a zone with
// zero overflux score.
func TestEnterPortal_EmptyPayload_BackwardCompat(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"nil payload", nil},
		{"empty slice", []byte{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw, hubZI := setupPortalGateway(t)

			sess, spy := newTestSession(1)
			defer sess.Conn.Close()
			gw.joinZone(sess, hubZI, joinResponseZoneJoined, "")
			spy.Reset()

			gw.handleEnterPortal(sess, tc.payload)
			msgs := drainSpy(spy)

			if findMessage(msgs, message.OpZoneTransfer) == nil {
				t.Fatal("OpZoneTransfer not received with empty payload")
			}

			instanceID := fmt.Sprintf("arena_s%d", sess.ID)
			gw.mu.Lock()
			zi := gw.zones[instanceID]
			gw.mu.Unlock()
			if zi == nil {
				t.Fatalf("instance zone %q not found", instanceID)
			}

			oflx := zi.zone.OverfluxState()
			// An empty/nil conditions list produces a zero-score state (not nil).
			if oflx != nil && oflx.TotalScore != 0 {
				t.Errorf("TotalScore = %d, want 0 for empty payload", oflx.TotalScore)
			}
		})
	}
}
