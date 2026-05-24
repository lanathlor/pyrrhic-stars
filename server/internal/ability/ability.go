package ability

import "codex-online/server/internal/combat"

// AbilityCategory classifies the execution path of an ability.
type AbilityCategory uint8

const (
	CategoryMelee  AbilityCategory = iota // arc/cone hit resolution
	CategoryRanged                        // spawn projectiles
	CategoryAoE                           // radius hit resolution
	CategoryCharge                        // linear dash with per-entity collision
)

// TargetStrategy controls who the ability targets.
type TargetStrategy uint8

const (
	TargetNearest  TargetStrategy = iota // nearest alive target
	TargetFarthest                       // farthest alive target
	TargetCurrent                        // keep current target
)

// ProjectileDef describes projectile spawning for a ranged ability.
type ProjectileDef struct {
	Count    int     `yaml:"count"`
	Speed    float32 `yaml:"speed"`
	Damage   float32 `yaml:"damage"`
	Spread   float32 `yaml:"spread"`   // angle between projectiles (radians)
	OriginY  float32 `yaml:"origin_y"` // Y offset from committer position
	Lifetime float32 `yaml:"lifetime"`
}

// ChargeDef describes charge movement for a charge ability.
type ChargeDef struct {
	Speed          float32 `yaml:"speed"`
	Damage         float32 `yaml:"damage"`
	MaxDistance    float32 `yaml:"max_distance"`
	HitRadius      float32 `yaml:"hit_radius"`
	StopOnWall     bool    `yaml:"stop_on_wall"`
	StopOnObstacle bool    `yaml:"stop_on_obstacle"`
}

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
	HitAllyTarget      HitType = 10   // single ally targeted by peer ID
	HitAllyLowestHP    HitType = 11   // auto-select lowest HP ally
	HitAllyRandom      HitType = 12   // random ally
	HitGroundPlacement HitType = 13   // place a ground zone at caster position
)

// HitDef describes how an ability finds its targets.
type HitDef struct {
	Type        HitType `yaml:"type"`
	Range       float32 `yaml:"range"`
	ArcDegrees  float32 `yaml:"arc_degrees"`
	Radius      float32 `yaml:"radius"`
	TargetCount int     `yaml:"target_count"`
}

// ResourceCost describes a resource cost for committing an ability.
type ResourceCost struct {
	Resource string  `yaml:"resource"`
	Amount   float32 `yaml:"amount"`
}

// BuffEffect describes a buff applied on commit.
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

// DebuffEffect describes a debuff applied to hit enemy targets on commit.
type DebuffEffect struct {
	ID       string  `yaml:"id"`
	Type     string  `yaml:"type"`     // entity.DebuffSlow, DebuffRoot, DebuffVulnerability
	Value    float32 `yaml:"value"`    // magnitude (e.g. 0.3 for 30% slow)
	Duration float32 `yaml:"duration"` // seconds
}

// AbilityDef describes a single ability.
type AbilityDef struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`

	// School identifies which flux commitment pool this ability draws from.
	// Only relevant for Arcanotechnicien abilities with flux costs.
	School string `yaml:"school"`

	// Hit resolution
	Hit HitDef `yaml:"hit"`

	// Timing
	Cooldown float32 `yaml:"cooldown"` // per-ability cooldown in seconds
	GCD      float32 `yaml:"gcd"`      // global cooldown applied after commit

	// Resource costs
	Costs []ResourceCost `yaml:"costs"`

	// Damage
	BaseDamage float32 `yaml:"base_damage"`

	// Committer effects
	SelfBuffs   []BuffEffect `yaml:"self_buffs"`
	ShieldGrant float32      `yaml:"shield_grant"` // added to "shield" resource

	// Target effects
	TargetDoTs    []DoTEffect    `yaml:"target_dots"`
	TargetDebuffs []DebuffEffect `yaml:"target_debuffs"`

	// Healing
	BaseHeal    float32 `yaml:"base_heal"`
	HealScaling string  `yaml:"heal_scaling"`
	Delivery    uint8   `yaml:"delivery"` // entity.DeliveryMethod for Harmony tracking

	// Healing zone fields
	ZoneRadius   float32 `yaml:"zone_radius"`
	ZoneDuration float32 `yaml:"zone_duration"`
	ZoneHealTick float32 `yaml:"zone_heal_tick"`
	ZoneInterval float32 `yaml:"zone_interval"`

	// Splash damage (secondary AoE around primary hit target)
	SplashRadius         float32 `yaml:"splash_radius"`
	SplashDamageFraction float32 `yaml:"splash_damage_fraction"`

	// Shield scaling: grant shield proportional to damage dealt instead of flat
	ShieldScalesWithDamage bool    `yaml:"shield_scales_with_damage"`
	ShieldPerDamage        float32 `yaml:"shield_per_damage"`

	// Cleanse: number of debuffs to remove from committer (0 = none, stub for future)
	Cleanse int `yaml:"cleanse"`

	// Complex behavior (overrides data-driven resolution)
	Handler string `yaml:"handler"`

	// BD ability data
	OriginConfig int `yaml:"origin_config"` // required blade config (-1 = any)
	DestConfig   int `yaml:"dest_config"`   // config to transition to (-1 = no change)

	// Locks out other abilities for this duration (like blade_swirl preventing attacks)
	LockoutDuration float32 `yaml:"lockout_duration"`

	// Shield cap when granting shield (0 = no cap)
	ShieldCap float32 `yaml:"shield_cap"`

	// Category classifies the execution path (melee/ranged/aoe/charge).
	Category AbilityCategory `yaml:"category"`

	// Commit/execute timing
	CommitTime  float32 `yaml:"commit_time"`  // time entity is committed before execute window
	ExecuteTime float32 `yaml:"execute_time"` // duration of execution window

	// Projectile spawning (ranged abilities)
	Projectile *ProjectileDef `yaml:"projectile"`

	// Charge movement (charge abilities)
	Charge *ChargeDef `yaml:"charge"`

	// Bullet-hell pattern (overrides Projectile fan if set)
	Pattern *combat.PatternDef `yaml:"-"`

	// DamageSource categorizes the damage for client rendering.
	DamageSource uint8 `yaml:"damage_source"`

	// Selection (used by encounter design for weighted ability picking)
	BaseWeight     int            `yaml:"base_weight"`
	MinRange       float32        `yaml:"min_range"`
	MaxRange       float32        `yaml:"max_range"`
	TargetStrategy TargetStrategy `yaml:"target_strategy"`

	// Commit behavior
	FaceTarget  bool `yaml:"face_target"`  // face target at commit
	TrackTarget bool `yaml:"track_target"` // continuously update target during commit

	// Lifecycle control (used by AbilityRunner)
	Cancellable      bool `yaml:"cancellable"`        // BT can abort during commit phase
	CanMoveCommitted bool `yaml:"can_move_committed"` // entity can move during commit
	CanMoveExecuting bool `yaml:"can_move_executing"` // entity can move during execute

	// Player channel control
	CancelConditions uint8  `yaml:"cancel_conditions"` // bitmask: which events cancel during commit
	OnCommitTick     string `yaml:"on_commit_tick"`    // handler name called each tick during commit

	// Sustain (extended channel after execute — Arcanotechnicien class mechanic)
	Sustain           bool    `yaml:"sustain"`              // ability supports extended channel
	SustainCostPerSec float32 `yaml:"sustain_cost_per_sec"` // flux drained per second
	SustainEffect     float32 `yaml:"sustain_effect"`       // base effect per tick (heal or damage)
	SustainInterval   float32 `yaml:"sustain_interval"`     // seconds between sustain ticks
	SustainScaling    float32 `yaml:"sustain_scaling"`      // effect multiplier increase per second held
	SustainHandler    string  `yaml:"sustain_handler"`      // optional per-tick handler name
	SustainCooldown   float32 `yaml:"sustain_cooldown"`     // cooldown applied to ability when sustain is cancelled
}
