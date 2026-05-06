package enemyai

import "codex-online/server/internal/ability"

// PhaseDef defines stat overrides for a specific HP phase.
type PhaseDef struct {
	// Trigger: phase activates when HP ratio <= this value
	HPThresholdPct float32

	// Transition duration (invulnerability)
	TransitionTime float32

	// Movement
	MoveSpeed      float32
	BackpedalSpeed float32 // 0 = use EnemyDef default

	// Cooldown override (0 = use ability's Cooldown)
	CooldownOverride float32

	// Per-ability weight overrides, keyed by AbilityDef.ID
	WeightOverrides map[string]int

	// Per-ability stat overrides, keyed by AbilityDef.ID
	AbilityOverrides map[string]AbilityOverride
}

// AbilityOverride lets a phase modify specific fields of an ability.
// Nil pointers mean "use the base definition."
type AbilityOverride struct {
	CommitTime        *float32 // overrides CommitTime (YAML: telegraph_time)
	Damage            *float32 // overrides BaseDamage, Projectile.Damage, or Charge.Damage
	ProjectileCount   *int
	AoERadius         *float32
	ChargeSpeed       *float32
	ChargeMaxDistance  *float32
	CooldownTime      *float32 // overrides Cooldown
}

// EnemyDef declares a complete enemy type.
type EnemyDef struct {
	Name      string
	MaxHealth float32
	MoveSpeed float32 // base move speed (phase 1)
	Radius    float32 // collision radius

	// Movement behavior
	//   PreferredRange > 0: enemy tries to stay at this distance from players.
	//     Too close → backpedal at BackpedalSpeed. Too far → close in at MoveSpeed.
	//     Still faces target, still needs LoS, still uses melee if cornered.
	//   PreferredRange == 0: classic chase — move toward target until in melee range.
	PreferredRange float32
	BackpedalSpeed float32 // speed when backpedaling (0 = 50% of MoveSpeed)

	// Abilities
	Abilities  []ability.AbilityDef
	AntiRepeat float32 // weight divisor for last-used attack (e.g. 2.0)

	// Phases (sorted by HPThresholdPct descending, e.g. 0.6, 0.3)
	Phases []PhaseDef

	// TreeData holds parsed YAML tree data for data-driven mobs (Tier 1/2).
	// Nil for Go-defined mobs (Tier 3 bosses) which use hardcoded tree builders.
	TreeData any
}

// CurrentPhase returns the PhaseDef for the given phase number, or nil for phase 1.
func (d *EnemyDef) CurrentPhase(phase int) *PhaseDef {
	// Phase 1 has no overrides (base stats)
	if phase <= 1 {
		return nil
	}
	// Simple mapping: phase 2 → index 0, phase 3 → index 1
	idx := phase - 2
	if idx >= 0 && idx < len(d.Phases) {
		return &d.Phases[idx]
	}
	return nil
}

// ResolveAbility returns a copy of the ability with phase overrides applied.
func (d *EnemyDef) ResolveAbility(abil *ability.AbilityDef, phase int) ability.AbilityDef {
	resolved := *abil // copy

	pd := d.CurrentPhase(phase)
	if pd == nil {
		return resolved
	}

	ovr, ok := pd.AbilityOverrides[abil.ID]
	if !ok {
		return resolved
	}

	if ovr.CommitTime != nil {
		resolved.CommitTime = *ovr.CommitTime
	}
	if ovr.CooldownTime != nil {
		resolved.Cooldown = *ovr.CooldownTime
	}
	if ovr.ProjectileCount != nil && resolved.Projectile != nil {
		cp := *resolved.Projectile
		cp.Count = *ovr.ProjectileCount
		resolved.Projectile = &cp
	}
	if ovr.AoERadius != nil {
		resolved.Hit.Radius = *ovr.AoERadius
	}
	if resolved.Charge != nil && (ovr.ChargeSpeed != nil || ovr.ChargeMaxDistance != nil) {
		cp := *resolved.Charge
		if ovr.ChargeSpeed != nil {
			cp.Speed = *ovr.ChargeSpeed
		}
		if ovr.ChargeMaxDistance != nil {
			cp.MaxDistance = *ovr.ChargeMaxDistance
		}
		resolved.Charge = &cp
	}
	if ovr.Damage != nil {
		switch {
		case resolved.Charge != nil:
			// Deep-copy if not already done above
			if ovr.ChargeSpeed == nil && ovr.ChargeMaxDistance == nil {
				cp := *resolved.Charge
				resolved.Charge = &cp
			}
			resolved.Charge.Damage = *ovr.Damage
		case resolved.Projectile != nil:
			// Deep-copy if not already done above
			if ovr.ProjectileCount == nil {
				cp := *resolved.Projectile
				resolved.Projectile = &cp
			}
			resolved.Projectile.Damage = *ovr.Damage
		default:
			resolved.BaseDamage = *ovr.Damage
		}
	}

	return resolved
}

// CurrentMoveSpeed returns the move speed for the given phase.
func (d *EnemyDef) CurrentMoveSpeed(phase int) float32 {
	pd := d.CurrentPhase(phase)
	if pd != nil && pd.MoveSpeed > 0 {
		return pd.MoveSpeed
	}
	return d.MoveSpeed
}

// CurrentBackpedalSpeed returns the backpedal speed for the given phase.
func (d *EnemyDef) CurrentBackpedalSpeed(phase int) float32 {
	pd := d.CurrentPhase(phase)
	if pd != nil && pd.BackpedalSpeed > 0 {
		return pd.BackpedalSpeed
	}
	if d.BackpedalSpeed > 0 {
		return d.BackpedalSpeed
	}
	return d.CurrentMoveSpeed(phase) * 0.5
}

// CurrentCooldownTime returns the cooldown for the given phase/ability.
func (d *EnemyDef) CurrentCooldownTime(abil *ability.AbilityDef, phase int) float32 {
	pd := d.CurrentPhase(phase)
	if pd != nil && pd.CooldownOverride > 0 {
		return pd.CooldownOverride
	}
	return abil.Cooldown
}

// AbilityByIndex returns the ability at the given index, or nil.
func (d *EnemyDef) AbilityByIndex(idx int) *ability.AbilityDef {
	if idx >= 0 && idx < len(d.Abilities) {
		return &d.Abilities[idx]
	}
	return nil
}

// LongestMeleeRange returns the longest melee range among all abilities.
func (d *EnemyDef) LongestMeleeRange() float32 {
	var best float32
	for i := range d.Abilities {
		if d.Abilities[i].Category == ability.CategoryMelee && d.Abilities[i].Hit.Range > best {
			best = d.Abilities[i].Hit.Range
		}
	}
	return best
}

// Helper for creating float32 pointers in phase definitions.
func F32(v float32) *float32 { return &v }

// Helper for creating int pointers in phase definitions.
func Int(v int) *int { return &v }
