package system

import (
	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
)

// InputSystem processes all queued client inputs for the current tick.
type InputSystem struct{}

func (s *InputSystem) Tick(w *World, _ float32) {
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
	inp, ok := codec.DecodePlayerInput(payload)
	if !ok {
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
	const tickDt = 1.0 / 20.0
	deltaY := newPos.Y - p.Position.Y
	if deltaY > 0 {
		maxUp := float32(0.0)
		inElevator := false
		for _, ev := range w.Level.Elevators {
			if newPos.X > ev.CenterX-ev.HalfX && newPos.X < ev.CenterX+ev.HalfX &&
				newPos.Z > ev.CenterZ-ev.HalfZ && newPos.Z < ev.CenterZ+ev.HalfZ &&
				newPos.Y >= ev.BottomY-1.0 && newPos.Y <= ev.TopY+1.0 {
				allowed := ev.Speed * tickDt * 1.5
				if allowed > maxUp {
					maxUp = allowed
				}
				inElevator = true
			}
		}
		if !inElevator {
			maxUp = 5.0 * tickDt * 2.0
		}
		if deltaY > maxUp+0.1 {
			newPos.Y = p.Position.Y
		}
	}

	// Client-authoritative: accept position, clamp to boundaries
	p.Position = newPos
	w.Level.ClampPlayer(&p.Position)
	p.RotationY = inp.RotY
	p.LastInput = entity.PlayerInput{PosX: inp.PosX, PosY: inp.PosY, PosZ: inp.PosZ, RotY: inp.RotY, Tick: inp.Tick}
	p.VisualState = inp.VisualState
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

	// Update rotation from ability packet so hitscan uses the exact aim at time of shot
	if inp.RotY != 0 {
		p.RotationY = inp.RotY
	}
	p.AimPitch = inp.AimPitch

	// Dodge is special: it doesn't go through the engine (client-authoritative movement)
	// but we still need to check/spend stamina for vanguard
	if inp.Action == entity.ActionDodge {
		p.SpendResource("stamina", 20)
		w.logCombatEvent(combatlog.LogEntry{
			EventType:    combatlog.EventDodge,
			SourceEntity: combatlog.FormatPlayerID(peerID),
			SourceClass:  p.ClassID,
			PosX:         p.Position.X,
			PosY:         p.Position.Y,
			PosZ:         p.Position.Z,
		})
		return
	}

	// Look up ability from action map
	abilityID, ok := p.ActionMap[inp.Action]
	if !ok {
		return
	}

	// Cast through the ability engine
	w.enemyTargetBuf = enemiesToTargets(w.enemyTargetBuf, w.Enemies)
	ctx := &ability.CastContext{
		Caster:    p,
		Targets:   w.enemyTargetBuf,
		Obstacles: w.Level.Obstacles,
	}
	result := w.AbilityEngine.Cast(abilityID, ctx)
	if !result.OK {
		return
	}

	// Log cast_start
	w.logCombatEvent(combatlog.LogEntry{
		EventType:    combatlog.EventCastStart,
		SourceEntity: combatlog.FormatPlayerID(peerID),
		SourceClass:  p.ClassID,
		AbilityID:    abilityID,
		PosX:         p.Position.X,
		PosY:         p.Position.Y,
		PosZ:         p.Position.Z,
	})

	// Convert ability results to combat damage events and apply threat
	for _, r := range result.Events {
		w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
			TargetPeerID: r.TargetID,
			SourcePeerID: r.SourceID,
			Amount:       r.Amount,
			HitPos:       r.HitPos,
			SourceType:   r.SourceType,
		})

		// Log damage
		w.logCombatEvent(combatlog.LogEntry{
			EventType:    combatlog.EventDamage,
			SourceEntity: combatlog.FormatPlayerID(peerID),
			SourceClass:  p.ClassID,
			Target:       combatlog.FormatEnemyID(r.TargetID),
			AbilityID:    abilityID,
			Amount:       r.Amount,
			PosX:         r.HitPos.X,
			PosY:         r.HitPos.Y,
			PosZ:         r.HitPos.Z,
		})

		if enemy, ok := r.Target.(*entity.Enemy); ok {
			enemy.AddThreat(peerID, r.Amount)
			w.AggroEnemy(enemy, peerID)

			// Log death if enemy died from this hit
			if !enemy.Alive {
				w.logCombatDeath(combatlog.FormatEnemyID(r.TargetID), combatlog.FormatPlayerID(peerID), p.ClassID, abilityID)
				checkEnemyGroupDead(w, enemy)
			}
			w.logPhaseChange(enemy)
		}
	}

	// Log buff applications
	def := w.AbilityEngine.GetAbility(abilityID)
	if def != nil {
		for _, buff := range def.SelfBuffs {
			w.logCombatEvent(combatlog.LogEntry{
				EventType:    combatlog.EventBuffApply,
				SourceEntity: combatlog.FormatPlayerID(peerID),
				SourceClass:  p.ClassID,
				Target:       combatlog.FormatPlayerID(peerID),
				AbilityID:    buff.ID,
			})
		}

		// Log cooldown start
		if def.Cooldown > 0 {
			w.logCombatEvent(combatlog.LogEntry{
				EventType:    combatlog.EventCooldownStart,
				SourceEntity: combatlog.FormatPlayerID(peerID),
				SourceClass:  p.ClassID,
				AbilityID:    abilityID,
			})
		}
	}
}

func handleInteractInput(w *World, peerID uint16, payload []byte) {
	inp, valid := codec.DecodeInteractInput(payload)
	if !valid {
		return
	}
	p, ok := w.Players[peerID]
	if !ok {
		return
	}

	switch inp.Action {
	case message.InteractClassSelect:
		className := inp.ClassName
		if _, ok := entity.Classes[className]; ok {
			// Re-create player with new class
			np := entity.NewPlayer(peerID, className)
			np.Username = p.Username
			np.Position = p.Position
			np.RotationY = p.RotationY
			np.VisualState = p.VisualState
			np.SpawnTick = p.SpawnTick
			*p = *np
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

	switch respawnType {
	case 1: // hub
		if w.OnPlayerRespawnHub != nil {
			w.OnPlayerRespawnHub(peerID)
		}
	case 0: // arena
		if w.State == StateFightOver || w.State == StateLobby || w.State == StateSpawned {
			player.Alive = true
			player.Health = player.MaxHealth
			player.State = entity.PlayerStateMove
			deadGroups := w.DeadGroupIDs()
			player.Position = pickSpawnPoint(w.Level.PlayerSpawns, level.ZoneState{BossDefeated: w.BossDefeated, DeadGroupIDs: deadGroups}, 0)
		}
	}
}
