package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Shield Block constants.
const (
	shieldBlockDR              float32 = 0.15  // damage passthrough (85% DR, constant)
	shieldBlockCooldown        float32 = 1.5   // cooldown after block ends
	shieldGuardParryWindow     float32 = 0.12  // tighter than Blade's 0.15
	guardParryReflectFraction  float32 = 0.3   // 30% of blocked damage reflected
	guardParryReflectRange     float32 = 6.0
	guardParryReflectArc       float32 = 120.0
	guardBreakVulnMult         float32 = 1.25  // 25% increased damage taken
	guardBreakDuration         float32 = 1.5
	ShieldStaminaDrainFraction float32 = 0.5 // 50% of pre-DR damage drains stamina (exported for entity pkg)
)

var vgShieldBlockDef = AbilityDef{
	ID: "vg_shield_block", Name: "Shield Block",
	Handler: "vg_shield_block",
}

var vgShieldBlockStopDef = AbilityDef{
	ID: "vg_shield_block_stop", Name: "Shield Block Stop",
	Handler: "vg_shield_block_stop",
}

// VgShieldBlockState tracks sustained shield block for the tick handler.
type VgShieldBlockState struct {
	Active              bool
	Elapsed             float32
	ParryReflectPending bool
	ParryReflectDamage  float32 // damage to reflect on next tick
}

// SetParryReflectPending marks a reflect as pending (called from player.ApplyDamage).
func (s *VgShieldBlockState) SetParryReflectPending(dmg float32) {
	s.ParryReflectPending = true
	s.ParryReflectDamage += dmg
}

func vgShieldBlockHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["vg_shield_block"] > 0 {
		return CastResult{Reason: "cooldown"}
	}
	state := getVgShieldBlockState(p)
	if state.Active {
		return CastResult{Reason: "already blocking"}
	}
	if p.GetResource("stamina") <= 0 {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	state.Active = true
	state.Elapsed = 0
	state.ParryReflectPending = false
	state.ParryReflectDamage = 0

	// Ensure Devotion state exists for ApplyDamage integration
	getDevotionState(p)

	// Guard Parry window — tighter than Blade's
	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_shield_parry",
		Type:     entity.BuffDamageReduction,
		Value:    0.0,
		Duration: shieldGuardParryWindow * p.TempoMult(),
	})
	// Sustained block DR — constant, does not decay
	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_shield_block",
		Type:     entity.BuffDamageReduction,
		Value:    shieldBlockDR,
		Duration: 0, // permanent — managed by tick handler
	})
	p.State = entity.PlayerStateBlock
	return CastResult{OK: true}
}

func vgShieldBlockStopHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	EndVgShieldBlock(p)
	return CastResult{OK: true}
}

// EndVgShieldBlock cleanly ends a shield block, removing buffs and setting cooldown.
func EndVgShieldBlock(p *entity.Player) {
	state := getVgShieldBlockState(p)
	if !state.Active {
		return
	}
	state.Active = false
	state.Elapsed = 0
	state.ParryReflectPending = false
	state.ParryReflectDamage = 0
	p.RemoveBuff("vg_shield_block")
	p.RemoveBuff("vg_shield_parry")
	p.RemoveBuff("brace")
	p.Cooldowns["vg_shield_block"] = shieldBlockCooldown
	if p.State == entity.PlayerStateBlock {
		p.State = entity.PlayerStateMove
	}
}

func vgShieldBlockTick(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	state := getVgShieldBlockState(p)
	if !state.Active {
		return nil
	}

	var events []DamageResult

	// Guard Parry reflect: resolve pending reflected damage
	if state.ParryReflectPending && ctx != nil {
		reflectDmg := state.ParryReflectDamage * guardParryReflectFraction * p.CasterDamageMult()
		state.ParryReflectPending = false
		state.ParryReflectDamage = 0

		hit := HitDef{Type: HitMeleeArc, Range: guardParryReflectRange, ArcDegrees: guardParryReflectArc}
		eng.hitBuf = resolveMeleeArc(eng.hitBuf[:0], p, ctx.Targets, ctx.Obstacles, hit, reflectDmg, combat.SourcePlayerAttack)

		for i := range eng.hitBuf {
			if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
				th.AddThreat(p.ID, eng.hitBuf[i].Amount*1.5) // bonus threat from parry reflect
			}
		}
		events = append(events, eng.hitBuf...)
	}

	// Check stamina — Guard Break if depleted
	stamina := p.Resources["stamina"]
	if stamina == nil || stamina.Current <= 0 {
		triggerGuardBreak(p)
		return events
	}

	// No per-second stamina drain for Shield block (unlike Blade).
	// Stamina is drained proportionally to damage absorbed in ApplyDamage.

	state.Elapsed += dt
	return events
}

// triggerGuardBreak ends block and applies a stagger + vulnerability debuff.
func triggerGuardBreak(p *entity.Player) {
	EndVgShieldBlock(p)
	p.State = entity.PlayerStateStagger
	p.AddBuff(entity.ActiveBuff{
		ID:       "guard_break",
		Type:     entity.BuffDamageReduction,
		Value:    guardBreakVulnMult, // >1.0 = increased damage taken
		Duration: guardBreakDuration,
	})
	p.GCDTimer = guardBreakDuration
}

func getVgShieldBlockState(p *entity.Player) *VgShieldBlockState {
	if s, ok := p.AbilityState["vg_shield_block"].(*VgShieldBlockState); ok {
		return s
	}
	s := &VgShieldBlockState{}
	p.AbilityState["vg_shield_block"] = s
	return s
}
