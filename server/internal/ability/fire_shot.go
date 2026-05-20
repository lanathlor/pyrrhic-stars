package ability

// fireShotDef uses a custom handler for magazine, stability, and pressure.
var fireShotDef = AbilityDef{
	ID: "fire_shot", Name: "Fire Shot",
	Handler:    "fire_shot_assault",
	Hit:        HitDef{Type: HitHitscan},
	BaseDamage: 10,
	Cooldown:   0.18,
}
