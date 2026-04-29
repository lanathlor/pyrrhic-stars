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
	"codex-online/server/internal/zone"
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

	case message.OpGroupLeave:
		grp, disbanded := g.groups.LeaveGroup(sess.ID)
		slog.Info("player left group", "player", sess.ID, "disbanded", disbanded)
		if !disbanded && grp != nil {
			g.broadcastGroupState(grp)
		}
		sendEmptyGroupState(sess.Conn)

	case message.OpGroupKick:
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

	case message.OpEnterPortal:
		if sess.ZoneID != "hub" {
			sendGroupError(sess.Conn, "can only enter portal from hub")
			return
		}
		grp := g.groups.GetGroup(sess.ID)
		var arenaID string
		if grp != nil {
			arenaID = fmt.Sprintf("arena_g%d", grp.ID)
		} else {
			arenaID = fmt.Sprintf("arena_s%d", sess.ID)
		}
		slog.Info("player entering portal", "player_id", sess.ID, "arena", arenaID)
		g.transferPlayer(sess, arenaID, zone.ZoneTypeArena)
		if grp != nil {
			g.broadcastGroupState(grp)
		}

	default:
		slog.Warn("unknown group opcode", "opcode", opcode)
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
