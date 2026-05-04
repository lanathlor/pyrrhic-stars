package combat

import "codex-online/server/internal/entity"

// PatternHandle is an opaque ID for tracking an active pattern instance.
type PatternHandle uint32

// EmitterType identifies a built-in emitter shape.
type EmitterType uint8

const (
	EmitterRadial       EmitterType = iota // 360° ring outward
	EmitterCone                            // fan within angle centered on facing
	EmitterLine                            // perpendicular line, all travel forward
	EmitterArc                             // partial ring segment
	EmitterRingContract                    // ring that closes inward
	EmitterTargeted                        // aimed at facing direction
	EmitterRandomZone                      // random scatter
)

// ProjectileDef describes the projectile spawned by each wave of an emitter.
type ProjectileDef struct {
	Speed           float32 // initial speed (units/s)
	Damage          float32
	Lifetime        float32 // seconds
	Radius          float32 // hit radius override (0 = default)
	Acceleration    float32 // speed change per second (neg = decelerate)
	MaxSpeed        float32 // speed cap (0 = no cap)
	AngularVelocity float32 // radians/s rotation of flight direction (creates curves)
}

// EmitterDef describes one emitter within a pattern.
type EmitterDef struct {
	Type          EmitterType
	Count         int     // projectiles per wave
	Waves         int     // total waves (1 = single burst)
	WaveInterval  float32 // seconds between waves
	OffsetPerWave float32 // radians: cumulative rotation per wave (spiral arms)
	StartAngle    float32 // radians: initial rotation from facing

	// Per-type shape parameters
	ArcAngle    float32 // radians: EmitterCone/EmitterArc total spread
	LineWidth   float32 // units: EmitterLine perpendicular spread
	StartRadius float32 // EmitterRingContract: initial spawn distance
	ZoneRadius  float32 // EmitterRandomZone: scatter area

	// Aimed behavior
	AimAtTarget bool // center on target direction instead of fixed facing

	Projectile ProjectileDef
}

// PatternDef is the complete emitter composition for a bullet-hell ability.
type PatternDef struct {
	Emitters []EmitterDef
}

// SpawnRequest is produced by the pattern engine each tick.
// The system layer converts these into actual Projectile entities.
type SpawnRequest struct {
	Position        entity.Vec3
	Direction       entity.Vec3
	Speed           float32
	Damage          float32
	Lifetime        float32
	Acceleration    float32
	MaxSpeed        float32
	AngularVelocity float32
	Radius          float32
	OwnerID         uint16
	EnemyIdx        int
	VisualTag       string
}
