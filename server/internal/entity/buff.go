package entity

// Buff type constants (applied to players).
const (
	BuffDamageMult      = "damage_mult"
	BuffDamageReduction = "damage_reduction"
	BuffCooldownMult    = "cooldown_mult"
	BuffCCImmunity      = "cc_immunity" // immune to crowd control (stub — enemies can't CC yet)
	BuffThorns          = "thorns"      // enemies striking caster take damage (stub — no reverse-damage path yet)
)

// ActiveBuff is a time-limited buff on a player.
type ActiveBuff struct {
	ID       string  // unique identifier, e.g. "overclock", "rechamber_buff"
	Type     string  // buff category: "damage_mult", "damage_reduction", "cooldown_mult"
	Value    float32 // multiplier value (e.g. 1.8 for 80% damage boost, 0.3 for 70% block)
	Duration float32 // remaining seconds (0 = permanent until removed)
}

// Debuff type constants (applied to enemies).
const (
	DebuffSlow          = "slow"          // reduces movement speed by Value fraction
	DebuffRoot          = "root"          // prevents all movement
	DebuffVulnerability = "vulnerability" // increases damage taken by Value fraction
)

// ActiveDebuff is a time-limited debuff on an enemy, sourced from a player ability.
type ActiveDebuff struct {
	ID       string  // unique identifier, e.g. "bd_slow", "bd_root"
	Type     string  // debuff category: DebuffSlow, DebuffRoot, DebuffVulnerability
	Value    float32 // effect magnitude (e.g. 0.3 for 30% slow, 0.2 for 20% vulnerability)
	Duration float32 // remaining seconds
	SourceID uint16  // peer ID of the player who applied it
}

// ActiveDoT is a damage-over-time effect on an enemy, sourced from a player.
type ActiveDoT struct {
	EnemyID    uint16
	SourcePeer uint16
	AbilityID  string
	Damage     float32
	Remaining  float32
	Interval   float32
	TickTimer  float32
}
