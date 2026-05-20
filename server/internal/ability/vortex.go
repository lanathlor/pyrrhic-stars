package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// Vortex — forward advancing spin. Multi-hit along path.
var bladeSwirlDef = AbilityDef{
	ID: "vortex", Name: "Vortex",
	Handler: "vortex",
}

// Vortex tier tuning.
type vortexTuning struct {
	duration float32
	hits     int
}

var vortexTiers = [3]vortexTuning{
	{duration: 0.6, hits: 2}, // standard
	{duration: 0.8, hits: 3}, // empowered
	{duration: 1.0, hits: 4}, // maximum
}

const (
	vortexCooldown    float32 = 10.0
	vortexStaminaCost float32 = 25
	vortexHitRadius   float32 = 4.0
	vortexDamageBase  float32 = 25.0
)

// VortexState tracks the vortex multi-hit for a vanguard.
type VortexState struct {
	Timer     float32
	Duration  float32
	TotalHits int
	HitsDone  int
}

func vortexHandler(eng *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["vortex"] > 0 || p.GCDTimer > 0 {
		return CastResult{Reason: "cooldown"}
	}
	if !p.SpendResource("stamina", vortexStaminaCost*p.TenacityEfficiency()) {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	ons := getOnslaughtState(p)
	tier := ons.Tier()
	tuning := vortexTiers[tier]

	state := getVortexState(p)
	state.Timer = tuning.duration
	state.Duration = tuning.duration
	state.TotalHits = tuning.hits
	state.HitsDone = 0

	p.Cooldowns["vortex"] = vortexCooldown
	p.GCDTimer = tuning.duration
	p.State = entity.PlayerStateAttack

	p.AddBuff(entity.ActiveBuff{
		ID:       "vortex",
		Type:     entity.BuffDamageReduction,
		Value:    0.8,
		Duration: tuning.duration,
	})

	// First hit immediately
	damage := vortexDamageBase * p.CasterDamageMult() * ons.DamageMult(p.GearStats.Mastery)
	eng.hitBuf = resolveAoECircle(eng.hitBuf, p.Position, p.ID, ctx.Targets, ctx.Obstacles, vortexHitRadius, damage, combat.SourcePlayerAttack)
	for i := range eng.hitBuf {
		if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
			th.AddThreat(p.ID, eng.hitBuf[i].Amount)
		}
	}

	if len(eng.hitBuf) > 0 {
		ons.Increment(len(eng.hitBuf))
	}
	state.HitsDone = 1

	return CastResult{OK: true, Events: eng.hitBuf}
}

func vortexTick(eng *Engine, p *entity.Player, dt float32, ctx *TickContext) []DamageResult {
	state := getVortexState(p)
	if state.Timer <= 0 {
		return nil
	}
	state.Timer -= dt

	var events []DamageResult

	// Check if we should deliver the next hit based on evenly-spaced intervals.
	// Hit 0 fires on cast. Hits 1..N-1 fire during ticks at interval spacing.
	if state.HitsDone < state.TotalHits && ctx != nil {
		interval := state.Duration / float32(state.TotalHits)
		elapsed := state.Duration - state.Timer
		// +1 because hit 0 already happened at elapsed=0
		expectedHits := int(elapsed/interval) + 1
		if expectedHits > state.TotalHits {
			expectedHits = state.TotalHits
		}

		if expectedHits > state.HitsDone {
			ons := getOnslaughtState(p)
			tickDmg := vortexDamageBase * p.CasterDamageMult() * ons.DamageMult(p.GearStats.Mastery)
			eng.hitBuf = resolveAoECircle(eng.hitBuf[:0], p.Position, p.ID, ctx.Targets, ctx.Obstacles, vortexHitRadius, tickDmg, combat.SourcePlayerAttack)
			for i := range eng.hitBuf {
				if th, ok := eng.hitBuf[i].Target.(entity.Threateable); ok {
					th.AddThreat(p.ID, eng.hitBuf[i].Amount)
				}
			}
			if len(eng.hitBuf) > 0 {
				ons.Increment(len(eng.hitBuf))
			}
			events = append(events, eng.hitBuf...)
			state.HitsDone = expectedHits
		}
	}

	if state.Timer <= 0 {
		state.Timer = 0
		state.HitsDone = 0
	}

	return events
}

func getVortexState(p *entity.Player) *VortexState {
	if s, ok := p.AbilityState["vortex"].(*VortexState); ok {
		return s
	}
	s := &VortexState{}
	p.AbilityState["vortex"] = s
	return s
}
