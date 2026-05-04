package combat

import (
	"math"
	"math/rand/v2"
	"testing"

	"codex-online/server/internal/entity"
)

// --- Emitter benchmarks ---

func BenchmarkEmitRadial_8(b *testing.B) {
	emitter := &EmitterDef{
		Type:  EmitterRadial,
		Count: 8,
		Projectile: ProjectileDef{
			Speed: 10, Damage: 5, Lifetime: 3,
		},
	}
	origin := entity.Vec3{}
	facing := entity.Vec3{Z: 1}
	rng := rand.New(rand.NewPCG(1, 2))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		emitWave(emitter, origin, facing, 0, rng)
	}
}

func BenchmarkEmitRadial_32(b *testing.B) {
	emitter := &EmitterDef{
		Type:  EmitterRadial,
		Count: 32,
		Projectile: ProjectileDef{
			Speed: 6, Damage: 10, Lifetime: 5,
		},
	}
	origin := entity.Vec3{}
	facing := entity.Vec3{Z: 1}
	rng := rand.New(rand.NewPCG(1, 2))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		emitWave(emitter, origin, facing, 0, rng)
	}
}

func BenchmarkEmitCone_12(b *testing.B) {
	emitter := &EmitterDef{
		Type:     EmitterCone,
		Count:    12,
		ArcAngle: math.Pi / 3,
		Projectile: ProjectileDef{
			Speed: 14, Damage: 8, Lifetime: 3,
		},
	}
	origin := entity.Vec3{}
	facing := entity.Vec3{Z: 1}
	rng := rand.New(rand.NewPCG(1, 2))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		emitWave(emitter, origin, facing, 0, rng)
	}
}

func BenchmarkEmitRandomZone_16(b *testing.B) {
	emitter := &EmitterDef{
		Type:       EmitterRandomZone,
		Count:      16,
		ZoneRadius: 10,
		Projectile: ProjectileDef{
			Speed: 5, Damage: 3, Lifetime: 4,
		},
	}
	origin := entity.Vec3{}
	facing := entity.Vec3{Z: 1}
	rng := rand.New(rand.NewPCG(1, 2))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		emitWave(emitter, origin, facing, 0, rng)
	}
}

// --- Pattern engine benchmarks ---

// BenchmarkPatternEngine_Spawn measures the cost of creating a new active pattern.
func BenchmarkPatternEngine_Spawn(b *testing.B) {
	pe := NewPatternEngine()
	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:         EmitterRadial,
			Count:        16,
			Waves:        8,
			WaveInterval: 0.15,
			Projectile: ProjectileDef{
				Speed: 8, Damage: 10, Lifetime: 4,
			},
		}},
	}
	origin := entity.Vec3{X: 5, Z: 10}
	facing := entity.Vec3{Z: -1}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pe.Spawn(def, "fire_spiral", 0, 0, origin, facing)
		// Reset to avoid unbounded growth
		pe.Active = pe.Active[:0]
	}
}

// BenchmarkPatternEngine_TickIdle measures tick cost with no active patterns.
func BenchmarkPatternEngine_TickIdle(b *testing.B) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pe.Tick(0.05, rng)
	}
}

// BenchmarkPatternEngine_Tick1Pattern measures ticking a single active pattern
// (no wave fires this tick — just timer accumulation).
func BenchmarkPatternEngine_Tick1Pattern(b *testing.B) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))
	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:         EmitterRadial,
			Count:        16,
			Waves:        100,
			WaveInterval: 0.5, // long interval so most ticks just accumulate
			Projectile: ProjectileDef{
				Speed: 8, Damage: 10, Lifetime: 4,
			},
		}},
	}
	pe.Spawn(def, "test", 0, 0, entity.Vec3{}, entity.Vec3{Z: 1})
	// Consume the first wave
	pe.Tick(0.05, rng)
	pe.Pending = pe.Pending[:0]

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pe.Active[0].WaveTimer = 0.01 // keep below interval
		pe.Tick(0.05, rng)
	}
}

// BenchmarkPatternEngine_Tick10Patterns measures ticking 10 active patterns,
// each firing a wave this tick (worst-case per-tick load: 160 spawns).
func BenchmarkPatternEngine_Tick10Patterns(b *testing.B) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))
	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:          EmitterRadial,
			Count:         16,
			Waves:         10000,
			WaveInterval:  0.05,
			OffsetPerWave: 0.1,
			Projectile: ProjectileDef{
				Speed: 8, Damage: 10, Lifetime: 4,
				AngularVelocity: 0.5,
			},
		}},
	}
	for range 10 {
		pe.Spawn(def, "spiral", 0, 0, entity.Vec3{}, entity.Vec3{Z: 1})
	}
	// Consume first waves
	pe.Tick(0.05, rng)
	pe.Pending = pe.Pending[:0]

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pe.Pending = pe.Pending[:0]
		// Reset wave timers and index so each iteration fires a wave
		for _, ap := range pe.Active {
			ap.WaveTimer = 0.05
			ap.WaveIdx = 1 // prevent completion
		}
		pe.Tick(0.05, rng)
	}
}

// BenchmarkPatternEngine_Tick10Patterns_32Count measures 10 patterns each
// emitting 32 projectiles per wave — a dense bullet-hell scenario (320 spawns/tick).
func BenchmarkPatternEngine_Tick10Patterns_32Count(b *testing.B) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))
	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:          EmitterRadial,
			Count:         32,
			Waves:         10000,
			WaveInterval:  0.05,
			OffsetPerWave: 0.08,
			Projectile: ProjectileDef{
				Speed: 6, Damage: 8, Lifetime: 5,
				AngularVelocity: 0.3,
				Acceleration:    2.0,
				MaxSpeed:        12,
			},
		}},
	}
	for range 10 {
		pe.Spawn(def, "dense", 0, 0, entity.Vec3{}, entity.Vec3{Z: 1})
	}
	pe.Tick(0.05, rng)
	pe.Pending = pe.Pending[:0]

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		pe.Pending = pe.Pending[:0]
		// Reset wave timers and index so each iteration fires a wave
		for _, ap := range pe.Active {
			ap.WaveTimer = 0.05
			ap.WaveIdx = 1 // prevent completion
		}
		pe.Tick(0.05, rng)
	}
}

// --- Projectile Tick benchmarks ---

// BenchmarkProjectileTick_Linear measures the base cost of linear projectile motion.
func BenchmarkProjectileTick_Linear(b *testing.B) {
	p := &entity.Projectile{
		Position:  entity.Vec3{},
		Direction: entity.Vec3{Z: 1},
		Speed:     10,
		Lifetime:  100,
		Alive:     true,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Tick(0.05)
	}
}

// BenchmarkProjectileTick_Curved measures cost with angular velocity (trig per tick).
func BenchmarkProjectileTick_Curved(b *testing.B) {
	p := &entity.Projectile{
		Position:        entity.Vec3{},
		Direction:       entity.Vec3{Z: 1},
		Speed:           10,
		Lifetime:        100,
		Alive:           true,
		AngularVelocity: 0.5,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Tick(0.05)
	}
}

// BenchmarkProjectileTick_Full measures the full overhead: acceleration + angular velocity.
func BenchmarkProjectileTick_Full(b *testing.B) {
	p := &entity.Projectile{
		Position:        entity.Vec3{},
		Direction:       entity.Vec3{Z: 1},
		Speed:           5,
		Lifetime:        100,
		Alive:           true,
		Acceleration:    3.0,
		MaxSpeed:        15,
		AngularVelocity: 0.8,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Tick(0.05)
	}
}

// BenchmarkProjectileBatch_500Linear simulates ticking 500 linear projectiles per frame.
func BenchmarkProjectileBatch_500Linear(b *testing.B) {
	projs := make([]*entity.Projectile, 500)
	for i := range projs {
		projs[i] = &entity.Projectile{
			Position:  entity.Vec3{X: float32(i), Z: float32(i)},
			Direction: entity.Vec3{Z: 1},
			Speed:     10,
			Lifetime:  100,
			Alive:     true,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, p := range projs {
			p.Tick(0.05)
		}
	}
}

// BenchmarkProjectileBatch_500Curved simulates ticking 500 curved projectiles per frame.
func BenchmarkProjectileBatch_500Curved(b *testing.B) {
	projs := make([]*entity.Projectile, 500)
	for i := range projs {
		projs[i] = &entity.Projectile{
			Position:        entity.Vec3{X: float32(i), Z: float32(i)},
			Direction:       entity.Vec3{Z: 1},
			Speed:           8,
			Lifetime:        100,
			Alive:           true,
			AngularVelocity: 0.5,
			Acceleration:    1.0,
			MaxSpeed:        15,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, p := range projs {
			p.Tick(0.05)
		}
	}
}

// BenchmarkProjectileBatch_1000Mixed simulates a real bullet-hell scenario:
// 1000 projectiles, ~60% curved, ~40% linear.
func BenchmarkProjectileBatch_1000Mixed(b *testing.B) {
	projs := make([]*entity.Projectile, 1000)
	for i := range projs {
		p := &entity.Projectile{
			Position:  entity.Vec3{X: float32(i % 50), Z: float32(i / 50)},
			Direction: entity.Vec3{X: float32(i%7) * 0.1, Z: 1},
			Speed:     float32(5 + i%10),
			Lifetime:  100,
			Alive:     true,
		}
		if i%5 < 3 { // 60% curved
			p.AngularVelocity = 0.3 + float32(i%4)*0.1
			p.Acceleration = float32(i%3) * 0.5
			p.MaxSpeed = 15
		}
		projs[i] = p
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, p := range projs {
			p.Tick(0.05)
		}
	}
}
