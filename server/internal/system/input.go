package system

import (
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

// InputSystem processes all queued client inputs for the current tick.
type InputSystem struct{}

func (s *InputSystem) Tick(w *World, dt float32) {
	for _, inp := range w.InputQueue {
		switch inp.Opcode {
		case message.OpPlayerInput:
			handlePlayerInput(w, inp.PeerID, inp.Payload)
		case message.OpAbilityInput:
			handleAbilityInput(w, inp.PeerID, inp.Payload)
		case message.OpInteractInput:
			handleInteractInput(w, inp.PeerID, inp.Payload)
		case message.OpRespawnRequest:
			handleRespawnRequest(w, inp.PeerID, inp.Payload)
		}
	}
	w.InputQueue = w.InputQueue[:0]
}

func handlePlayerInput(w *World, peerID uint16, payload []byte) {
	inp := codec.DecodePlayerInput(payload)
	if inp == nil {
		return
	}
	p, ok := w.Players[peerID]
	if !ok {
		return
	}

	// After a zone transfer, the client may send stale positions from the
	// previous zone for a few frames before it processes the transfer.
	// Reject all position updates within 10 ticks (~500ms) of spawn.
	const spawnGraceTicks uint32 = 10
	if p.SpawnTick > 0 && w.TickNum-p.SpawnTick < spawnGraceTicks {
		return
	}

	// Reject positions that teleport too far from the server-assigned position.
	newPos := entity.Vec3{X: inp.PosX, Y: inp.PosY, Z: inp.PosZ}
	dx := newPos.X - p.Position.X
	dy := newPos.Y - p.Position.Y
	dz := newPos.Z - p.Position.Z
	dist := dx*dx + dy*dy + dz*dz
	if dist > 100.0 { // > 10 units teleport = reject
		return
	}

	// Client-authoritative: accept position, clamp to boundaries
	p.Position = newPos
	w.Level.ClampPlayer(&p.Position)
	p.RotationY = inp.RotY
	p.LastInput = &entity.PlayerInput{PosX: inp.PosX, PosY: inp.PosY, PosZ: inp.PosZ, RotY: inp.RotY, Tick: inp.Tick}
	p.AnimName = inp.AnimName
	p.AnimSpeed = inp.AnimSpeed
	p.AimPitch = inp.AimPitch
}

func handleAbilityInput(w *World, peerID uint16, payload []byte) {
	inp := codec.DecodeAbilityInput(payload)
	if inp == nil {
		return
	}
	p, ok := w.Players[peerID]
	if !ok || !p.Alive {
		return
	}
	if w.State != StateFight {
		return
	}

	enemy := w.FirstEnemy()

	switch inp.Action {
	case entity.ActionShoot:
		// Gunner: hitscan, gated by fire cooldown
		if p.ClassName == "gunner" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.18
			p.State = entity.PlayerStateAttack
			p.AimPitch = inp.AimPitch
			evt := combat.ResolvePlayerAttackOnEnemy(p, enemy, w.Level.Obstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				w.DamageEvents = append(w.DamageEvents, *evt)
				enemy.AddThreat(peerID, evt.Amount)
			}
		}
	case entity.ActionMelee:
		// Vanguard/blade_dancer: melee, gated by cooldown
		if p.FireCooldown <= 0 {
			if p.ClassName == "vanguard" {
				p.FireCooldown = 0.55
			} else {
				p.FireCooldown = 0.3
			}
			p.State = entity.PlayerStateAttack
			evt := combat.ResolvePlayerAttackOnEnemy(p, enemy, w.Level.Obstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				w.DamageEvents = append(w.DamageEvents, *evt)
				enemy.AddThreat(peerID, evt.Amount)
			}
		}
	case entity.ActionHeavy:
		if p.ClassName == "vanguard" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.8
			p.State = entity.PlayerStateAttack
			evt := combat.ResolvePlayerAttackOnEnemy(p, enemy, w.Level.Obstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				w.DamageEvents = append(w.DamageEvents, *evt)
				enemy.AddThreat(peerID, evt.Amount)
			}
		}
	}
}

func handleInteractInput(w *World, peerID uint16, payload []byte) {
	inp := codec.DecodeInteractInput(payload)
	if inp == nil {
		return
	}
	p, ok := w.Players[peerID]
	if !ok {
		return
	}

	switch inp.Action {
	case message.InteractClassSelect:
		className := inp.ClassName
		if className == "gunner" || className == "vanguard" || className == "blade_dancer" {
			p.ClassName = className
			// Re-init class stats
			np := entity.NewPlayer(peerID, className)
			p.Health = np.Health
			p.MaxHealth = np.MaxHealth
			p.Stamina = np.Stamina
			p.MaxStamina = np.MaxStamina
		}
	case message.InteractReadyToggle:
		p.Ready = !p.Ready
	case message.InteractExitPortal:
		if w.State == StateFightOver && w.BossDefeated {
			if w.OnPlayerRespawnHub != nil {
				w.OnPlayerRespawnHub(peerID)
			}
		}
	}
}

func handleRespawnRequest(w *World, peerID uint16, payload []byte) {
	respawnType, ok := codec.DecodeRespawnRequest(payload)
	if !ok {
		return
	}
	player := w.Players[peerID]
	if player == nil || player.Alive {
		return
	}

	if respawnType == 1 { // hub
		if w.OnPlayerRespawnHub != nil {
			w.OnPlayerRespawnHub(peerID)
		}
	} else if respawnType == 0 { // arena
		if w.State == StateFightOver || w.State == StateLobby {
			player.Alive = true
			player.Health = player.MaxHealth
			player.State = entity.PlayerStateMove
			player.Position = entity.Vec3{X: 0, Y: 0.1, Z: 20}
		}
	}
}
