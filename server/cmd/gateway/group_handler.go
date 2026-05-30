package main

import (
	"encoding/binary"
	"fmt"
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/group"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/session"
)

// handleGroupMessage processes all group-related opcodes (0x0050–0x005F).
func (g *gateway) handleGroupMessage(sess *session.Session, opcode uint16, payload []byte) {
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
		g.handleEnterPortal(sess)

	default:
		slog.Warn("unknown group opcode", "opcode", opcode)
	}
}

// handleEnterPortal resolves the portal target for the player's current zone and
// transfers them to the appropriate zone.
func (g *gateway) handleEnterPortal(sess *session.Session) {
	// Determine target zone from portal definitions, fallback to "arena"
	targetZone := "arena"
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
		g.transferPlayer(sess, targetZone, lvl, 0)
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
	slog.Info("player entering portal", "player_id", sess.ID, "target_zone", targetZone, "instance", instanceID, "group_size", groupSize)
	g.transferPlayer(sess, instanceID, lvl, groupSize)
	if grp != nil {
		g.broadcastGroupState(grp)
	}
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
