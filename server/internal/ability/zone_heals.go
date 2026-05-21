package ability

import "codex-online/server/internal/entity"

var vitalBloomDef = AbilityDef{
	ID:     "vital_bloom",
	Name:   "Vital Bloom",
	School: "biometabolic",
	Hit:  HitDef{Type: HitGroundPlacement, Range: 15},
	GCD:  1.0,
	Costs: []ResourceCost{
		{Resource: "flux", Amount: 8},
	},
	ZoneRadius:   4.0,
	ZoneDuration: 8.0,
	ZoneInterval: 1.0,
	Delivery:     uint8(entity.DeliveryZone),
	Handler:      "vital_bloom",
}

var restorationMatrixDef = AbilityDef{
	ID:       "restoration_matrix",
	Name:     "Restoration Matrix",
	School:   "bioarcanotechnic",
	Hit:      HitDef{Type: HitGroundPlacement, Range: 18},
	GCD:      1.0,
	Cooldown: 12.0,
	Costs: []ResourceCost{
		{Resource: "flux", Amount: 50},
	},
	ZoneRadius:   5.0,
	ZoneDuration: 10.0,
	ZoneHealTick: 12,
	ZoneInterval: 1.0,
	Delivery:     uint8(entity.DeliveryZone),
	Handler:      "restoration_matrix",
}

func vitalBloomHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "not a player"}
	}

	sacrifice := p.Health * 0.15
	if p.Health-sacrifice < 1 {
		sacrifice = p.Health - 1
	}
	if sacrifice <= 0 {
		return CastResult{Reason: "too low HP"}
	}

	p.Health -= sacrifice
	p.SpendResource("flux", vitalBloomDef.Costs[0].Amount)

	// Confluence: grant stack on spell completion.
	if p.Confluence != nil {
		p.Confluence.OnSpellComplete()
	}

	healPerTick := sacrifice * 0.3

	zonePos := p.Position

	if ctx.SpawnZone != nil {
		ctx.SpawnZone(&entity.HealingZone{
			OwnerID:     p.ID,
			Position:    zonePos,
			Radius:      vitalBloomDef.ZoneRadius,
			HealPerTick: healPerTick,
			Duration:    vitalBloomDef.ZoneDuration,
			TickTimer:   vitalBloomDef.ZoneInterval,
			Interval:    vitalBloomDef.ZoneInterval,
			AbilityID:   "vital_bloom",
		})
	}

	p.GCDTimer = vitalBloomDef.GCD
	return CastResult{OK: true}
}

func restorationMatrixHandler(_ *Engine, ctx *CastContext) CastResult {
	p, ok := ctx.Caster.(*entity.Player)
	if !ok {
		return CastResult{Reason: "not a player"}
	}

	p.SpendResource("flux", restorationMatrixDef.Costs[0].Amount)

	// Confluence: grant stack on spell completion.
	if p.Confluence != nil {
		p.Confluence.OnSpellComplete()
	}

	zonePos := p.Position

	if ctx.SpawnZone != nil {
		ctx.SpawnZone(&entity.HealingZone{
			OwnerID:     p.ID,
			Position:    zonePos,
			Radius:      restorationMatrixDef.ZoneRadius,
			HealPerTick: restorationMatrixDef.ZoneHealTick * (1 + p.GearStats.Identity/100),
			Duration:    restorationMatrixDef.ZoneDuration,
			TickTimer:   restorationMatrixDef.ZoneInterval,
			Interval:    restorationMatrixDef.ZoneInterval,
			AbilityID:   "restoration_matrix",
		})
	}

	p.GCDTimer = restorationMatrixDef.GCD
	if restorationMatrixDef.Cooldown > 0 {
		p.Cooldowns["restoration_matrix"] = restorationMatrixDef.Cooldown
	}
	return CastResult{OK: true}
}
