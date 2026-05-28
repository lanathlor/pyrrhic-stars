package ability

import "codex-online/server/internal/entity"

var dodgeDef = AbilityDef{
	ID: IDDodge, Name: "Dodge",
	Hit:   HitDef{Type: HitNone},
	Costs: []ResourceCost{{Resource: entity.ResourceStamina, Amount: 20}},
}
