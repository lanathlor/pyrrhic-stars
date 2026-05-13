package system

import (
	"log/slog"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

// handleDebugInput dispatches a debug opcode. Called from InputSystem when DevMode is true.
func handleDebugInput(w *World, peerID uint16, opcode uint16, payload []byte) {
	switch opcode {
	case message.OpDebugForceCast:
		handleDebugForceCast(w, payload)
	case message.OpDebugSetPhase:
		handleDebugSetPhase(w, payload)
	case message.OpDebugGodMode:
		handleDebugGodMode(w, peerID, payload)
	case message.OpDebugTimeScale:
		handleDebugTimeScale(w, payload)
	case message.OpDebugResetBoss:
		handleDebugResetBoss(w)
	case message.OpDebugRepeatAbility:
		handleDebugRepeatAbility(w, payload)
	case message.OpDebugReloadYAML:
		handleDebugReloadYAML()
	case message.OpDebugRequestInfo:
		handleDebugRequestInfo(w, peerID)
	}
}

func handleDebugForceCast(w *World, payload []byte) {
	abilityID, ok := codec.DecodeDebugStr8(payload)
	if !ok || abilityID == "" {
		return
	}
	brain, _ := findBossBrain(w)
	if brain == nil {
		return
	}
	if brain.ForceCast(abilityID) {
		slog.Info("debug: force-cast", "ability", abilityID)
	}
}

func handleDebugSetPhase(w *World, payload []byte) {
	phase, ok := codec.DecodeDebugPhase(payload)
	if !ok {
		return
	}
	boss := findBoss(w)
	if boss == nil {
		return
	}
	boss.Phase = int(phase)
	slog.Info("debug: set phase", "phase", phase)
}

func handleDebugGodMode(w *World, peerID uint16, payload []byte) {
	enabled, ok := codec.DecodeDebugGodMode(payload)
	if !ok {
		return
	}
	p, exists := w.Players[peerID]
	if !exists {
		return
	}
	p.GodMode = enabled
	slog.Info("debug: god mode", "peer_id", peerID, "enabled", enabled)
}

func handleDebugTimeScale(w *World, payload []byte) {
	scale, ok := codec.DecodeDebugTimeScale(payload)
	if !ok {
		return
	}
	// Clamp to safe range.
	if scale < 0.1 {
		scale = 0.1
	}
	if scale > 2.0 {
		scale = 2.0
	}
	w.TimeScale = scale
	slog.Info("debug: time scale", "scale", scale)
}

func handleDebugResetBoss(w *World) {
	boss := findBoss(w)
	bossIdx := findBossIndex(w)
	if boss == nil || bossIdx < 0 {
		return
	}

	// Reset boss entity.
	if bossIdx < len(w.Level.EnemySpawns) {
		boss.Reset(w.Level.EnemySpawns[bossIdx].Position, entity.EnemyPatrol)
	}

	// Clear projectiles and gate.
	w.Projectiles = nil
	w.BossGateActive = false
	w.DebugRepeatAbility = ""

	// Rebuild brain from (possibly reloaded) DefRegistry.
	def := enemyai.DefRegistry[boss.DefName]
	if def != nil && bossIdx < len(w.Brains) {
		newBrain := enemyai.NewBrain(def, boss, w.AbilityEngine)
		newBrain.BoundsMinX = w.Level.EnemyBoundsMinX
		newBrain.BoundsMaxX = w.Level.EnemyBoundsMaxX
		newBrain.BoundsMinZ = w.Level.EnemyBoundsMinZ
		newBrain.BoundsMaxZ = w.Level.EnemyBoundsMaxZ
		w.Brains[bossIdx] = newBrain
	}

	// Respawn all players at full HP.
	for _, p := range w.Players {
		p.Alive = true
		p.Health = p.MaxHealth
		p.State = entity.PlayerStateMove
		p.Velocity = entity.Vec3{}
	}

	slog.Info("debug: boss reset")
}

func handleDebugRepeatAbility(w *World, payload []byte) {
	abilityID, ok := codec.DecodeDebugStr8(payload)
	if !ok {
		return
	}
	w.DebugRepeatAbility = abilityID
	if abilityID == "" {
		slog.Info("debug: repeat mode disabled")
	} else {
		slog.Info("debug: repeat mode", "ability", abilityID)
	}
}

func handleDebugReloadYAML() {
	if err := enemyai.LoadMobs(enemyai.MobsDir()); err != nil {
		slog.Error("debug: reload mobs failed", "error", err)
		return
	}
	if err := enemyai.LoadEncounters(enemyai.EncountersDir()); err != nil {
		slog.Error("debug: reload encounters failed", "error", err)
		return
	}
	slog.Info("debug: YAML reloaded")
}

func handleDebugRequestInfo(w *World, peerID uint16) {
	brain, _ := findBossBrain(w)
	if brain == nil {
		return
	}
	payload := codec.EncodeDebugInfo(brain.DefName(), brain.AbilityIDs())
	msg := message.Encode(message.OpDebugInfo, 0, payload)
	if c, ok := w.Clients[peerID]; ok {
		c.Send(msg)
	}
}

// findBossBrain returns the boss BrainTicker and its index, or nil if not found.
func findBossBrain(w *World) (enemyai.BrainTicker, int) {
	for i, e := range w.Enemies {
		if e.IsBoss && i < len(w.Brains) {
			return w.Brains[i], i
		}
	}
	return nil, -1
}
