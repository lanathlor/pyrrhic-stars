package ability

import "codex-online/server/internal/entity"

// PlayerRunnerPhase tracks where a player is in the ability lifecycle.
type PlayerRunnerPhase uint8

const (
	PRunnerIdle     PlayerRunnerPhase = iota
	PRunnerCommit                     // channeling / wind-up
	PRunnerExecute                    // ability fires
	PRunnerSustain                    // extended channel after execute (Arcanotechnicien)
	PRunnerCooldown                   // recovery
)

// CancelCondition is a bitmask for events that cancel an ability during commit.
type CancelCondition uint8

const (
	CancelNone     CancelCondition = 0
	CancelOnMove   CancelCondition = 1 << iota
	CancelOnDamage
	CancelOnInput
)

// PlayerAbilityRunner owns the Idle->Commit->Execute->Sustain->Cooldown lifecycle for
// a single player. The zone tick loop calls Tick each frame; the combat system
// calls Start/Cancel in response to player input.
type PlayerAbilityRunner struct {
	Phase           PlayerRunnerPhase
	AbilityID       string
	Def             *AbilityDef
	Timer           float32
	Charge          float32 // 0->1 normalized during commit; scaling multiplier during sustain
	TotalCommitTime float32

	// Sustain state
	SustainElapsed   float32     // total time in sustain (for scaling)
	SustainTickTimer float32     // countdown to next sustain tick
	SustainStartPos  entity.Vec3 // position when sustain began (for CancelOnMove)
	SustainStartTick uint32      // tick when sustain began (for CancelOnDamage)
}

// Start initiates an ability. Returns true if accepted (runner was idle).
func (r *PlayerAbilityRunner) Start(def *AbilityDef) bool {
	if r.Phase != PRunnerIdle {
		return false
	}
	r.Phase = PRunnerCommit
	r.AbilityID = def.ID
	r.Def = def
	r.Timer = def.CommitTime
	r.TotalCommitTime = def.CommitTime
	r.Charge = 0
	return true
}

// StartSustain begins the sustain phase directly (for instant + sustain abilities).
func (r *PlayerAbilityRunner) StartSustain(def *AbilityDef, pos entity.Vec3, tick uint32) {
	r.Phase = PRunnerSustain
	r.AbilityID = def.ID
	r.Def = def
	r.Timer = 0
	r.Charge = 1.0
	r.TotalCommitTime = 0
	r.SustainElapsed = 0
	r.SustainTickTimer = def.SustainInterval
	r.SustainStartPos = pos
	r.SustainStartTick = tick
}

// Cancel aborts the current ability if in commit or sustain phase.
func (r *PlayerAbilityRunner) Cancel() bool {
	if r.Phase == PRunnerCommit || r.Phase == PRunnerSustain {
		if r.Phase == PRunnerSustain {
			// Sustain ended — enter cooldown instead of reset
			r.enterCooldown()
			return true
		}
		r.reset()
		return true
	}
	return false
}

// Tick advances the ability lifecycle by dt seconds. Returns true on the tick
// that the commit phase expires (fire signal) or a sustain tick fires.
func (r *PlayerAbilityRunner) Tick(dt float32) bool {
	switch r.Phase {
	case PRunnerIdle:
		return false
	case PRunnerCommit:
		return r.tickCommit(dt)
	case PRunnerExecute:
		return r.tickExecute(dt)
	case PRunnerSustain:
		return r.tickSustain(dt)
	case PRunnerCooldown:
		r.tickCooldown(dt)
		return false
	}
	return false
}

func (r *PlayerAbilityRunner) tickCommit(dt float32) bool {
	r.Timer -= dt
	if r.TotalCommitTime > 0 {
		r.Charge = 1.0 - (r.Timer / r.TotalCommitTime)
		if r.Charge > 1.0 {
			r.Charge = 1.0
		}
	}
	if r.Timer <= 0 {
		r.Phase = PRunnerExecute
		r.Timer = r.Def.ExecuteTime
		r.Charge = 1.0
		return true // signal: execute now
	}
	return false
}

func (r *PlayerAbilityRunner) tickExecute(dt float32) bool {
	r.Timer -= dt
	if r.Timer <= 0 {
		if r.Def != nil && r.Def.Sustain {
			r.enterSustain()
		} else {
			r.enterCooldown()
		}
	}
	return false
}

// tickSustain advances the sustain phase. Returns true each time a sustain tick fires.
func (r *PlayerAbilityRunner) tickSustain(dt float32) bool {
	r.SustainElapsed += dt
	r.SustainTickTimer -= dt
	// Update charge to reflect scaling multiplier (for client VFX intensity)
	if r.Def != nil {
		r.Charge = 1.0 + r.SustainElapsed*r.Def.SustainScaling
	}
	if r.SustainTickTimer <= 0 {
		r.SustainTickTimer += r.Def.SustainInterval
		return true // signal: apply sustain effect
	}
	return false
}

func (r *PlayerAbilityRunner) tickCooldown(dt float32) {
	r.Timer -= dt
	if r.Timer <= 0 {
		r.reset()
	}
}

// enterSustain transitions from execute to the sustain phase.
func (r *PlayerAbilityRunner) enterSustain() {
	r.Phase = PRunnerSustain
	r.Timer = 0
	r.SustainElapsed = 0
	r.SustainTickTimer = r.Def.SustainInterval
	r.Charge = 1.0
	// SustainStartPos and SustainStartTick are set by the combat system
	// after entering sustain (it has access to player position and tick).
}

// enterCooldown transitions to the cooldown phase with a short GCD.
func (r *PlayerAbilityRunner) enterCooldown() {
	r.Phase = PRunnerCooldown
	r.Timer = 0.3
}

// IsBusy returns true if the runner is in any phase other than idle.
func (r *PlayerAbilityRunner) IsBusy() bool {
	return r.Phase != PRunnerIdle
}

// SyncToPlayer writes runner state to player fields for wire serialization.
func (r *PlayerAbilityRunner) SyncToPlayer(p *entity.Player) {
	p.ChannelAbilityID = r.AbilityID
	p.ChannelTimer = r.Timer
	p.ChannelCharge = r.Charge
	p.ChannelPhase = uint8(r.Phase)
}

// SustainCooldownOnCancel returns the sustain cooldown to apply when cancelling
// a sustain phase. Returns 0 if not in sustain or no cooldown defined.
// Must be called BEFORE Cancel() since Cancel changes the phase.
func (r *PlayerAbilityRunner) SustainCooldownOnCancel() (abilityID string, cooldown float32) {
	if r.Phase == PRunnerSustain && r.Def != nil && r.Def.SustainCooldown > 0 {
		return r.AbilityID, r.Def.SustainCooldown
	}
	return "", 0
}

// ForceReset clears the runner to idle, bypassing phase checks.
// Used when the caller needs to start a new ability immediately after cancel.
func (r *PlayerAbilityRunner) ForceReset() {
	r.reset()
}

// reset clears the runner back to idle.
func (r *PlayerAbilityRunner) reset() {
	r.Phase = PRunnerIdle
	r.AbilityID = ""
	r.Def = nil
	r.Timer = 0
	r.Charge = 0
	r.TotalCommitTime = 0
	r.SustainElapsed = 0
	r.SustainTickTimer = 0
	r.SustainStartPos = entity.Vec3{}
	r.SustainStartTick = 0
}
