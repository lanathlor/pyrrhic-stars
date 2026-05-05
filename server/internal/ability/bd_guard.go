package ability

import "codex-online/server/internal/entity"

var bdGuardDef = AbilityDef{
	ID: "bd_guard", Name: "BD Guard",
	Hit: HitDef{Type: HitNone},
	SelfBuffs: []BuffEffect{{
		ID:       "guard",
		Type:     entity.BuffDamageReduction,
		Value:    0.5,
		Duration: 1.5,
	}},
}
