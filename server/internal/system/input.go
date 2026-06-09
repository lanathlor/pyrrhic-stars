package system

import (
	"math"
	"slices"

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
		if w.DevMode && message.IsDebugInput(inp.Opcode) {
			handleDebugInput(w, inp.PeerID, inp.Opcode, inp.Payload)
			continue
		}
		switch inp.Opcode {
		case message.OpPlayerInput:
			handlePlayerInput(w, inp.PeerID, inp.Payload)
		case message.OpAbilityInput:
			handleAbilityInput(w, inp.PeerID, inp.Payload)
		case message.OpInteractInput:
			handleInteractInput(w, inp.PeerID, inp.Payload)
		case message.OpRespawnRequest:
			handleRespawnRequest(w, inp.PeerID, inp.Payload)
		case message.OpSetLoadout:
			handleSetLoadout(w, inp.PeerID, inp.Payload)
		case message.OpSetFluxCommitment:
			handleSetFluxCommitment(w, inp.PeerID, inp.Payload)
		}
	}
	w.InputQueue = w.InputQueue[:0]
}

func hasNaNOrInf32(v float32) bool {
	f := float64(v)
	return math.IsNaN(f) || math.IsInf(f, 0)
}

func handlePlayerInput(w *World, peerID uint16, payload []byte) {
	inp, ok := codec.DecodePlayerInput(payload)
	if !ok {
		return
	}
	if hasNaNOrInf32(inp.PosX) || hasNaNOrInf32(inp.PosY) || hasNaNOrInf32(inp.PosZ) || hasNaNOrInf32(inp.RotY) || hasNaNOrInf32(inp.AimPitch) {
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

	newPos := entity.Vec3{X: inp.PosX, Y: inp.PosY, Z: inp.PosZ}
	newPos, valid := validateAndClampPosition(w, p, newPos)
	if !valid {
		return
	}

	// Client-authoritative: accept position, clamp to boundaries
	p.Position = newPos
	w.Level.ClampPlayer(&p.Position)
	p.RotationY = inp.RotY
	p.LastInput = entity.PlayerInput{PosX: inp.PosX, PosY: inp.PosY, PosZ: inp.PosZ, RotY: inp.RotY, Tick: inp.Tick}
	p.VisualState = inp.VisualState
	p.AimPitch = inp.AimPitch
}

func validateAndClampPosition(w *World, p *entity.Player, newPos entity.Vec3) (entity.Vec3, bool) {
	// Reject positions that teleport too far from the server-assigned position.
	dx := newPos.X - p.Position.X
	dy := newPos.Y - p.Position.Y
	dz := newPos.Z - p.Position.Z
	dist := dx*dx + dy*dy + dz*dz
	if dist > 100.0 { // > 10 units teleport = reject
		return newPos, false
	}

	// Y validation: hard reject if outside zone Y bounds
	if w.Level.PlayerBoundsMaxY != 0 || w.Level.PlayerBoundsMinY != 0 {
		if newPos.Y < w.Level.PlayerBoundsMinY || newPos.Y > w.Level.PlayerBoundsMaxY {
			return newPos, false
		}
	}

	newPos = clampPositionY(w, p, newPos)
	newPos = clampPositionXZ(p, newPos)
	return newPos, true
}

func clampPositionY(w *World, p *entity.Player, newPos entity.Vec3) entity.Vec3 {
	const tickDt = 1.0 / 20.0
	deltaY := newPos.Y - p.Position.Y

	// Upward movement validation (elevator + jump tolerance)
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
			return newPos
		}
	}

	// Navmesh floor validation: prevent clipping below the floor
	if w.Level.Navmesh != nil {
		if floorY, ok := w.Level.Navmesh.SampleY(newPos.X, newPos.Z, newPos.Y); ok {
			if newPos.Y < floorY-0.1 {
				newPos.Y = floorY + 0.1
			}
		}
	}

	return newPos
}

func clampPositionXZ(p *entity.Player, newPos entity.Vec3) entity.Vec3 {
	const tickDt = 1.0 / 20.0
	speedMult := float32(1.0)
	if p.HasBuff("brace") {
		speedMult = 0.0
	} else if p.HasBuff("vg_shield_block") {
		speedMult = 0.4
	}
	hDx := newPos.X - p.Position.X
	hDz := newPos.Z - p.Position.Z
	hDistSq := hDx*hDx + hDz*hDz
	mv := p.Movement()
	maxSpd := mv.SprintSpeed * speedMult
	maxDist := maxSpd * tickDt * 1.5 // 50% tolerance
	if hDistSq > maxDist*maxDist {
		if speedMult == 0 {
			newPos.X = p.Position.X
			newPos.Z = p.Position.Z
		} else {
			scale := maxDist / float32(math.Sqrt(float64(hDistSq)))
			newPos.X = p.Position.X + hDx*scale
			newPos.Z = p.Position.Z + hDz*scale
		}
	}
	return newPos
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
	// but we still need to check/spend stamina for vanguard.
	// Harmonist has no dodge — gust step is their mobility (goes through loadout).
	if inp.Action == entity.ActionDodge {
		handleDodgeInput(w, p, peerID)
		return
	}

	// Action 255: explicit channel/sustain cancel (ESC on client)
	if inp.Action == 255 {
		handleCancelInput(w, p, peerID)
		return
	}

	// Look up ability from action map
	abilityID, ok := p.ActionMap[inp.Action]
	if !ok {
		return
	}

	// If the ability has a commit phase, route through the PlayerAbilityRunner
	// instead of committing immediately. The runner ticks in CombatSystem and fires
	// the actual Commit when the commit timer expires.
	if def := w.AbilityEngine.GetAbility(abilityID); def != nil && def.CommitTime > 0 {
		handleCommitPhaseAbility(w, p, peerID, inp, def)
		return
	}

	commitAbilityAndLog(w, p, peerID, abilityID, inp)
}

// cancelActiveRunner cancels an active ability runner, applies sustain
// cooldown, interrupts Confluence if channeling, and syncs state to player.
// Returns false if the runner was not busy or refused to cancel.
func cancelActiveRunner(p *entity.Player, runner *ability.PlayerAbilityRunner) bool {
	if runner == nil || !runner.IsBusy() {
		return false
	}
	wasChanneling := runner.Phase == ability.PRunnerCommit || runner.Phase == ability.PRunnerSustain
	if id, cd := runner.SustainCooldownOnCancel(); cd > 0 {
		p.Cooldowns[id] = cd
	}
	if !runner.Cancel() {
		return false
	}
	if wasChanneling && p.Confluence != nil {
		p.Confluence.OnInterrupt()
	}
	runner.SyncToPlayer(p)
	return true
}

func handleDodgeInput(w *World, p *entity.Player, peerID uint16) {
	if p.ClassID == entity.ClassArcanotechnicien {
		return // harmonist has no dodge
	}
	// Cancel any active channel when dodging
	cancelActiveRunner(p, w.AbilityRunners[peerID])
	if !p.SpendResource("stamina", 20) {
		return
	}
	// Server-side i-frames: prevent damage (and Onslaught reset) during dodge
	p.Invincible = true
	p.InvincibleTimer = 0.15
	w.logCombatEvent(combatlog.LogEntry{
		EventType:    combatlog.EventDodge,
		SourceEntity: combatlog.FormatPlayerID(peerID),
		SourceClass:  p.ClassID,
		PosX:         p.Position.X,
		PosY:         p.Position.Y,
		PosZ:         p.Position.Z,
	})
}

func handleCancelInput(w *World, p *entity.Player, peerID uint16) {
	cancelActiveRunner(p, w.AbilityRunners[peerID])
}

func handleCommitPhaseAbility(w *World, p *entity.Player, peerID uint16, inp *codec.AbilityInputMsg, def *ability.AbilityDef) {
	runner := w.AbilityRunners[peerID]
	if runner == nil {
		runner = &ability.PlayerAbilityRunner{}
		w.AbilityRunners[peerID] = runner
	}
	if runner.IsBusy() {
		if !cancelActiveRunner(p, runner) {
			return // in execute or cooldown phase, cannot cancel
		}
		// Cancel during sustain enters cooldown -- force reset so Start succeeds
		runner.ForceReset()
	}
	runner.Start(def)
	p.ChannelTargetID = inp.TargetPeerID
	runner.SyncToPlayer(p)
}

func commitAbilityAndLog(w *World, p *entity.Player, peerID uint16, abilityID string, inp *codec.AbilityInputMsg) {
	// Cancel any active channel when committing an instant ability
	if runner := w.AbilityRunners[peerID]; runner != nil && runner.IsBusy() {
		cancelActiveRunner(p, runner)
		runner.ForceReset()
	}

	// Commit through the ability engine
	w.enemyTargetBuf = enemiesToTargets(w.enemyTargetBuf, w.Enemies)
	ctx := &ability.CommitContext{
		Committer:    p,
		Targets:      w.enemyTargetBuf,
		Obstacles:    w.Obstacles,
		Allies:       w.Players,
		TargetPeerID: inp.TargetPeerID,
		SpawnZone: func(zone *entity.HealingZone) {
			w.NextZoneID++
			zone.ID = w.NextZoneID
			w.HealingZones = append(w.HealingZones, zone)
		},
		SpawnLink: func(link *entity.DamageLink) {
			w.DamageLinks = append(w.DamageLinks, link)
		},
	}
	result := w.AbilityEngine.Commit(abilityID, ctx)
	if !result.OK {
		return
	}

	// Log commit_start
	w.logCombatEvent(combatlog.LogEntry{
		EventType:    combatlog.EventCommitStart,
		SourceEntity: combatlog.FormatPlayerID(peerID),
		SourceClass:  p.ClassID,
		AbilityID:    abilityID,
		PosX:         p.Position.X,
		PosY:         p.Position.Y,
		PosZ:         p.Position.Z,
	})

	processCommitDamageEvents(w, p, peerID, abilityID, result.Events)
	processCommitHealEvents(w, result.Heals)
	startSustainRunner(w, p, peerID, abilityID)
	logAbilityBuffsAndCooldown(w, p, peerID, abilityID)
}

func processCommitDamageEvents(w *World, p *entity.Player, peerID uint16, abilityID string, events []ability.DamageResult) {
	// Convert ability results to combat damage events and apply threat
	for _, r := range events {
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
}

func processCommitHealEvents(w *World, heals []ability.HealResult) {
	// Convert heal results to damage events (SourceType=5 distinguishes heals on the client)
	for _, h := range heals {
		w.DamageEvents = append(w.DamageEvents, combat.DamageEvent{
			TargetPeerID: h.TargetID,
			SourcePeerID: h.SourceID,
			Amount:       h.Amount,
			Overheal:     h.Overheal,
			HitPos:       h.HitPos,
			SourceType:   h.SourceType,
		})
	}
}

func startSustainRunner(w *World, p *entity.Player, peerID uint16, abilityID string) {
	// If the instant ability supports sustain, start the runner in sustain phase
	def := w.AbilityEngine.GetAbility(abilityID)
	if def == nil || !def.Sustain {
		return
	}
	runner := w.AbilityRunners[peerID]
	if runner == nil {
		runner = &ability.PlayerAbilityRunner{}
		w.AbilityRunners[peerID] = runner
	}
	runner.StartSustain(def, p.Position, w.TickNum)
	runner.SyncToPlayer(p)
}

func logAbilityBuffsAndCooldown(w *World, p *entity.Player, peerID uint16, abilityID string) {
	// Log buff applications
	def := w.AbilityEngine.GetAbility(abilityID)
	if def == nil {
		return
	}
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
		if w.BossDefeated {
			if w.OnPlayerRespawnHub != nil {
				w.OnPlayerRespawnHub(peerID)
			}
		}
	case message.InteractSpecSelect:
		specID := inp.ClassName // reuse the string field
		classDef, ok := entity.Classes[p.ClassID]
		if !ok {
			return
		}
		spec := classDef.GetSpec(specID)
		if spec == nil || !spec.Implemented {
			return
		}
		if p.SpecID == specID {
			return // already on this spec
		}
		np := entity.NewPlayerWithSpec(peerID, p.ClassID, specID)
		np.Username = p.Username
		np.Position = p.Position
		np.RotationY = p.RotationY
		np.VisualState = p.VisualState
		np.SpawnTick = p.SpawnTick
		*p = *np
	}
}

func handleSetLoadout(w *World, peerID uint16, payload []byte) {
	slots, ok := codec.DecodeSetLoadout(payload)
	if !ok {
		return
	}
	p, ok := w.Players[peerID]
	if !ok {
		return
	}

	// Validate every non-empty slot against the player's class/spec abilities.
	allowed := p.AllowedAbilities()
	for _, id := range slots {
		if id == "" {
			continue
		}
		valid := slices.Contains(allowed, id)
		if !valid {
			return
		}
	}

	if p.Loadout == nil {
		p.Loadout = &entity.Loadout{}
	}
	p.Loadout.Slots = slots
	p.ApplyLoadout()
}

func handleSetFluxCommitment(w *World, peerID uint16, payload []byte) {
	entries, ok := codec.DecodeFluxCommitment(payload)
	if !ok {
		return
	}
	p, ok := w.Players[peerID]
	if !ok {
		return
	}
	if p.FluxCommit == nil {
		return
	}

	// Validate total = 100%.
	var total int
	for _, e := range entries {
		total += int(e.Percentage)
	}
	if total != 100 {
		return
	}

	// Build school → percentage map and apply.
	schools := make(map[string]float32, len(entries))
	for _, e := range entries {
		schools[e.School] = float32(e.Percentage) / 100.0
	}
	p.FluxCommit.SetCommitment(schools)
	p.SyncFluxAggregate()
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
	case 0: // local respawn (allowed unless boss room is sealed)
		if !w.AnyGateClosed() {
			player.Alive = true
			player.Health = player.MaxHealth
			player.State = entity.PlayerStateMove
			player.Velocity = entity.Vec3{}
			deadGroups := w.DeadGroupIDs()
			player.Position = pickSpawnPoint(w.Level.PlayerSpawns, level.ZoneState{BossDefeated: w.BossDefeated, DeadGroupIDs: deadGroups}, 0)
			w.WipeHandled = false // allow future wipe detection
		}
	}
}
