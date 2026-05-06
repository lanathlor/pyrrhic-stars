package entity

// Buff type constants.
const (
	BuffDamageMult      = "damage_mult"
	BuffDamageReduction = "damage_reduction"
	BuffCooldownMult    = "cooldown_mult"
)

// ActiveBuff is a time-limited buff on a player.
type ActiveBuff struct {
	ID       string  // unique identifier, e.g. "overclock", "rechamber_buff"
	Type     string  // buff category: "damage_mult", "damage_reduction", "cooldown_mult"
	Value    float32 // multiplier value (e.g. 1.8 for 80% damage boost, 0.3 for 70% block)
	Duration float32 // remaining seconds (0 = permanent until removed)
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
