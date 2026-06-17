package system

import (
	"encoding/binary"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/message"
)

// NetworkSystem broadcasts world state, damage events, and game flow events
// to all connected clients. It runs last in the system pipeline.
type NetworkSystem struct{}

func (s *NetworkSystem) Tick(w *World, _ float32) {
	// Broadcast game flow events (produced by GameFlowSystem during this tick)
	for _, evt := range w.GameFlowEvents {
		broadcastGameFlow(w, evt.FlowType, evt.Text)
	}
	w.GameFlowEvents = w.GameFlowEvents[:0]

	// Broadcast lobby state while in lobby phase.
	if w.LobbyActive {
		broadcastLobbyState(w)
	}

	// Broadcast world state and damage events to all clients.
	broadcastWorldState(w)
	broadcastDamageEvents(w)

	// Clear damage events after broadcast
	w.DamageEvents = w.DamageEvents[:0]
}

// broadcastBufWS copies the pooled buffer once and shares the immutable copy
// with all WS clients. Used for reliable messages (game flow, lobby state)
// that always go over WebSocket.
func broadcastBufWS(w *World, buf []byte) {
	msg := make([]byte, len(buf))
	copy(msg, buf)
	if len(w.ClientSnapshot) > 0 {
		for _, c := range w.ClientSnapshot {
			c.Send(msg)
		}
	} else {
		for _, c := range w.Clients {
			c.Send(msg)
		}
	}
}

// broadcastBufUDP sends via UDP to associated clients. Clients without a
// confirmed UDP association are skipped (they will be disconnected shortly).
func broadcastBufUDP(w *World, buf []byte) {
	clients := w.ClientSnapshot
	if len(clients) == 0 {
		for _, c := range w.Clients {
			if c.HasUDP != nil && c.HasUDP() {
				c.SendUDP(buf)
			}
		}
		return
	}
	for _, c := range clients {
		if c.HasUDP != nil && c.HasUDP() {
			c.SendUDP(buf)
		}
	}
}

func broadcastWorldState(w *World) {
	// Use pooled SendBuf: write header placeholder, encode payload, then fill header.
	// Capacity is preserved across ticks; only grows if needed.
	const headerSize = 4
	if cap(w.SendBuf) < headerSize {
		w.SendBuf = make([]byte, headerSize, 4096) // pre-allocate 4KB
	}
	w.SendBuf = w.SendBuf[:headerSize] // keep capacity, reset length

	// Encode world state payload into SendBuf after the header area.
	// AppendEncodeWorldState grows the buffer if needed.
	w.SendBuf = codec.AppendEncodeWorldState(w.SendBuf, w.TickNum, w.Players, w.Enemies, w.Projectiles, w.NPCs)

	// Active telegraphs, appended after the world-state body (trailing array,
	// backward-compatible with older decoders that stop after the NPC block).
	w.SendBuf = codec.AppendTelegraphs(w.SendBuf, buildTelegraphs(w))

	// Now fill in the header: [opcode:2][senderID:2]
	binary.BigEndian.PutUint16(w.SendBuf[0:2], message.OpWorldState)
	binary.BigEndian.PutUint16(w.SendBuf[2:4], 0)

	broadcastBufUDP(w, w.SendBuf)
}

func broadcastDamageEvents(w *World) {
	const headerSize = 4 // [opcode:2][senderID:2]
	for _, evt := range w.DamageEvents {
		// Encode directly into pooled DamageBuf: header + damage payload.
		if cap(w.DamageBuf) < headerSize+25 {
			w.DamageBuf = make([]byte, 0, 256)
		}
		w.DamageBuf = w.DamageBuf[:headerSize]
		w.DamageBuf = codec.AppendEncodeDamageEvent(w.DamageBuf,
			evt.TargetPeerID, evt.SourcePeerID, evt.Amount,
			evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z,
			evt.SourceType, evt.Overheal,
		)
		binary.BigEndian.PutUint16(w.DamageBuf[0:2], message.OpDamageEvent)
		binary.BigEndian.PutUint16(w.DamageBuf[2:4], 0)
		broadcastBufUDP(w, w.DamageBuf)
	}
}

// EncodeLobbyStateMsg builds the full lobby state message for the world's
// current lobby phase, countdown, and player ready states. Callers must hold
// the zone lock.
func EncodeLobbyStateMsg(w *World) []byte {
	infos := make([]codec.LobbyPlayerInfo, 0, len(w.Players))
	for _, p := range w.Players {
		infos = append(infos, codec.LobbyPlayerInfo{
			PeerID:    p.ID,
			ClassName: p.ClassID,
			SpecName:  p.SpecID,
			Username:  p.Username,
			Ready:     p.Ready,
		})
	}
	var phase uint8
	var countdownSecs uint8
	if w.LobbyCountdown > 0 {
		phase = codec.LobbyPhaseCountdown
		countdownSecs = uint8((w.LobbyCountdown + 19) / 20) // ceil(ticks / 20)
	}
	payload := codec.EncodeLobbyState(phase, countdownSecs, infos)
	return message.Encode(message.OpLobbyState, 0, payload)
}

func broadcastLobbyState(w *World) {
	broadcastBufWS(w, EncodeLobbyStateMsg(w))
}

func broadcastGameFlow(w *World, flowType uint8, text string) {
	payload := codec.EncodeGameFlow(flowType, text)
	// Pooled GameFlowBuf: encode the full message.
	if cap(w.GameFlowBuf) < 4+len(payload) {
		w.GameFlowBuf = make([]byte, 0, 256)
	}
	w.GameFlowBuf = w.GameFlowBuf[:0]
	w.GameFlowBuf = message.AppendEncode(w.GameFlowBuf, message.OpGameFlowEvent, 0, payload)
	broadcastBufWS(w, w.GameFlowBuf)
}
