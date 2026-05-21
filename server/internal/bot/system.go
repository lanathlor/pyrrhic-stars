package bot

import (
	"codex-online/server/internal/codec"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
)

// System implements system.System and drives bot behavior each tick.
// It also intercepts bot-related debug opcodes from the input queue.
type System struct {
	Manager *Manager
}

// Tick processes bot opcodes and updates all bots.
func (s *System) Tick(w *system.World, dt float32) {
	if s.Manager == nil {
		return
	}

	// Scan input queue for bot opcodes, handle them and remove from queue.
	clean := w.InputQueue[:0]
	for _, inp := range w.InputQueue {
		switch inp.Opcode {
		case message.OpDebugSpawnBot:
			s.handleSpawn(w, inp.PeerID, inp.Payload)
		case message.OpDebugDismissBot:
			s.handleDismiss(w, inp.PeerID, inp.Payload)
		default:
			clean = append(clean, inp)
		}
	}
	w.InputQueue = clean

	s.Manager.TickAll(w, dt)
}

func (s *System) handleSpawn(w *system.World, peerID uint16, payload []byte) {
	className, specID, ok := codec.DecodeDebugSpawnBot(payload)
	if !ok {
		return
	}
	_, err := s.Manager.SpawnBot(peerID, className, specID, w)
	if err != nil {
		return
	}
}

func (s *System) handleDismiss(w *system.World, peerID uint16, payload []byte) {
	botID, ok := codec.DecodeDebugDismissBot(payload)
	if !ok {
		return
	}
	if botID == 0 {
		s.Manager.DismissAllForOwner(peerID, w)
	} else {
		s.Manager.DismissBot(botID, w)
	}
}
