package system

import (
	"encoding/binary"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/message"
)

// NetworkSystem broadcasts world state, damage events, and game flow events
// to all connected clients. It runs last in the system pipeline.
type NetworkSystem struct{}

func (s *NetworkSystem) Tick(w *World, dt float32) {
	// Broadcast game flow events (produced by GameFlowSystem during this tick)
	for _, evt := range w.GameFlowEvents {
		broadcastGameFlow(w, evt.FlowType, evt.Text)
	}
	w.GameFlowEvents = w.GameFlowEvents[:0]

	// Dispatch state broadcast based on zone type and game state
	if w.ZoneType == 0 { // Hub
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

func broadcastLobbyState(w *World) {
	infos := make([]codec.LobbyPlayerInfo, 0, len(w.Players))
	for _, p := range w.Players {
		infos = append(infos, codec.LobbyPlayerInfo{
			PeerID:    p.PeerID,
			ClassName: p.ClassName,
			Username:  p.Username,
			Ready:     p.Ready,
		})
	}
	payload := codec.EncodeLobbyState(infos)
	// Pooled LobbyBuf: encode the full message.
	if cap(w.LobbyBuf) < 4+len(payload) {
		w.LobbyBuf = make([]byte, 0, 512)
	}
	w.LobbyBuf = w.LobbyBuf[:0]
	w.LobbyBuf = message.AppendEncode(w.LobbyBuf, message.OpLobbyState, 0, payload)
	for _, c := range w.Clients {
		if w.TestMode {
			msg := make([]byte, len(w.LobbyBuf))
			copy(msg, w.LobbyBuf)
			c.Send(msg)
		} else {
			c.Send(w.LobbyBuf)
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

	// Now fill in the header: [opcode:2][senderID:2]
	binary.BigEndian.PutUint16(w.SendBuf[0:2], message.OpWorldState)
	binary.BigEndian.PutUint16(w.SendBuf[2:4], 0)

	for _, c := range w.Clients {
		if w.TestMode {
			msg := make([]byte, len(w.SendBuf))
			copy(msg, w.SendBuf)
			c.Send(msg)
		} else {
			c.Send(w.SendBuf)
		}
	}
}

func broadcastDamageEvents(w *World) {
	const headerSize = 4 // [opcode:2][senderID:2]
	for _, evt := range w.DamageEvents {
		// Encode directly into pooled DamageBuf: header + damage payload.
		if cap(w.DamageBuf) < headerSize+21 {
			w.DamageBuf = make([]byte, 0, 256)
		}
		w.DamageBuf = w.DamageBuf[:headerSize]
		w.DamageBuf = codec.AppendEncodeDamageEvent(w.DamageBuf,
			evt.TargetPeerID, evt.SourcePeerID, evt.Amount,
			evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z,
			evt.SourceType,
		)
		binary.BigEndian.PutUint16(w.DamageBuf[0:2], message.OpDamageEvent)
		binary.BigEndian.PutUint16(w.DamageBuf[2:4], 0)
		for _, c := range w.Clients {
			if w.TestMode {
				msg := make([]byte, len(w.DamageBuf))
				copy(msg, w.DamageBuf)
				c.Send(msg)
			} else {
				c.Send(w.DamageBuf)
			}
		}
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
	for _, c := range w.Clients {
		if w.TestMode {
			msg := make([]byte, len(w.GameFlowBuf))
			copy(msg, w.GameFlowBuf)
			c.Send(msg)
		} else {
			c.Send(w.GameFlowBuf)
		}
	}
}
