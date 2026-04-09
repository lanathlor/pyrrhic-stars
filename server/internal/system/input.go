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

	// Y validation: hard reject if outside zone Y bounds
	if w.Level.PlayerBoundsMaxY != 0 || w.Level.PlayerBoundsMinY != 0 {
		if newPos.Y < w.Level.PlayerBoundsMinY || newPos.Y > w.Level.PlayerBoundsMaxY {
			return
		}
	}

	// Y validation: limit upward movement speed
	// dt = 1/20 = 0.05 (constant 20Hz tick rate)
	const tickDt = 1.0 / 20.0
	deltaY := newPos.Y - p.Position.Y
	if deltaY > 0 {
		maxUp := float32(0.0)
		inElevator := false
		for _, ev := range w.Level.Elevators {
			if newPos.X > ev.CenterX-ev.HalfX && newPos.X < ev.CenterX+ev.HalfX &&
				newPos.Z > ev.CenterZ-ev.HalfZ && newPos.Z < ev.CenterZ+ev.HalfZ &&
				newPos.Y >= ev.BottomY-1.0 && newPos.Y <= ev.TopY+1.0 {
				// Elevator: allow speed * dt * 1.5 (smoothstep peaks at ~1.5x average)
				allowed := ev.Speed * tickDt * 1.5
				if allowed > maxUp {
					maxUp = allowed
				}
				inElevator = true
			}
		}
		if !inElevator {
			// Normal: allow jump velocity (~5 m/s upward, generous for frame jitter)
			maxUp = 5.0 * tickDt * 2.0
		}
		if deltaY > maxUp+0.1 {
			newPos.Y = p.Position.Y // reject Y component, keep XZ
		}
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

	// Update rotation from ability packet so hitscan uses the exact aim at time of shot
	if inp.RotY != 0 {
		p.RotationY = inp.RotY
	}
	p.AimPitch = inp.AimPitch

	switch inp.Action {
	case entity.ActionShoot:
		// Gunner: hitscan, gated by fire cooldown
		if p.ClassName == "gunner" && p.FireCooldown <= 0 {
			p.FireCooldown = 0.18
			p.State = entity.PlayerStateAttack
			evt, hitEnemy := combat.ResolvePlayerAttackOnEnemies(p, w.Enemies, w.Level.Obstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				w.DamageEvents = append(w.DamageEvents, *evt)
				hitEnemy.AddThreat(peerID, evt.Amount)
				w.AggroEnemy(hitEnemy, peerID)
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
			evt, hitEnemy := combat.ResolvePlayerAttackOnEnemies(p, w.Enemies, w.Level.Obstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				w.DamageEvents = append(w.DamageEvents, *evt)
				hitEnemy.AddThreat(peerID, evt.Amount)
				w.AggroEnemy(hitEnemy, peerID)
			}
		}
	case entity.ActionGuard:
		if p.ClassName == "blade_dancer" && !p.GuardActive {
			p.GuardActive = true
			p.GuardTimer = 1.5
		}
	case entity.ActionHeavy:
		if (p.ClassName == "vanguard" || p.ClassName == "blade_dancer") && p.FireCooldown <= 0 {
			if p.ClassName == "vanguard" {
				p.FireCooldown = 0.8
			} else {
				p.FireCooldown = 0.5
			}
			p.State = entity.PlayerStateAttack
			evt, hitEnemy := combat.ResolvePlayerAttackOnEnemies(p, w.Enemies, w.Level.Obstacles)
			if evt != nil {
				evt.SourcePeerID = peerID
				w.DamageEvents = append(w.DamageEvents, *evt)
				hitEnemy.AddThreat(peerID, evt.Amount)
				w.AggroEnemy(hitEnemy, peerID)
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
		if w.State == StateFightOver || w.State == StateLobby || w.State == StateSpawned {
			player.Alive = true
			player.Health = player.MaxHealth
			player.State = entity.PlayerStateMove
			player.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
		}
	}
}
