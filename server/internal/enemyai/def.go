package enemyai


// AbilityType identifies how an ability's damage is resolved.
type AbilityType uint8

const (
	AbilityMelee  AbilityType = iota // arc check, hit all in range
	AbilityRanged                     // spawn projectiles toward target
	AbilityAoE                        // radius check centered on enemy
	AbilityCharge                     // linear dash, hit along path
)

// TargetStrategy controls who the ability targets.
type TargetStrategy uint8

const (
	TargetNearest  TargetStrategy = iota // nearest alive player
	TargetFarthest                        // farthest alive player
	TargetCurrent                         // keep current chase target
)

// AbilityDef declares a single enemy ability.
type AbilityDef struct {
	Name string // unique identifier, e.g. "melee_swipe"

	Type           AbilityType
	TargetStrategy TargetStrategy

	// Timing
	TelegraphTime float32 // seconds of warning before execution
	ExecuteTime   float32 // duration of execution window (0 = runs until stop condition)
	CooldownTime  float32 // post-attack cooldown

	// Selection
	BaseWeight int     // base weight for weighted random
	MinRange   float32 // don't use if target closer than this
	MaxRange   float32 // don't use if target farther (0 = unlimited)

	// Telegraph behavior
	FaceTarget  bool // rotate to face target during telegraph
	TrackTarget bool // continuously update target position during telegraph

	// Melee-specific
	MeleeRange     float32
	MeleeDamage    float32
	MeleeConeAngle float32 // full cone angle in radians (0 = default 180°)

	// Ranged-specific
	ProjectileCount    int
	ProjectileSpeed    float32
	ProjectileDamage   float32
	ProjectileSpread   float32 // angle between projectiles (radians)
	ProjectileOriginY  float32 // Y offset from enemy position
	ProjectileLifetime float32

	// AoE-specific
	AoERadius float32
	AoEDamage float32

	// Charge-specific
	ChargeSpeed          float32
	ChargeDamage         float32
	ChargeMaxDistance     float32
	ChargeHitRadius      float32
	ChargeStopOnWall     bool
	ChargeStopOnObstacle bool

	// DamageSourceType maps to combat.DamageEvent.SourceType.
	DamageSourceType uint8
}

// PhaseDef defines stat overrides for a specific HP phase.
type PhaseDef struct {
	// Trigger: phase activates when HP ratio <= this value
	HPThresholdPct float32

	// Transition duration (invulnerability)
	TransitionTime float32

	// Movement
	MoveSpeed      float32
	BackpedalSpeed float32 // 0 = use EnemyDef default

	// Cooldown override (0 = use ability's CooldownTime)
	CooldownOverride float32

	// Chase threshold overrides (0 = use EnemyDef defaults)
	ChaseThreshold    float32
	ChaseThresholdFar float32

	// Per-ability weight overrides, keyed by AbilityDef.Name
	WeightOverrides map[string]int

	// Per-ability stat overrides, keyed by AbilityDef.Name
	AbilityOverrides map[string]AbilityOverride
}

// AbilityOverride lets a phase modify specific fields of an ability.
// Nil pointers mean "use the base definition."
type AbilityOverride struct {
	TelegraphTime    *float32
	Damage           *float32 // overrides MeleeDamage, AoEDamage, ChargeDamage, or ProjectileDamage
	ProjectileCount  *int
	AoERadius        *float32
	ChargeSpeed      *float32
	ChargeMaxDistance *float32
	CooldownTime     *float32
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

	// Chase behavior
	ChaseThreshold        float32 // seconds before forcing attack
	ChaseThresholdFar     float32 // threshold when target is far
	FarDistanceMultiplier float32 // multiplied by longest melee range to determine "far"

	// Abilities
	Abilities  []AbilityDef
	AntiRepeat float32 // weight divisor for last-used attack (e.g. 2.0)

	// Phases (sorted by HPThresholdPct descending, e.g. 0.6, 0.3)
	Phases []PhaseDef
}

// CurrentPhase returns the PhaseDef for the given phase number, or nil for phase 1.
func (d *EnemyDef) CurrentPhase(phase int) *PhaseDef {
	// Phase 1 has no overrides (base stats)
	if phase <= 1 {
		return nil
	}
	// Find the matching phase def
	for i := range d.Phases {
		idx := len(d.Phases) - 1 - i // search from most restrictive
		if phase >= len(d.Phases)-idx+1 {
			// Match phase number to index: phase 2 = index 0, phase 3 = index 1, etc.
		}
		_ = idx
	}
	// Simple mapping: phase 2 → index 0, phase 3 → index 1
	idx := phase - 2
	if idx >= 0 && idx < len(d.Phases) {
		return &d.Phases[idx]
	}
	return nil
}

// ResolveAbility returns a copy of the ability with phase overrides applied.
func (d *EnemyDef) ResolveAbility(ability *AbilityDef, phase int) AbilityDef {
	resolved := *ability // copy

	pd := d.CurrentPhase(phase)
	if pd == nil {
		return resolved
	}

	ovr, ok := pd.AbilityOverrides[ability.Name]
	if !ok {
		return resolved
	}

	if ovr.TelegraphTime != nil {
		resolved.TelegraphTime = *ovr.TelegraphTime
	}
	if ovr.CooldownTime != nil {
		resolved.CooldownTime = *ovr.CooldownTime
	}
	if ovr.ProjectileCount != nil {
		resolved.ProjectileCount = *ovr.ProjectileCount
	}
	if ovr.AoERadius != nil {
		resolved.AoERadius = *ovr.AoERadius
	}
	if ovr.ChargeSpeed != nil {
		resolved.ChargeSpeed = *ovr.ChargeSpeed
	}
	if ovr.ChargeMaxDistance != nil {
		resolved.ChargeMaxDistance = *ovr.ChargeMaxDistance
	}
	if ovr.Damage != nil {
		switch resolved.Type {
		case AbilityMelee:
			resolved.MeleeDamage = *ovr.Damage
		case AbilityRanged:
			resolved.ProjectileDamage = *ovr.Damage
		case AbilityAoE:
			resolved.AoEDamage = *ovr.Damage
		case AbilityCharge:
			resolved.ChargeDamage = *ovr.Damage
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
func (d *EnemyDef) CurrentCooldownTime(ability *AbilityDef, phase int) float32 {
	pd := d.CurrentPhase(phase)
	if pd != nil && pd.CooldownOverride > 0 {
		return pd.CooldownOverride
	}
	return ability.CooldownTime
}

// AbilityByIndex returns the ability at the given index, or nil.
func (d *EnemyDef) AbilityByIndex(idx int) *AbilityDef {
	if idx >= 0 && idx < len(d.Abilities) {
		return &d.Abilities[idx]
	}
	return nil
}

// LongestMeleeRange returns the longest melee range among all abilities.
func (d *EnemyDef) LongestMeleeRange() float32 {
	var best float32
	for i := range d.Abilities {
		if d.Abilities[i].Type == AbilityMelee && d.Abilities[i].MeleeRange > best {
			best = d.Abilities[i].MeleeRange
		}
	}
	return best
}

// Helper for creating float32 pointers in phase definitions.
func F32(v float32) *float32 { return &v }

// Helper for creating int pointers in phase definitions.
func Int(v int) *int { return &v }

// Source type constants matching combat.DamageEvent.SourceType.
const (
	SourcePlayer      uint8 = 0
	SourceEnemyMelee  uint8 = 1
	SourceEnemyRanged uint8 = 2
	SourceEnemyAoE    uint8 = 3
	SourceEnemyCharge uint8 = 4
)
