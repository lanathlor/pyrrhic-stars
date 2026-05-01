package ability

var fireShotDef = AbilityDef{
	ID: "fire_shot", Name: "Fire Shot",
	Hit:        HitDef{Type: HitHitscan, Range: 100},
	BaseDamage: 10,
	Cooldown:   0.18,
}
