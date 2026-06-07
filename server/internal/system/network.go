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

	// Dispatch state broadcast based on zone type and game state
	if w.ZoneType == 0 { // OpenWorld
		broadcastWorldState(w)
	} else {
		switch w.State {
		case StateLobby:
			broadcastLobbyState(w)
		case StateSpawned, StateFight, StateFightOver:
			broadcastWorldState(w)
			broadcastDamageEvents(w)
		}
	}

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

// broadcastBufUDP sends via UDP to associated clients and falls back to WS
// for clients without UDP. This ensures clients behind strict NAT (or those
// that haven't completed association yet) still receive game state.
func broadcastBufUDP(w *World, buf []byte) {
	var wsCopy []byte // lazy-allocated WS copy for fallback clients
	clients := w.ClientSnapshot
	if len(clients) == 0 {
		// No snapshot: iterate map directly (hub zones, tests).
		for _, c := range w.Clients {
			broadcastUDPOrWS(c, buf, &wsCopy)
		}
		return
	}
	for _, c := range clients {
		broadcastUDPOrWS(c, buf, &wsCopy)
	}
}

func broadcastUDPOrWS(c *Client, buf []byte, wsCopy *[]byte) {
	if c.HasUDP != nil && c.HasUDP() {
		c.SendUDP(buf)
	} else {
		if *wsCopy == nil {
			*wsCopy = make([]byte, len(buf))
			copy(*wsCopy, buf)
		}
		c.Send(*wsCopy)
	}
}

func broadcastLobbyState(w *World) {
	w.lobbyInfoBuf = w.lobbyInfoBuf[:0]
	for _, p := range w.Players {
		w.lobbyInfoBuf = append(w.lobbyInfoBuf, codec.LobbyPlayerInfo{
			PeerID:    p.ID,
			ClassName: p.ClassName(),
			SpecName:  p.SpecID,
			Username:  p.Username,
			Ready:     p.Ready,
		})
	}
	payload := codec.EncodeLobbyState(w.lobbyInfoBuf)
	// Pooled LobbyBuf: encode the full message.
	if cap(w.LobbyBuf) < 4+len(payload) {
		w.LobbyBuf = make([]byte, 0, 512)
	}
	w.LobbyBuf = w.LobbyBuf[:0]
	w.LobbyBuf = message.AppendEncode(w.LobbyBuf, message.OpLobbyState, 0, payload)
	broadcastBufWS(w, w.LobbyBuf)
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
