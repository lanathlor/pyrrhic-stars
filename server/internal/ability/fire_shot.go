package ability

import "codex-online/server/internal/entity"

var fireShotDef = AbilityDef{
	ID: "fire_shot", Name: "Fire Shot",
	Hit:        HitDef{Type: HitHitscan, Range: 100},
	BaseDamage: 10,
	Cooldown:   0.18,
}

// Enhanced round constants.
const (
	enhancedRoundProcEvery = 5   // proc on every Nth fire_shot hit
	enhancedRoundBaseDmg   = 10.0 // base bonus damage at Identity=0
)

// FireShotState tracks the enhanced round hit counter for gunner.
type FireShotState struct {
	HitCount int
}

func getFireShotState(p *entity.Player) *FireShotState {
	if s, ok := p.AbilityState["fire_shot"].(*FireShotState); ok {
		return s
	}
	s := &FireShotState{}
	p.AbilityState["fire_shot"] = s
	return s
}
