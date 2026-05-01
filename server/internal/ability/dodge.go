package ability

var dodgeDef = AbilityDef{
	ID: "dodge", Name: "Dodge",
	Hit:   HitDef{Type: HitNone},
	Costs: []ResourceCost{{Resource: "stamina", Amount: 20}},
}
