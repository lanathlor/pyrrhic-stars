package combat

import (
	"math"
	"math/rand/v2"
	"testing"

	"codex-online/server/internal/entity"
)

func TestRadialEmitter_CountAndSpacing(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:  EmitterRadial,
			Count: 8,
			Waves: 1,
			Projectile: ProjectileDef{
				Speed:    10,
				Damage:   5,
				Lifetime: 3,
			},
		}},
	}

	origin := entity.Vec3{}
	facing := entity.Vec3{Z: 1}
	pe.Spawn(def, "test", 0, 0, origin, facing)
	pe.Tick(0.05, rng)
	spawns := pe.DrainSpawns()

	if len(spawns) != 8 {
		t.Fatalf("expected 8 spawns, got %d", len(spawns))
	}

	// Check even spacing: each direction should be 45° apart
	expectedStep := 2 * math.Pi / 8
	for i, s := range spawns {
		angle := math.Atan2(float64(s.Direction.X), float64(s.Direction.Z))
		expectedAngle := expectedStep * float64(i)
		// Normalize to [0, 2π)
		angle = math.Mod(angle+2*math.Pi, 2*math.Pi)
		expectedAngle = math.Mod(expectedAngle+2*math.Pi, 2*math.Pi)
		if math.Abs(angle-expectedAngle) > 0.01 {
			t.Errorf("spawn %d: angle %.3f, expected %.3f", i, angle, expectedAngle)
		}
	}
}

func TestRadialEmitter_WaveOffset(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	offsetDeg := float32(15)
	offsetRad := offsetDeg * math.Pi / 180

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:          EmitterRadial,
			Count:         4,
			Waves:         2,
			WaveInterval:  0.1,
			OffsetPerWave: offsetRad,
			Projectile: ProjectileDef{
				Speed:    10,
				Damage:   5,
				Lifetime: 3,
			},
		}},
	}

	origin := entity.Vec3{}
	facing := entity.Vec3{Z: 1}
	pe.Spawn(def, "test", 0, 0, origin, facing)

	// Tick 1: first wave fires immediately
	pe.Tick(0.05, rng)
	wave0 := append([]SpawnRequest{}, pe.DrainSpawns()...)

	// Tick 2+3: accumulate 0.1s for second wave
	pe.Tick(0.05, rng)
	pe.Tick(0.05, rng)
	wave1 := pe.DrainSpawns()

	if len(wave0) != 4 || len(wave1) != 4 {
		t.Fatalf("expected 4+4 spawns, got %d+%d", len(wave0), len(wave1))
	}

	// Wave 1's first projectile should be rotated by offsetRad from wave 0's first
	angle0 := math.Atan2(float64(wave0[0].Direction.X), float64(wave0[0].Direction.Z))
	angle1 := math.Atan2(float64(wave1[0].Direction.X), float64(wave1[0].Direction.Z))
	diff := angle1 - angle0
	if math.Abs(diff-float64(offsetRad)) > 0.01 {
		t.Errorf("wave offset: got %.4f, expected %.4f", diff, offsetRad)
	}
}

func TestConeEmitter_SpreadWithinAngle(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	arcAngle := float32(60) * math.Pi / 180 // 60 degrees

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:     EmitterCone,
			Count:    5,
			Waves:    1,
			ArcAngle: arcAngle,
			Projectile: ProjectileDef{
				Speed:    10,
				Damage:   5,
				Lifetime: 3,
			},
		}},
	}

	facing := entity.Vec3{Z: 1} // facing +Z
	origin := entity.Vec3{}
	pe.Spawn(def, "test", 0, 0, origin, facing)
	pe.Tick(0.05, rng)
	spawns := pe.DrainSpawns()

	if len(spawns) != 5 {
		t.Fatalf("expected 5 spawns, got %d", len(spawns))
	}

	facingAngle := math.Atan2(float64(facing.X), float64(facing.Z))
	halfArc := float64(arcAngle) / 2

	for i, s := range spawns {
		angle := math.Atan2(float64(s.Direction.X), float64(s.Direction.Z))
		diff := angle - facingAngle
		if math.Abs(diff) > halfArc+0.01 {
			t.Errorf("spawn %d: angle diff %.3f exceeds half-arc %.3f", i, diff, halfArc)
		}
	}
}

func TestLineEmitter_ParallelDirections(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:      EmitterLine,
			Count:     4,
			Waves:     1,
			LineWidth: 10,
			Projectile: ProjectileDef{
				Speed:    8,
				Damage:   5,
				Lifetime: 3,
			},
		}},
	}

	facing := entity.Vec3{Z: 1}
	origin := entity.Vec3{}
	pe.Spawn(def, "test", 0, 0, origin, facing)
	pe.Tick(0.05, rng)
	spawns := pe.DrainSpawns()

	if len(spawns) != 4 {
		t.Fatalf("expected 4 spawns, got %d", len(spawns))
	}

	// All directions should be parallel (same angle)
	refAngle := math.Atan2(float64(spawns[0].Direction.X), float64(spawns[0].Direction.Z))
	for i, s := range spawns[1:] {
		angle := math.Atan2(float64(s.Direction.X), float64(s.Direction.Z))
		if math.Abs(angle-refAngle) > 0.01 {
			t.Errorf("spawn %d: direction not parallel (%.3f vs %.3f)", i+1, angle, refAngle)
		}
	}

	// Positions should be spread along perpendicular axis
	for i := 1; i < len(spawns); i++ {
		if spawns[i].Position == spawns[i-1].Position {
			t.Errorf("spawn %d: same position as previous", i)
		}
	}
}

func TestRingContract_DirectionPointsInward(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:        EmitterRingContract,
			Count:       8,
			Waves:       1,
			StartRadius: 10,
			Projectile: ProjectileDef{
				Speed:    5,
				Damage:   10,
				Lifetime: 2,
			},
		}},
	}

	origin := entity.Vec3{X: 5, Z: 5}
	facing := entity.Vec3{Z: 1}
	pe.Spawn(def, "test", 0, 0, origin, facing)
	pe.Tick(0.05, rng)
	spawns := pe.DrainSpawns()

	if len(spawns) != 8 {
		t.Fatalf("expected 8 spawns, got %d", len(spawns))
	}

	for i, s := range spawns {
		// Projectile should be at radius 10 from origin
		dist := s.Position.Sub(origin).Length()
		if math.Abs(float64(dist-10)) > 0.1 {
			t.Errorf("spawn %d: distance from origin %.2f, expected 10", i, dist)
		}
		// Direction should point toward origin
		toOrigin := origin.Sub(s.Position).Normalized()
		dot := s.Direction.X*toOrigin.X + s.Direction.Z*toOrigin.Z
		if dot < 0.95 {
			t.Errorf("spawn %d: direction dot with inward=%.3f, expected ~1.0", i, dot)
		}
	}
}

func TestPatternLifecycle_WaveTiming(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:         EmitterRadial,
			Count:        4,
			Waves:        3,
			WaveInterval: 0.1,
			Projectile: ProjectileDef{
				Speed:    10,
				Damage:   5,
				Lifetime: 3,
			},
		}},
	}

	h := pe.Spawn(def, "test", 0, 0, entity.Vec3{}, entity.Vec3{Z: 1})

	// Tick 1 (0.05s): first wave fires immediately
	pe.Tick(0.05, rng)
	if len(pe.DrainSpawns()) != 4 {
		t.Fatal("wave 0 should fire immediately")
	}

	// Tick 2 (0.05s): timer at 0.05, not yet 0.1
	pe.Tick(0.05, rng)
	if len(pe.DrainSpawns()) != 0 {
		t.Fatal("wave 1 should not fire yet (0.05 < 0.1)")
	}

	// Tick 3 (0.05s): timer at 0.10, fires wave 1
	pe.Tick(0.05, rng)
	if len(pe.DrainSpawns()) != 4 {
		t.Fatal("wave 1 should fire at 0.1s")
	}

	// Tick 4+5: fires wave 2
	pe.Tick(0.05, rng)
	pe.Tick(0.05, rng)
	if len(pe.DrainSpawns()) != 4 {
		t.Fatal("wave 2 should fire at 0.2s")
	}

	// Pattern should be done
	if pe.IsActive(h) {
		t.Fatal("pattern should be done after all waves")
	}
}

func TestPatternLifecycle_MultiEmitterSequence(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{
			{
				Type:  EmitterRadial,
				Count: 4,
				Waves: 1,
				Projectile: ProjectileDef{
					Speed: 10, Damage: 5, Lifetime: 3,
				},
			},
			{
				Type:     EmitterCone,
				Count:    3,
				Waves:    1,
				ArcAngle: math.Pi / 3,
				Projectile: ProjectileDef{
					Speed: 15, Damage: 10, Lifetime: 2,
				},
			},
		},
	}

	h := pe.Spawn(def, "test", 0, 0, entity.Vec3{}, entity.Vec3{Z: 1})

	// Tick 1: emitter 0 fires (radial, 4 projectiles)
	pe.Tick(0.05, rng)
	spawns := pe.DrainSpawns()
	if len(spawns) != 4 {
		t.Fatalf("emitter 0: expected 4, got %d", len(spawns))
	}

	// Tick 2: emitter 1 fires immediately (cone, 3 projectiles)
	pe.Tick(0.05, rng)
	spawns = pe.DrainSpawns()
	if len(spawns) != 3 {
		t.Fatalf("emitter 1: expected 3, got %d", len(spawns))
	}

	// Pattern done
	if pe.IsActive(h) {
		t.Fatal("pattern should be done after both emitters complete")
	}
}

func TestPatternCancel(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:         EmitterRadial,
			Count:        4,
			Waves:        10,
			WaveInterval: 0.1,
			Projectile: ProjectileDef{
				Speed: 10, Damage: 5, Lifetime: 3,
			},
		}},
	}

	h := pe.Spawn(def, "test", 0, 0, entity.Vec3{}, entity.Vec3{Z: 1})

	// Fire first wave
	pe.Tick(0.05, rng)
	if len(pe.DrainSpawns()) != 4 {
		t.Fatal("first wave should fire")
	}

	// Cancel
	pe.Cancel(h)
	if pe.IsActive(h) {
		t.Fatal("pattern should be cancelled")
	}

	// No more spawns
	pe.Tick(0.05, rng)
	pe.Tick(0.05, rng)
	pe.Tick(0.05, rng)
	if len(pe.DrainSpawns()) != 0 {
		t.Fatal("cancelled pattern should not spawn more")
	}
}

func TestProjectileCurvedMotion(t *testing.T) {
	// Verify projectile with angular velocity traces a curve
	p := &entity.Projectile{
		Position:        entity.Vec3{},
		Direction:       entity.Vec3{Z: 1},
		Speed:           10,
		Lifetime:        5,
		Alive:           true,
		AngularVelocity: math.Pi / 2, // 90 deg/s
	}

	// After 1 second (20 ticks at 0.05s), direction should be ~90° rotated
	for range 20 {
		p.Tick(0.05)
	}

	// Direction should now point roughly in +X
	if math.Abs(float64(p.Direction.X)-1.0) > 0.05 {
		t.Errorf("after 1s at π/2 rad/s: dir.X=%.3f, expected ~1.0", p.Direction.X)
	}
	if math.Abs(float64(p.Direction.Z)) > 0.05 {
		t.Errorf("after 1s at π/2 rad/s: dir.Z=%.3f, expected ~0.0", p.Direction.Z)
	}

	// Position should have traced a curve (not a straight line)
	// For circular motion at speed=10, radius=10/(π/2)≈6.37
	// After quarter circle: should be near (radius, 0, radius)
	if p.Position.X < 1 {
		t.Errorf("curved projectile should have moved in +X, got X=%.2f", p.Position.X)
	}
}

func TestProjectileAcceleration(t *testing.T) {
	p := &entity.Projectile{
		Position:     entity.Vec3{},
		Direction:    entity.Vec3{Z: 1},
		Speed:        2,
		Lifetime:     5,
		Alive:        true,
		Acceleration: 10,
		MaxSpeed:     8,
	}

	// After 1 second: speed should be capped at 8
	for range 20 {
		p.Tick(0.05)
	}

	if math.Abs(float64(p.Speed)-8.0) > 0.1 {
		t.Errorf("speed after 1s with accel=10, max=8: got %.2f", p.Speed)
	}
}

func TestProjectileDeceleration(t *testing.T) {
	p := &entity.Projectile{
		Position:     entity.Vec3{},
		Direction:    entity.Vec3{Z: 1},
		Speed:        10,
		Lifetime:     5,
		Alive:        true,
		Acceleration: -20, // strong deceleration
	}

	// After 1 second: speed should be clamped at 0
	for range 20 {
		p.Tick(0.05)
	}

	if p.Speed != 0 {
		t.Errorf("speed should be 0 after deceleration, got %.2f", p.Speed)
	}
}

func TestSpawnRequest_OwnerPropagation(t *testing.T) {
	pe := NewPatternEngine()
	rng := rand.New(rand.NewPCG(1, 2))

	def := &PatternDef{
		Emitters: []EmitterDef{{
			Type:  EmitterRadial,
			Count: 2,
			Waves: 1,
			Projectile: ProjectileDef{
				Speed: 10, Damage: 5, Lifetime: 3,
			},
		}},
	}

	pe.Spawn(def, "fire_spiral", 0, 3, entity.Vec3{}, entity.Vec3{Z: 1})
	pe.Tick(0.05, rng)
	spawns := pe.DrainSpawns()

	for i, s := range spawns {
		if s.OwnerID != 0 {
			t.Errorf("spawn %d: OwnerID=%d, expected 0", i, s.OwnerID)
		}
		if s.EnemyIdx != 3 {
			t.Errorf("spawn %d: EnemyIdx=%d, expected 3", i, s.EnemyIdx)
		}
		if s.VisualTag != "fire_spiral" {
			t.Errorf("spawn %d: VisualTag=%q, expected fire_spiral", i, s.VisualTag)
		}
	}
}
