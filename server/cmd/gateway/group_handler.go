package main

import (
	"encoding/binary"
	"fmt"
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/group"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/overflux"
	"codex-online/server/internal/session"
	"codex-online/server/internal/zone"
)

// defaultPortalTarget is the fallback zone when a portal has no TargetZone set.
const defaultPortalTarget = "arena"

// handleGroupMessage processes all group-related opcodes (0x0050–0x005F).
func (g *gateway) handleGroupMessage(sess *session.Session, opcode uint16, payload []byte) { //nolint:funlen // switch dispatch
	switch opcode {
	case message.OpGroupCreate:
		grp, err := g.groups.CreateGroup(sess.ID)
		if err != nil {
			sendGroupError(sess.Conn, err.Error())
			return
		}
		slog.Info("group created", "group_id", grp.ID, "leader", sess.ID)
		g.broadcastGroupState(grp)

	case message.OpGroupInvite:
		if len(payload) < 2 {
			return
		}
		targetPeerID := binary.LittleEndian.Uint16(payload[0:2])
		targetGlobalID := g.sessions.ResolveZonePeer(sess.ZoneID, targetPeerID)
		if targetGlobalID == 0 {
			sendGroupError(sess.Conn, "player not found")
			return
		}
		invite, err := g.groups.InvitePlayer(sess.ID, targetGlobalID)
		if err != nil {
			sendGroupError(sess.Conn, err.Error())
			return
		}
		targetSess := g.sessions.GetByID(targetGlobalID)
		if targetSess != nil {
			sendGroupInviteRecv(targetSess.Conn, invite.GroupID, sess.Username)
		}
		slog.Info("group invite sent", "from", sess.ID, "to", targetGlobalID, "group", invite.GroupID)

	case message.OpGroupInviteByName:
		g.handleGroupInviteByName(sess, payload)

	case message.OpGroupInviteReply:
		g.handleGroupInviteReply(sess, payload)

	case message.OpGroupLeave:
		grp, disbanded := g.groups.LeaveGroup(sess.ID)
		slog.Info("player left group", "player", sess.ID, "disbanded", disbanded)
		if !disbanded && grp != nil {
			g.broadcastGroupState(grp)
		}
		sendEmptyGroupState(sess.Conn)

	case message.OpGroupKick:
		g.handleGroupKick(sess, payload)

	case message.OpEnterPortal:
		g.handleEnterPortal(sess, payload)

	case message.OpInstanceJoinReply:
		g.handleInstanceJoinReply(sess, payload)

	case message.OpInstanceReset:
		g.handleInstanceReset(sess)

	default:
		slog.Warn("unknown group opcode", "opcode", opcode)
	}
}

// handleEnterPortal resolves the portal target for the player's current zone and
// transfers them to the appropriate zone. For instanced zones, it either creates
// a new instance (with overflux conditions from payload) or joins an existing
// group instance. Group members in open-world zones receive a join prompt.
func (g *gateway) handleEnterPortal(sess *session.Session, payload []byte) { //nolint:funlen // portal flow has create+join+notify paths
	// Determine target zone from portal definitions, fallback to "arena".
	targetZone := defaultPortalTarget
	g.mu.Lock()
	if zi, ok := g.zones[sess.ZoneID]; ok {
		portals := zi.zone.Portals()
		if len(portals) > 0 {
			targetZone = portals[0].TargetZone
		}
	}
	g.mu.Unlock()

	lvl, err := g.loadLevel(targetZone)
	if err != nil {
		slog.Error("load level for portal target", "zone", targetZone, "error", err)
		return
	}

	// Open-world zones are shared; use the zone name directly.
	if lvl.ZoneType != "instanced" {
		slog.Info("player entering portal to open-world zone", "player_id", sess.ID, "target", targetZone)
		g.transferPlayer(sess, targetZone, lvl, 0, nil)
		return
	}

	// Instanced zones get a unique instance ID per group or solo player.
	grp := g.groups.GetGroup(sess.ID)
	var instanceID string
	groupSize := 1
	if grp != nil {
		instanceID = fmt.Sprintf("%s_g%d", targetZone, grp.ID)
		groupSize = len(grp.Members)
	} else {
		instanceID = fmt.Sprintf("%s_s%d", targetZone, sess.ID)
	}

	// If the instance already exists and still has live players, join it
	// directly (no new conditions). An instance with zero clients is an
	// abandoned run (e.g. a previously cleared dungeon whose leave-time
	// teardown was missed): tear it down so a fresh instance spawns below,
	// rather than dropping the player back into an empty, already-cleared zone.
	if existing := g.getZone(instanceID); existing != nil {
		if existing.zone.ClientCount() > 0 {
			slog.Info("player joining existing instance", "player_id", sess.ID, "instance", instanceID)
			g.transferPlayer(sess, instanceID, lvl, groupSize, nil)
			if grp != nil {
				g.broadcastGroupState(grp)
			}
			return
		}
		slog.Info("removing abandoned empty instance before re-entry", "player_id", sess.ID, "instance", instanceID)
		g.removeZone(instanceID)
	}

	// New instance: decode overflux conditions from payload.
	conditions, decErr := overflux.DecodeConditions(payload)
	if decErr != nil {
		slog.Warn("bad overflux payload", "player_id", sess.ID, "error", decErr)
	}
	oflx := overflux.NewState(conditions)

	slog.Info("player creating instance", "player_id", sess.ID, "target_zone", targetZone, "instance", instanceID, "group_size", groupSize, "overflux", oflx.TotalScore)
	g.transferPlayer(sess, instanceID, lvl, groupSize, oflx)

	// Notify group members in open-world zones about the new instance.
	if grp != nil {
		g.broadcastGroupState(grp)
		promptPayload := overflux.EncodeJoinPrompt(targetZone, sess.Username, oflx)
		promptMsg := message.Encode(message.OpInstanceJoinPrompt, 0, promptPayload)
		for _, memberID := range grp.Members {
			if memberID == sess.ID {
				continue
			}
			ms := g.sessions.GetByID(memberID)
			if ms == nil {
				continue
			}
			// Only prompt members in open-world zones (not already in an instance).
			if ms.ZoneType != uint8(zone.ZoneTypeOpenWorld) {
				continue
			}
			ms.Conn.Send(promptMsg)
		}
	}
}

// handleInstanceJoinReply processes a group member's accept/decline of an
// instance join prompt.
func (g *gateway) handleInstanceJoinReply(sess *session.Session, payload []byte) {
	if len(payload) < 1 {
		return
	}
	accept := payload[0] == 1
	if !accept {
		slog.Info("instance join declined", "player_id", sess.ID)
		return
	}

	grp := g.groups.GetGroup(sess.ID)
	if grp == nil {
		sendGroupError(sess.Conn, "not in a group")
		return
	}

	// Resolve the portal target from the player's current zone.
	targetZone := defaultPortalTarget
	g.mu.Lock()
	if zi, ok := g.zones[sess.ZoneID]; ok {
		portals := zi.zone.Portals()
		if len(portals) > 0 {
			targetZone = portals[0].TargetZone
		}
	}
	g.mu.Unlock()

	instanceID := fmt.Sprintf("%s_g%d", targetZone, grp.ID)
	if g.getZone(instanceID) == nil {
		sendGroupError(sess.Conn, "instance no longer exists")
		return
	}

	lvl, err := g.loadLevel(targetZone)
	if err != nil {
		slog.Error("load level for instance join", "zone", targetZone, "error", err)
		return
	}

	slog.Info("player joining instance via prompt", "player_id", sess.ID, "instance", instanceID)
	g.transferPlayer(sess, instanceID, lvl, len(grp.Members), nil)
	g.broadcastGroupState(grp)
}

// handleInstanceReset allows the group leader to destroy the group's current
// instance so a fresh one can be created with new overflux conditions.
func (g *gateway) handleInstanceReset(sess *session.Session) {
	grp := g.groups.GetGroup(sess.ID)
	if grp == nil {
		sendGroupError(sess.Conn, "not in a group")
		return
	}
	if grp.LeaderID != sess.ID {
		sendGroupError(sess.Conn, "only the leader can reset")
		return
	}

	// Find and destroy group instances. Convention: {zone}_g{groupID}.
	prefix := fmt.Sprintf("_g%d", grp.ID)
	g.mu.Lock()
	var toRemove []string
	for zoneID, zi := range g.zones {
		if zi.zone.Type != zone.ZoneTypeInstanced {
			continue
		}
		if len(zoneID) > len(prefix) && zoneID[len(zoneID)-len(prefix):] == prefix {
			toRemove = append(toRemove, zoneID)
		}
	}
	g.mu.Unlock()

	for _, zoneID := range toRemove {
		zi := g.getZone(zoneID)
		if zi == nil {
			continue
		}
		// Transfer any players still in the instance back to hub.
		for _, peerID := range zi.zone.GetPeerIDs() {
			globalID := g.sessions.ResolveZonePeer(zoneID, peerID)
			if globalID == 0 {
				continue
			}
			peerSess := g.sessions.GetByID(globalID)
			if peerSess == nil {
				continue
			}
			hubLvl, err := g.loadLevel(defaultOpenWorldZone)
			if err != nil {
				slog.Error("load hub for instance reset", "error", err)
				continue
			}
			g.transferPlayer(peerSess, defaultOpenWorldZone, hubLvl, 0, nil)
		}
		g.removeZone(zoneID)
		slog.Info("instance reset", "zone_id", zoneID, "leader", sess.ID)
	}
	g.broadcastGroupState(grp)
}

// handleGroupInviteByName invites a player resolved by account or character name,
// reaching any online player regardless of zone. Payload: [type:u8][name:str8]
// where type 0=account username, 1=character name.
func (g *gateway) handleGroupInviteByName(sess *session.Session, payload []byte) {
	if len(payload) < 2 {
		return
	}
	nameType := payload[0]
	nameLen := int(payload[1])
	if len(payload) < 2+nameLen {
		return
	}
	name := string(payload[2 : 2+nameLen])

	var target *session.Session
	if nameType == 1 {
		target = g.sessions.FindOnlineByCharName(name)
	} else {
		target = g.sessions.FindOnlineByUsername(name)
	}
	if target == nil {
		sendGroupError(sess.Conn, "player not found or offline")
		return
	}
	if target.ID == sess.ID {
		sendGroupError(sess.Conn, "you cannot invite yourself")
		return
	}

	invite, err := g.groups.InvitePlayer(sess.ID, target.ID)
	if err != nil {
		sendGroupError(sess.Conn, err.Error())
		return
	}
	sendGroupInviteRecv(target.Conn, invite.GroupID, sess.Username)
	slog.Info("group invite by name sent", "from", sess.ID, "to", target.ID, "group", invite.GroupID)
}

// handleGroupInviteReply processes an accept or decline for a pending group invite.
func (g *gateway) handleGroupInviteReply(sess *session.Session, payload []byte) {
	if len(payload) < 5 {
		return
	}
	groupID := binary.LittleEndian.Uint32(payload[0:4])
	accept := payload[4] == 1
	if accept {
		grp, err := g.groups.AcceptInvite(sess.ID, groupID)
		if err != nil {
			sendGroupError(sess.Conn, err.Error())
			return
		}
		slog.Info("group invite accepted", "player", sess.ID, "group", groupID)
		g.broadcastGroupState(grp)
	} else {
		g.groups.DeclineInvite(sess.ID, groupID)
		slog.Info("group invite declined", "player", sess.ID, "group", groupID)
	}
}

// handleGroupKick removes a targeted player from the group on behalf of the leader.
func (g *gateway) handleGroupKick(sess *session.Session, payload []byte) {
	if len(payload) < 2 {
		return
	}
	targetPeerID := binary.LittleEndian.Uint16(payload[0:2])
	targetGlobalID := g.sessions.ResolveZonePeer(sess.ZoneID, targetPeerID)
	if targetGlobalID == 0 {
		sendGroupError(sess.Conn, "player not found")
		return
	}
	grp, err := g.groups.KickPlayer(sess.ID, targetGlobalID)
	if err != nil {
		sendGroupError(sess.Conn, err.Error())
		return
	}
	slog.Info("player kicked from group", "leader", sess.ID, "target", targetGlobalID)
	g.broadcastGroupState(grp)
	targetSess := g.sessions.GetByID(targetGlobalID)
	if targetSess != nil {
		sendEmptyGroupState(targetSess.Conn)
	}
}

// broadcastGroupState sends OpGroupState to all members of a group.
func (g *gateway) broadcastGroupState(grp *group.Group) {
	buf := encodeGroupState(g, grp)
	msg := message.Encode(message.OpGroupState, 0, buf)
	for _, memberID := range grp.Members {
		sess := g.sessions.GetByID(memberID)
		if sess != nil {
			sess.Conn.Send(msg)
		}
	}
}

func encodeGroupState(gw *gateway, grp *group.Group) []byte {
	leaderSess := gw.sessions.GetByID(grp.LeaderID)
	leaderPeer := uint16(0)
	if leaderSess != nil {
		leaderPeer = leaderSess.PeerID
	}
	members := make([]codec.GroupMemberInfo, len(grp.Members))
	for i, memberID := range grp.Members {
		sess := gw.sessions.GetByID(memberID)
		if sess != nil {
			members[i] = codec.GroupMemberInfo{PeerID: sess.PeerID, Username: sess.Username}
		}
	}
	return codec.EncodeGroupState(grp.ID, leaderPeer, members)
}

func sendGroupError(client *network.Client, errMsg string) {
	client.Send(message.Encode(message.OpGroupError, 0, codec.EncodeGroupError(errMsg)))
}

func sendGroupInviteRecv(client *network.Client, groupID uint32, leaderName string) {
	client.Send(message.Encode(message.OpGroupInviteRecv, 0, codec.EncodeGroupInviteRecv(groupID, leaderName)))
}

func sendEmptyGroupState(client *network.Client) {
	client.Send(message.Encode(message.OpGroupState, 0, codec.EncodeEmptyGroupState()))
}
