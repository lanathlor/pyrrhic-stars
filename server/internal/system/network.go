package system

import (
	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
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
	msg := message.Encode(message.OpLobbyState, 0, payload)
	for _, c := range w.Clients {
		c.Send(msg)
	}
}

func broadcastWorldState(w *World) {
	players := make([]*entity.Player, 0, len(w.Players))
	for _, p := range w.Players {
		players = append(players, p)
	}
	payload := codec.EncodeWorldState(w.TickNum, players, w.Enemies, w.Projectiles, w.NPCs)
	msg := message.Encode(message.OpWorldState, 0, payload)
	for _, c := range w.Clients {
		c.Send(msg)
	}
}

func broadcastDamageEvents(w *World) {
	for _, evt := range w.DamageEvents {
		payload := codec.EncodeDamageEvent(
			evt.TargetPeerID, evt.SourcePeerID, evt.Amount,
			evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z,
			evt.SourceType,
		)
		msg := message.Encode(message.OpDamageEvent, 0, payload)
		for _, c := range w.Clients {
			c.Send(msg)
		}
	}
}

func broadcastGameFlow(w *World, flowType uint8, text string) {
	payload := codec.EncodeGameFlow(flowType, text)
	msg := message.Encode(message.OpGameFlowEvent, 0, payload)
	for _, c := range w.Clients {
		c.Send(msg)
	}
}
