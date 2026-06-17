package main

import (
	"errors"
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/friend"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/session"
)

// handleFriendMessage dispatches friend-related opcodes (0x00F0–0x00FF).
func (g *gateway) handleFriendMessage(sess *session.Session, opcode uint16, payload []byte) {
	// Friend operations require an authenticated account.
	if sess.UserUUID == "" {
		return
	}
	switch opcode {
	case message.OpFriendRequest:
		g.handleFriendRequest(sess, payload)
	case message.OpFriendRespond:
		g.handleFriendRespond(sess, payload)
	case message.OpFriendRemove:
		g.handleFriendRemove(sess, payload)
	case message.OpFriendListReq:
		g.sendFriendList(sess)
	default:
		slog.Warn("unknown friend opcode", "opcode", opcode)
	}
}

// handleFriendRequest sends a friend request resolved by account or character
// name. Payload: [type:u8][name:str8] where type 0=account, 1=character.
func (g *gateway) handleFriendRequest(sess *session.Session, payload []byte) {
	if len(payload) < 2 {
		return
	}
	nameType := payload[0]
	name, ok := readStr8Payload(payload, 1)
	if !ok {
		return
	}

	targetID, autoAccepted, err := g.friends.Request(sess.UserUUID, nameType, name)
	if err != nil {
		sendFriendError(sess.Conn, friendErrorMessage(err))
		return
	}

	if autoAccepted {
		// Mutual cross-request: both are now friends, refresh both lists.
		g.sendFriendList(sess)
		g.refreshFriendList(targetID)
		slog.Info("friend request auto-accepted", "from", sess.UserUUID, "to", targetID)
		return
	}

	// Pending request: deliver live to the target if online, and refresh the
	// requester's own list is unnecessary (no new friend yet).
	if ts := g.sessions.FindOnlineByUserUUID(targetID); ts != nil {
		sendFriendRequestRecv(ts.Conn, sess.UserUUID, resolveDisplayName(sess))
	}
	slog.Info("friend request sent", "from", sess.UserUUID, "to", targetID)
}

// handleFriendRespond accepts or declines a pending request.
// Payload: [accept:u8][requesterUserID:str8].
func (g *gateway) handleFriendRespond(sess *session.Session, payload []byte) {
	if len(payload) < 2 {
		return
	}
	accept := payload[0] == 1
	requesterID, ok := readStr8Payload(payload, 1)
	if !ok {
		return
	}

	if err := g.friends.Respond(sess.UserUUID, requesterID, accept); err != nil {
		sendFriendError(sess.Conn, friendErrorMessage(err))
		return
	}
	// Refresh both parties so the new friendship (or its absence) is reflected.
	g.sendFriendList(sess)
	g.refreshFriendList(requesterID)
	slog.Info("friend request responded", "by", sess.UserUUID, "from", requesterID, "accept", accept)
}

// handleFriendRemove deletes a friendship. Payload: [friendUserID:str8].
func (g *gateway) handleFriendRemove(sess *session.Session, payload []byte) {
	friendID, ok := readStr8Payload(payload, 0)
	if !ok {
		return
	}
	if err := g.friends.Remove(sess.UserUUID, friendID); err != nil {
		sendFriendError(sess.Conn, friendErrorMessage(err))
		return
	}
	g.sendFriendList(sess)
	g.refreshFriendList(friendID)
	slog.Info("friend removed", "by", sess.UserUUID, "friend", friendID)
}

// sendFriendList sends the user's current friend list with live online status.
func (g *gateway) sendFriendList(sess *session.Session) {
	entries, err := g.friends.List(sess.UserUUID)
	if err != nil {
		slog.Error("friend list", "user", sess.UserUUID, "error", err)
		return
	}
	infos := make([]codec.FriendInfo, len(entries))
	for i, e := range entries {
		infos[i] = codec.FriendInfo{
			UserID: e.UserID,
			Name:   e.Name,
			Online: g.sessions.IsUserOnline(e.UserID),
		}
	}
	sess.Conn.Send(message.Encode(message.OpFriendList, 0, codec.EncodeFriendList(infos)))
}

// refreshFriendList re-sends the friend list to a user if they are online.
func (g *gateway) refreshFriendList(userUUID string) {
	if s := g.sessions.FindOnlineByUserUUID(userUUID); s != nil {
		g.sendFriendList(s)
	}
}

// deliverPendingFriendRequests replays any pending incoming friend requests to a
// user who just connected.
func (g *gateway) deliverPendingFriendRequests(sess *session.Session) {
	pending, err := g.friends.PendingIncoming(sess.UserUUID)
	if err != nil {
		slog.Error("pending friend requests", "user", sess.UserUUID, "error", err)
		return
	}
	for _, p := range pending {
		sendFriendRequestRecv(sess.Conn, p.UserID, p.Name)
	}
}

// notifyFriendsStatus tells a user's online friends that they came online/offline.
func (g *gateway) notifyFriendsStatus(userUUID string, online bool) {
	entries, err := g.friends.List(userUUID)
	if err != nil {
		return
	}
	msg := message.Encode(message.OpFriendStatus, 0, codec.EncodeFriendStatus(userUUID, online))
	for _, e := range entries {
		if fs := g.sessions.FindOnlineByUserUUID(e.UserID); fs != nil {
			fs.Conn.Send(msg)
		}
	}
}

func sendFriendError(client *network.Client, msg string) {
	client.Send(message.Encode(message.OpFriendError, 0, codec.EncodeFriendError(msg)))
}

func sendFriendRequestRecv(client *network.Client, requesterUserID, requesterName string) {
	client.Send(message.Encode(message.OpFriendRequestRecv, 0,
		codec.EncodeFriendRequestRecv(requesterUserID, requesterName)))
}

// friendErrorMessage maps a service error to a user-facing message.
func friendErrorMessage(err error) string {
	switch {
	case errors.Is(err, friend.ErrSelf):
		return "you cannot add yourself"
	case errors.Is(err, friend.ErrTargetNotFound):
		return "player not found"
	case errors.Is(err, friend.ErrAlreadyFriends):
		return "already friends"
	case errors.Is(err, friend.ErrRequestPending):
		return "request already pending"
	case errors.Is(err, friend.ErrAmbiguous):
		return "name is ambiguous; use character name"
	case errors.Is(err, friend.ErrNoRequest):
		return "no pending request from that player"
	default:
		return "friend operation failed"
	}
}

// readStr8Payload reads a [len:u8][bytes] string at off, returning the string and
// whether the payload was long enough.
func readStr8Payload(payload []byte, off int) (string, bool) {
	if len(payload) < off+1 {
		return "", false
	}
	n := int(payload[off])
	off++
	if len(payload) < off+n {
		return "", false
	}
	return string(payload[off : off+n]), true
}
