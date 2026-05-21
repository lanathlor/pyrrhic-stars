package ability

import "codex-online/server/internal/entity"

// PlayerRunnerPhase tracks where a player is in the ability lifecycle.
type PlayerRunnerPhase uint8

const (
	PRunnerIdle     PlayerRunnerPhase = iota
	PRunnerCommit                     // channeling / wind-up
	PRunnerExecute                    // ability fires
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

// PlayerAbilityRunner owns the Idle->Commit->Execute->Cooldown lifecycle for
// a single player. The zone tick loop calls Tick each frame; the combat system
// calls Start/Cancel in response to player input.
type PlayerAbilityRunner struct {
	Phase           PlayerRunnerPhase
	AbilityID       string
	Def             *AbilityDef
	Timer           float32
	Charge          float32 // 0->1 normalized during commit
	TotalCommitTime float32
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

// Cancel aborts the current ability if in commit phase.
func (r *PlayerAbilityRunner) Cancel() bool {
	if r.Phase != PRunnerCommit {
		return false
	}
	r.reset()
	return true
}

// Tick advances the ability lifecycle by dt seconds. Returns true on the tick
// that the commit phase expires and execute begins (the "fire" signal).
func (r *PlayerAbilityRunner) Tick(dt float32) bool {
	switch r.Phase {
	case PRunnerIdle:
		return false
	case PRunnerCommit:
		return r.tickCommit(dt)
	case PRunnerExecute:
		return r.tickExecute(dt)
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
		r.enterCooldown()
	}
	return false
}

func (r *PlayerAbilityRunner) tickCooldown(dt float32) {
	r.Timer -= dt
	if r.Timer <= 0 {
		r.reset()
	}
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

// reset clears the runner back to idle.
func (r *PlayerAbilityRunner) reset() {
	r.Phase = PRunnerIdle
	r.AbilityID = ""
	r.Def = nil
	r.Timer = 0
	r.Charge = 0
	r.TotalCommitTime = 0
}
