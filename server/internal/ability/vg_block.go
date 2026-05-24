package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Block constants
const (
	blockDrainPerSec float32 = 15.0 // stamina drained per second while blocking
	blockDRStart     float32 = 0.2  // damage passthrough at block start (80% DR)
	blockDREnd       float32 = 0.5  // damage passthrough at full decay (50% DR)
	blockDecayTime   float32 = 1.5  // seconds to reach full decay
	blockCooldown    float32 = 3.0  // cooldown after block ends
	blockParryTime   float32 = 0.15 // perfect-block parry window

	parryCounterDamage float32 = 40.0 // damage dealt by parry counter-swing
	parryCounterArc    float32 = 120.0
	parryCounterRange  float32 = 6.0
)

var vgBlockDef = AbilityDef{
	ID: "vg_block", Name: "Blade Parry",
	Handler: "vg_block",
}

var vgBlockStopDef = AbilityDef{
	ID: "vg_block_stop", Name: "Block Stop",
	Handler: "vg_block_stop",
}

// VgBlockState tracks sustained block for the tick handler.
type VgBlockState struct {
	Active              bool
	Elapsed             float32
	ParryCounterPending bool // set when parry absorbs a hit — resolved on next tick
}

// SetParryPending marks a counter-swing as pending (called from player.ApplyDamage).
func (s *VgBlockState) SetParryPending() {
	s.ParryCounterPending = true
}

func vgBlockHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["vg_block"] > 0 {
		return CommitResult{Reason: "cooldown"}
	}
	state := getVgBlockState(p)
	if state.Active {
		return CommitResult{Reason: "already blocking"}
	}
	if p.GetResource("stamina") <= 0 {
		return CommitResult{Reason: ReasonInsufficientStamina}
	}

	state.Active = true
	state.Elapsed = 0
	state.ParryCounterPending = false

	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_parry",
		Type:     entity.BuffDamageReduction,
		Value:    0.0,
		Duration: blockParryTime * p.TempoMult(),
	})
	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_block",
		Type:     entity.BuffDamageReduction,
		Value:    blockDRStart,
		Duration: 0, // permanent — managed by tick handler
	})
	p.State = entity.PlayerStateBlock
	return CommitResult{OK: true}
}

func vgBlockStopHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	EndVgBlock(p)
	return CommitResult{OK: true}
}

// EndVgBlock cleanly ends a vanguard block, removing buffs and setting cooldown.
// Exported so input.go can call it when another ability cancels block.
func EndVgBlock(p *entity.Player) {
	state := getVgBlockState(p)
	if !state.Active {
		return
	}
	state.Active = false
	state.Elapsed = 0
	state.ParryCounterPending = false
	p.RemoveBuff("vg_block")
	p.RemoveBuff("vg_parry")
	p.Cooldowns["vg_block"] = blockCooldown
	if p.State == entity.PlayerStateBlock {
		p.State = entity.PlayerStateMove
	}
}

func vgBlockTick(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	state := getVgBlockState(p)
	if !state.Active {
		return nil
	}

	var events []DamageResult

	// Blade Parry counter-swing: resolve pending parry hit
	if state.ParryCounterPending && ctx != nil {
		state.ParryCounterPending = false
		damage := parryCounterDamage * p.CommitterDamageMult()
		hit := HitDef{Type: HitMeleeArc, Range: parryCounterRange, ArcDegrees: parryCounterArc}
		eng.hitBuf = resolveMeleeArc(eng.hitBuf[:0], p, ctx.Targets, ctx.Obstacles, hit, damage, combat.SourcePlayerAttack)

		// Parry counter-swing builds Onslaught stacks
		if len(eng.hitBuf) > 0 {
			ons := getOnslaughtState(p)
			ons.Increment(len(eng.hitBuf))
		}

		for i := range eng.hitBuf {
			if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
				th.AddThreat(p.ID, eng.hitBuf[i].Amount)
			}
		}
		events = append(events, eng.hitBuf...)
	}

	// Drain stamina
	stamina := p.Resources["stamina"]
	if stamina == nil || stamina.Current <= 0 {
		EndVgBlock(p)
		return events
	}
	stamina.Current -= blockDrainPerSec * dt * p.TenacityEfficiency()
	stamina.DelayTimer = stamina.RegenDelay
	if stamina.Current <= 0 {
		stamina.Current = 0
		EndVgBlock(p)
		return events
	}

	// Advance elapsed and decay DR
	state.Elapsed += dt
	if b := p.GetBuff("vg_block"); b != nil {
		v := blockDRStart + (blockDREnd-blockDRStart)*(state.Elapsed/blockDecayTime)
		if v > blockDREnd {
			v = blockDREnd
		}
		b.Value = v
	}

	return events
}

func getVgBlockState(p *entity.Player) *VgBlockState {
	if s, ok := p.AbilityState["vg_block"].(*VgBlockState); ok {
		return s
	}
	s := &VgBlockState{}
	p.AbilityState["vg_block"] = s
	return s
}
