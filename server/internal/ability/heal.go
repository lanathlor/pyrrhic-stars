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
	target := resolveHealTarget(def, caster, allies, targetPeerID)
	if target == nil {
		return nil
	}
	return applyHeal(def, caster, target)
}

// applyHeal scales the base heal amount with caster stats and passive bonuses,
// applies it to the target, and returns the populated HealResult.
func applyHeal(def *AbilityDef, caster *entity.Player, target *entity.Player) *HealResult {
	heal := scaleHealAmount(def, caster, target)

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

// scaleHealAmount computes the final scaled heal value before application.
func scaleHealAmount(def *AbilityDef, caster *entity.Player, target *entity.Player) float32 {
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

	return heal
}

// resolveHealTarget selects the heal recipient based on the ability's HitType.
// Returns nil when no valid target exists (unknown hit type or no injured allies).
func resolveHealTarget(def *AbilityDef, caster *entity.Player, allies map[uint16]*entity.Player, targetPeerID uint16) *entity.Player {
	switch def.Hit.Type {
	case HitAllyTarget:
		if t, ok := allies[targetPeerID]; ok && t.Alive && t.ID != caster.ID {
			return t
		}
		return caster // fallback: heal self

	case HitAllyLowestHP:
		var lowestHP float32 = 999999
		var target *entity.Player
		for _, p := range allies {
			if p.Alive && p.Health < p.MaxHealth && p.Health < lowestHP {
				lowestHP = p.Health
				target = p
			}
		}
		if target == nil {
			return caster
		}
		return target

	case HitAllyRandom:
		var injured []*entity.Player
		for _, p := range allies {
			if p.Alive && p.Health < p.MaxHealth {
				injured = append(injured, p)
			}
		}
		if len(injured) > 0 {
			return injured[int(caster.ID)%len(injured)]
		}
		return caster

	default:
		return nil
	}
}
