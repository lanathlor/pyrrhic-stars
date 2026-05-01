package ability

var groundSlamDef = AbilityDef{
	ID:              "ground_slam", Name: "Ground Slam",
	Hit:             HitDef{Type: HitAoECone, Range: 7, ArcDegrees: 90},
	BaseDamage:      60,
	Cooldown:        8.0,
	LockoutDuration: 1.2,
	Costs:           []ResourceCost{{Resource: "stamina", Amount: 20}},
}
