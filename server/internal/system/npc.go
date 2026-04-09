package system

import (
	"math"

	"codex-online/server/internal/entity"
)

// NPCSystem moves hub NPCs along their waypoints with idle pauses.
type NPCSystem struct{}

func (s *NPCSystem) Tick(w *World, dt float32) {
	for _, npc := range w.NPCs {
		switch npc.State {
		case entity.NPCIdle:
			npc.IdleTimer -= dt
			if npc.IdleTimer <= 0 && len(npc.Waypoints) > 1 {
				npc.WaypointIdx = (npc.WaypointIdx + 1) % len(npc.Waypoints)
				npc.State = entity.NPCWalk
			}
		case entity.NPCWalk:
			target := npc.Waypoints[npc.WaypointIdx]
			dir := target.Sub(npc.Position)
			dir.Y = 0 // stay on ground plane
			dist := dir.Length()
			step := npc.Speed * dt
			if dist <= step {
				npc.Position = target
				npc.State = entity.NPCIdle
				npc.IdleTimer = npc.IdleDuration
			} else {
				move := dir.Normalized().Scale(step)
				npc.Position = npc.Position.Add(move)
				// Face movement direction (Godot characters face -Z, so offset by PI)
				npc.RotationY = float32(math.Atan2(float64(move.X), float64(move.Z))) + math.Pi
			}
		}
	}
}
