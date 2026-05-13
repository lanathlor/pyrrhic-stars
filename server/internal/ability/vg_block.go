package ability

import "codex-online/server/internal/entity"

// Block constants
const (
	blockDrainPerSec float32 = 15.0 // stamina drained per second while blocking
	blockDRStart     float32 = 0.2  // damage passthrough at block start (80% DR)
	blockDREnd       float32 = 0.5  // damage passthrough at full decay (50% DR)
	blockDecayTime   float32 = 1.5  // seconds to reach full decay
	blockCooldown    float32 = 3.0  // cooldown after block ends
	blockParryTime   float32 = 0.15 // perfect-block parry window
)

var vgBlockDef = AbilityDef{
	ID: "vg_block", Name: "Block",
	Handler: "vg_block",
}

var vgBlockStopDef = AbilityDef{
	ID: "vg_block_stop", Name: "Block Stop",
	Handler: "vg_block_stop",
}

// VgBlockState tracks sustained block for the tick handler.
type VgBlockState struct {
	Active  bool
	Elapsed float32
}

func vgBlockHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	if p.Cooldowns["vg_block"] > 0 {
		return CastResult{Reason: "cooldown"}
	}
	state := getVgBlockState(p)
	if state.Active {
		return CastResult{Reason: "already blocking"}
	}
	if p.GetResource("stamina") <= 0 {
		return CastResult{Reason: ReasonInsufficientStamina}
	}

	state.Active = true
	state.Elapsed = 0

	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_parry",
		Type:     entity.BuffDamageReduction,
		Value:    0.0,
		Duration: blockParryTime,
	})
	p.AddBuff(entity.ActiveBuff{
		ID:       "vg_block",
		Type:     entity.BuffDamageReduction,
		Value:    blockDRStart,
		Duration: 0, // permanent — managed by tick handler
	})
	p.State = entity.PlayerStateBlock
	return CastResult{OK: true}
}

func vgBlockStopHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "invalid caster"}
	}
	EndVgBlock(p)
	return CastResult{OK: true}
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
	p.RemoveBuff("vg_block")
	p.RemoveBuff("vg_parry")
	p.Cooldowns["vg_block"] = blockCooldown
	if p.State == entity.PlayerStateBlock {
		p.State = entity.PlayerStateMove
	}
}

func vgBlockTick(_ *Engine, p *entity.Player, dt float32, _ *TickContext) []DamageResult {
	state := getVgBlockState(p)
	if !state.Active {
		return nil
	}

	// Drain stamina
	stamina := p.Resources["stamina"]
	if stamina == nil || stamina.Current <= 0 {
		EndVgBlock(p)
		return nil
	}
	stamina.Current -= blockDrainPerSec * dt
	stamina.DelayTimer = stamina.RegenDelay
	if stamina.Current <= 0 {
		stamina.Current = 0
		EndVgBlock(p)
		return nil
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

	return nil
}

func getVgBlockState(p *entity.Player) *VgBlockState {
	if s, ok := p.AbilityState["vg_block"].(*VgBlockState); ok {
		return s
	}
	s := &VgBlockState{}
	p.AbilityState["vg_block"] = s
	return s
}
