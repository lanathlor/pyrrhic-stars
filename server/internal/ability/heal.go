package ability

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// resolveHeal selects an ally target and applies healing from the ability definition.
// Returns nil if the ability has no BaseHeal or no valid target is found.
// Always returns a result for valid targets, even if the heal is 100% overheal.
func resolveHeal(def *AbilityDef, caster *entity.Player, allies map[uint16]*entity.Player, targetPeerID uint16) *HealResult {
	if def.BaseHeal <= 0 {
		return nil
	}

	var target *entity.Player
	switch def.Hit.Type {
	case HitAllyTarget:
		if t, ok := allies[targetPeerID]; ok && t.Alive && t.ID != caster.ID {
			target = t
		}
		if target == nil {
			target = caster // fallback: heal self
		}
	case HitAllyLowestHP:
		var lowestHP float32 = 999999
		for _, p := range allies {
			if p.Alive && p.Health < p.MaxHealth && p.Health < lowestHP {
				lowestHP = p.Health
				target = p
			}
		}
		if target == nil {
			target = caster
		}
	case HitAllyRandom:
		var injured []*entity.Player
		for _, p := range allies {
			if p.Alive && p.Health < p.MaxHealth {
				injured = append(injured, p)
			}
		}
		if len(injured) > 0 {
			target = injured[int(caster.ID)%len(injured)]
		} else {
			target = caster
		}
	default:
		return nil
	}

	if target == nil || !target.Alive {
		return nil
	}

	heal := def.BaseHeal
	heal *= (1.0 + caster.GearStats.Identity/100.0)

	// Confluence: Arcanotechnicien class-wide ability power bonus.
	if caster.Confluence != nil {
		heal *= caster.Confluence.AbilityPowerMult()
	}

	// VitalCharge: consume stored drain from Life Swap to empower this heal.
	if caster.VitalCharge > 0 && caster.VitalChargeTimer > 0 {
		heal += caster.VitalCharge
		caster.VitalCharge = 0
		caster.VitalChargeTimer = 0
	}

	// Sympathetic Field: Harmonist passive aura amplifies healing for
	// allies inside the field radius (15% bonus).
	if r := caster.SympatheticFieldRadius(); r > 0 {
		dx := caster.Position.X - target.Position.X
		dz := caster.Position.Z - target.Position.Z
		if dx*dx+dz*dz <= r*r {
			heal *= 1.15
		}
	}

	before := target.Health
	target.Health += heal
	if target.Health > target.MaxHealth {
		target.Health = target.MaxHealth
	}
	actual := target.Health - before
	overheal := heal - actual

	// Harmony: bonus heal when delivery method differs from last heal on target.
	deliveryMethod := entity.DeliveryMethod(def.Delivery)
	harmonyBonus := CheckHarmony(caster, target.ID, deliveryMethod)
	var harmonyProc bool
	var harmonyAmount float32
	if harmonyBonus > 0 {
		harmonyProc = true
		beforeBonus := target.Health
		target.Health += harmonyBonus
		if target.Health > target.MaxHealth {
			target.Health = target.MaxHealth
		}
		harmonyAmount = target.Health - beforeBonus
		actual += harmonyAmount
		overheal += harmonyBonus - harmonyAmount
	}

	return &HealResult{
		TargetID:      target.ID,
		SourceID:      caster.ID,
		Amount:        actual,
		Overheal:      overheal,
		HitPos:        target.Position.Add(entity.Vec3{Y: 1.0}),
		SourceType:    combat.SourcePlayerHeal,
		HarmonyProc:   harmonyProc,
		HarmonyAmount: harmonyAmount,
	}
}
