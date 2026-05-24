package ability

import "codex-online/server/internal/entity"

var rechamberDef = AbilityDef{
	ID: "rechamber", Name: "Rechamber",
	Handler: "rechamber",
}

var rechamberConfirmDef = AbilityDef{
	ID: "rechamber_confirm", Name: "Rechamber Confirm",
	Handler: "rechamber_confirm",
}

// RechamberState tracks the rechamber phase machine for a gunner.
type RechamberState struct {
	Phase uint8 // 0=idle, 1=windup, 2=timing_window, 3=lockout
	Timer float32
}

// GetPhase implements the phaser interface for network encoding.
func (s *RechamberState) GetPhase() uint8 { return s.Phase }

func rechamberHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	state := getRechamberState(p)
	if state.Phase != 0 {
		return CommitResult{Reason: "rechamber in progress"}
	}
	if cd := p.Cooldowns["fire_shot"]; cd > 0 {
		return CommitResult{Reason: "fire cooldown"}
	}
	state.Phase = 1
	state.Timer = 0.6
	p.Cooldowns["fire_shot"] = 0.6
	return CommitResult{OK: true}
}

func rechamberConfirmHandler(_ *Engine, ctx *CommitContext) CommitResult {
	p, ok := ctx.Committer.(*entity.Player)
	if !ok {
		return CommitResult{Reason: "invalid caster"}
	}
	state := getRechamberState(p)
	if state.Phase != 2 {
		return CommitResult{Reason: "not in timing window"}
	}
	p.AddBuff(entity.ActiveBuff{
		ID:       "rechamber_buff",
		Type:     entity.BuffDamageMult,
		Value:    1.8,
		Duration: 4.0,
	})
	state.Phase = 0
	state.Timer = 0
	return CommitResult{OK: true}
}

func rechamberTick(_ *Engine, p *entity.Player, dt float32, _ *TickContext) []DamageResult {
	state := getRechamberState(p)
	if state.Phase == 0 {
		return nil
	}
	state.Timer -= dt * p.TempoMult()
	switch state.Phase {
	case 1:
		if state.Timer <= 0 {
			state.Phase = 2
			state.Timer = 0.35
		}
	case 2:
		if state.Timer <= 0 {
			state.Phase = 3
			state.Timer = 0.8
		}
	case 3:
		if state.Timer <= 0 {
			state.Phase = 0
			state.Timer = 0
		}
	}
	return nil
}

func getRechamberState(p *entity.Player) *RechamberState {
	if s, ok := p.AbilityState["rechamber"].(*RechamberState); ok {
		return s
	}
	s := &RechamberState{}
	p.AbilityState["rechamber"] = s
	return s
}
