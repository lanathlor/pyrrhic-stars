package ability

// HitType identifies how an ability resolves its targeting.
type HitType uint8

const (
	HitNone            HitType = iota // self-buff only, no damage
	HitHitscan                        // ray from eye position
	HitMeleeArc                       // cone in front of caster
	HitAoECircle                      // circle around caster
	HitAoECone                        // cone around caster
	HitAoECircleTarget                // circle around hitscan target
	HitNearestN                       // N nearest in-combat enemies
)

// HitDef describes how an ability finds its targets.
type HitDef struct {
	Type        HitType `yaml:"type"`
	Range       float32 `yaml:"range"`
	ArcDegrees  float32 `yaml:"arc_degrees"`
	Radius      float32 `yaml:"radius"`
	TargetCount int     `yaml:"target_count"`
}

// ResourceCost describes a resource cost for casting an ability.
type ResourceCost struct {
	Resource string  `yaml:"resource"`
	Amount   float32 `yaml:"amount"`
}

// BuffEffect describes a buff applied on cast.
type BuffEffect struct {
	ID       string  `yaml:"id"`
	Type     string  `yaml:"type"`
	Value    float32 `yaml:"value"`
	Duration float32 `yaml:"duration"`
}

// DoTEffect describes a damage-over-time applied to hit targets.
type DoTEffect struct {
	Damage   float32 `yaml:"damage"`
	Duration float32 `yaml:"duration"`
	Interval float32 `yaml:"interval"`
}

// AbilityDef describes a single ability.
type AbilityDef struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`

	// Hit resolution
	Hit HitDef `yaml:"hit"`

	// Timing
	Cooldown float32 `yaml:"cooldown"` // per-ability cooldown in seconds
	GCD      float32 `yaml:"gcd"`      // global cooldown applied after cast

	// Resource costs
	Costs []ResourceCost `yaml:"costs"`

	// Damage
	BaseDamage float32 `yaml:"base_damage"`

	// Caster effects
	SelfBuffs   []BuffEffect `yaml:"self_buffs"`
	ShieldGrant float32      `yaml:"shield_grant"` // added to "shield" resource

	// Target effects
	TargetDoTs []DoTEffect `yaml:"target_dots"`

	// Complex behavior (overrides data-driven resolution)
	Handler string `yaml:"handler"`

	// BD spell data
	OriginConfig int `yaml:"origin_config"` // required blade config (-1 = any)
	DestConfig   int `yaml:"dest_config"`   // config to transition to (-1 = no change)

	// Locks out other abilities for this duration (like blade_swirl preventing attacks)
	LockoutDuration float32 `yaml:"lockout_duration"`

	// Shield cap when granting shield (0 = no cap)
	ShieldCap float32 `yaml:"shield_cap"`
}
