package main

import (
	"log/slog"

	"codex-online/server/internal/message"
	"codex-online/server/internal/session"
)

// handleUserMessage processes user/account opcodes.
func (g *gateway) handleUserMessage(sess *session.Session, opcode uint16, payload []byte) {
	switch opcode {
	case message.OpSetUsername:
		if len(payload) < 1 {
			return
		}
		nameLen := int(payload[0])
		if len(payload) < 1+nameLen {
			return
		}
		raw := string(payload[1 : 1+nameLen])
		sess.Username = g.users.CleanUsername(raw, sess.ID)
		slog.Info("username set", "player_id", sess.ID, "username", sess.Username)

	default:
		slog.Warn("unknown user opcode", "opcode", opcode)
	}
}
