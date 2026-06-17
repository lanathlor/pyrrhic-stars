package combat

import (
	"codex-online/server/internal/entity"
	"math"
	"testing"
)

func TestCheckHitscanDirect(t *testing.T) {
	tests := []struct {
		name      string
		origin    entity.Vec3
		dir       entity.Vec3
		target    entity.Vec3
		radius    float32
		maxRange  float32
		obstacles []Obstacle
		want      bool
	}{
		{
			name:     "direct hit straight ahead",
			origin:   entity.Vec3{Z: 10},
			dir:      entity.Vec3{Z: -1},
			target:   entity.Vec3{Z: 0},
			radius:   1.0,
			maxRange: 100,
			want:     true,
		},
		{
			name:     "miss - target behind",
			origin:   entity.Vec3{Z: 10},
			dir:      entity.Vec3{Z: 1}, // aiming away
			target:   entity.Vec3{Z: 0},
			radius:   1.0,
			maxRange: 100,
			want:     false,
		},
		{
			name:     "miss - out of range",
			origin:   entity.Vec3{Z: 200},
			dir:      entity.Vec3{Z: -1},
			target:   entity.Vec3{Z: 0},
			radius:   1.0,
			maxRange: 100,
			want:     false,
		},
		{
			name:     "hit at edge of radius",
			origin:   entity.Vec3{X: 0.9, Z: 10},
			dir:      entity.Vec3{Z: -1},
			target:   entity.Vec3{X: 0, Z: 0},
			radius:   1.0,
			maxRange: 100,
			want:     true,
		},
		{
			name:     "miss just outside radius",
			origin:   entity.Vec3{X: 1.1, Z: 10},
			dir:      entity.Vec3{Z: -1},
			target:   entity.Vec3{X: 0, Z: 0},
			radius:   1.0,
			maxRange: 100,
			want:     false,
		},
		{
			name:     "blocked by obstacle",
			origin:   entity.Vec3{X: 0, Z: 10},
			dir:      entity.Vec3{Z: -1},
			target:   entity.Vec3{X: 0, Z: 0},
			radius:   1.0,
			maxRange: 100,
			obstacles: []Obstacle{
				{CX: 0, CZ: 5, HX: 1.0, HZ: 1.0},
			},
			want: false,
		},
		{
			name:     "obstacle off to the side - no block",
			origin:   entity.Vec3{X: 0, Z: 10},
			dir:      entity.Vec3{Z: -1},
			target:   entity.Vec3{X: 0, Z: 0},
			radius:   1.0,
			maxRange: 100,
			obstacles: []Obstacle{
				{CX: 5, CZ: 5, HX: 1.0, HZ: 1.0},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckHitscan(tt.origin, tt.dir, tt.target, tt.radius, tt.maxRange, tt.obstacles)
			if got != tt.want {
				t.Errorf("CheckHitscan = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCheckHitscanAngles tests hitscan from all cardinal directions and diagonals
// using realistic player eye height (1.6) aiming at enemy center (Y=1.0).
func TestCheckHitscanAngles(t *testing.T) {
	target := entity.Vec3{X: 0, Y: 1.0, Z: 0} // enemy center mass
	radius := float32(2.0)
	maxRange := float32(100.0)

	// Helper: compute aim direction from eye position to a point
	aimAt := func(from, to entity.Vec3) entity.Vec3 {
		return to.Sub(from).Normalized()
	}

	eyeY := float32(1.6)
	enemyFeet := entity.Vec3{X: 0, Y: 0.1, Z: 0}

	tests := []struct {
		name   string
		origin entity.Vec3
		dir    entity.Vec3
		want   bool
	}{
		// === Direct aim at center from all 4 cardinal directions, distance 10 ===
		{
			name:   "front Z+10 aim at center",
			origin: entity.Vec3{X: 0, Y: eyeY, Z: 10},
			dir:    aimAt(entity.Vec3{X: 0, Y: eyeY, Z: 10}, target),
			want:   true,
		},
		{
			name:   "back Z-10 aim at center",
			origin: entity.Vec3{X: 0, Y: eyeY, Z: -10},
			dir:    aimAt(entity.Vec3{X: 0, Y: eyeY, Z: -10}, target),
			want:   true,
		},
		{
			name:   "left X-10 aim at center",
			origin: entity.Vec3{X: -10, Y: eyeY, Z: 0},
			dir:    aimAt(entity.Vec3{X: -10, Y: eyeY, Z: 0}, target),
			want:   true,
		},
		{
			name:   "right X+10 aim at center",
			origin: entity.Vec3{X: 10, Y: eyeY, Z: 0},
			dir:    aimAt(entity.Vec3{X: 10, Y: eyeY, Z: 0}, target),
			want:   true,
		},
		// === Diagonals ===
		{
			name:   "diagonal NE aim at center",
			origin: entity.Vec3{X: 7, Y: eyeY, Z: 7},
			dir:    aimAt(entity.Vec3{X: 7, Y: eyeY, Z: 7}, target),
			want:   true,
		},
		{
			name:   "diagonal SW aim at center",
			origin: entity.Vec3{X: -7, Y: eyeY, Z: -7},
			dir:    aimAt(entity.Vec3{X: -7, Y: eyeY, Z: -7}, target),
			want:   true,
		},
		// === Close range side shot (the reported problem) ===
		{
			name:   "close left X-3 aim at center",
			origin: entity.Vec3{X: -3, Y: eyeY, Z: 0},
			dir:    aimAt(entity.Vec3{X: -3, Y: eyeY, Z: 0}, target),
			want:   true,
		},
		{
			name:   "close right X+3 aim at center",
			origin: entity.Vec3{X: 3, Y: eyeY, Z: 0},
			dir:    aimAt(entity.Vec3{X: 3, Y: eyeY, Z: 0}, target),
			want:   true,
		},
		// === Aim at feet from the side ===
		{
			name:   "side aim at feet",
			origin: entity.Vec3{X: 5, Y: eyeY, Z: 0},
			dir:    aimAt(entity.Vec3{X: 5, Y: eyeY, Z: 0}, enemyFeet),
			want:   true,
		},
		// === Aim straight horizontal from the side (Y=1.0, no pitch) ===
		{
			name:   "side pure horizontal",
			origin: entity.Vec3{X: 10, Y: 1.0, Z: 0},
			dir:    entity.Vec3{X: -1, Y: 0, Z: 0},
			want:   true,
		},
		// === Aim level from side at eye height (Y=1.6) — slight downward needed ===
		{
			name:   "side eye height level aim (no pitch)",
			origin: entity.Vec3{X: 10, Y: eyeY, Z: 0},
			dir:    entity.Vec3{X: -1, Y: 0, Z: 0}, // pure horizontal
			want:   true,                           // Y=1.6 is within cylinder [0, 2.5]
		},
		// === Close range side, aim slightly past center ===
		{
			name:   "close side aim slightly past",
			origin: entity.Vec3{X: -4, Y: eyeY, Z: 0.5},
			dir:    aimAt(entity.Vec3{X: -4, Y: eyeY, Z: 0.5}, entity.Vec3{X: 0, Y: 1.0, Z: 0.5}),
			want:   true,
		},
		// === Miss cases ===
		{
			name:   "side aim above head",
			origin: entity.Vec3{X: 10, Y: eyeY, Z: 0},
			dir:    entity.Vec3{X: -1, Y: 0.5, Z: 0}, // aiming up
			want:   false,
		},
		{
			name:   "side aim wide miss",
			origin: entity.Vec3{X: 10, Y: eyeY, Z: 5},
			dir:    entity.Vec3{X: -1, Y: 0, Z: 0}, // parallel, 5 units offset
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckHitscan(tt.origin, tt.dir, target, radius, maxRange, nil)
			if got != tt.want {
				t.Errorf("CheckHitscan = %v, want %v\n  origin=%v dir=%v target=%v radius=%.1f",
					got, tt.want, tt.origin, tt.dir, target, radius)
			}
		})
	}
}

// TestCheckHitscanRealisticGunner simulates a real gunner shooting using
// the same yaw/pitch → direction math as entity.Player.AimDirection().
func TestCheckHitscanRealisticGunner(t *testing.T) {
	type scenario struct {
		name      string
		playerPos entity.Vec3
		enemyPos  entity.Vec3
	}

	scenarios := []scenario{
		{"side shot from +X", entity.Vec3{X: 5, Y: 0.1, Z: 32}, entity.Vec3{X: 0, Y: 0.1, Z: 32}},
		{"side shot from -X", entity.Vec3{X: -5, Y: 0.1, Z: 32}, entity.Vec3{X: 0, Y: 0.1, Z: 32}},
		{"front shot from +Z", entity.Vec3{X: 0, Y: 0.1, Z: 42}, entity.Vec3{X: 0, Y: 0.1, Z: 32}},
		{"back shot from -Z", entity.Vec3{X: 0, Y: 0.1, Z: 22}, entity.Vec3{X: 0, Y: 0.1, Z: 32}},
		{"diagonal shot", entity.Vec3{X: 5, Y: 0.1, Z: 37}, entity.Vec3{X: 0, Y: 0.1, Z: 32}},
		{"close side shot", entity.Vec3{X: 3, Y: 0.1, Z: 32}, entity.Vec3{X: 0, Y: 0.1, Z: 32}},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			eye := entity.Vec3{X: sc.playerPos.X, Y: 1.6, Z: sc.playerPos.Z}
			targetCenter := sc.enemyPos.Add(entity.Vec3{Y: 1.0})

			// Compute direction exactly like the Godot client + server AimDirection:
			// dir = (-sin(yaw)*cos(pitch), sin(pitch), -cos(yaw)*cos(pitch))
			// Solving for yaw: -sin(yaw) = dir.X/cos(pitch), -cos(yaw) = dir.Z/cos(pitch)
			// So yaw = atan2(-dir.X, -dir.Z) = atan2(sin(yaw), cos(yaw))
			// For a desired direction toward the enemy:
			toEnemy := targetCenter.Sub(eye)
			horizDist := float32(math.Sqrt(float64(toEnemy.X*toEnemy.X + toEnemy.Z*toEnemy.Z)))
			pitch := float32(math.Atan2(float64(toEnemy.Y), float64(horizDist)))
			// yaw such that (-sin(yaw), -cos(yaw)) points toward target on XZ plane
			yaw := float32(math.Atan2(float64(-toEnemy.X), float64(-toEnemy.Z)))

			cp := float32(math.Cos(float64(pitch)))
			sp := float32(math.Sin(float64(pitch)))
			sy := float32(math.Sin(float64(yaw)))
			cy := float32(math.Cos(float64(yaw)))
			dir := entity.Vec3{X: -sy * cp, Y: sp, Z: -cy * cp}

			t.Logf("eye=%v target=%v yaw=%.3f pitch=%.3f dir=%v", eye, targetCenter, yaw, pitch, dir)

			got := CheckHitscan(eye, dir, targetCenter, 2.0, 100.0, nil)
			if !got {
				t.Error("MISSED — direct aim at enemy should hit")
			}
		})
	}
}

func TestCheckMeleeArc(t *testing.T) {
	tests := []struct {
		name      string
		attacker  entity.Vec3
		forward   entity.Vec3
		target    entity.Vec3
		rng       float32
		arc       float32
		obstacles []Obstacle
		want      bool
	}{
		{
			name:     "target in front within range",
			attacker: entity.Vec3{},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{Z: 2},
			rng:      3.0,
			arc:      120,
			want:     true,
		},
		{
			name:     "target behind",
			attacker: entity.Vec3{},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{Z: -2},
			rng:      3.0,
			arc:      120,
			want:     false,
		},
		{
			name:     "target out of range",
			attacker: entity.Vec3{},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{Z: 5},
			rng:      3.0,
			arc:      120,
			want:     false,
		},
		{
			name:     "target at 50 deg (within 120 arc)",
			attacker: entity.Vec3{},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{X: float32(math.Sin(50.0*math.Pi/180.0)) * 2, Z: float32(math.Cos(50.0*math.Pi/180.0)) * 2},
			rng:      3.0,
			arc:      120,
			want:     true,
		},
		{
			name:     "target at 70 deg (outside 120 arc = 60 each side)",
			attacker: entity.Vec3{},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{X: float32(math.Sin(70.0*math.Pi/180.0)) * 2, Z: float32(math.Cos(70.0*math.Pi/180.0)) * 2},
			rng:      3.0,
			arc:      120,
			want:     false,
		},
		{
			name:     "melee blocked by obstacle",
			attacker: entity.Vec3{X: 0, Z: 0},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{Z: 2},
			rng:      3.0,
			arc:      120,
			obstacles: []Obstacle{
				{CX: 0, CZ: 1, HX: 0.5, HZ: 0.5},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckMeleeArc(tt.attacker, tt.forward, tt.target, tt.rng, tt.arc, tt.obstacles)
			if got != tt.want {
				t.Errorf("CheckMeleeArc = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSegmentHitsObstacle(t *testing.T) {
	obstacles := []Obstacle{
		{CX: 0, CZ: 5, HX: 1.0, HZ: 1.0}, // box from (-1,4) to (1,6)
	}

	tests := []struct {
		name string
		a, b entity.Vec3
		want bool
	}{
		{
			name: "segment through obstacle",
			a:    entity.Vec3{X: 0, Z: 0},
			b:    entity.Vec3{X: 0, Z: 10},
			want: true,
		},
		{
			name: "segment misses obstacle",
			a:    entity.Vec3{X: 3, Z: 0},
			b:    entity.Vec3{X: 3, Z: 10},
			want: false,
		},
		{
			name: "segment ends before obstacle",
			a:    entity.Vec3{X: 0, Z: 0},
			b:    entity.Vec3{X: 0, Z: 3},
			want: false,
		},
		{
			name: "segment starts after obstacle",
			a:    entity.Vec3{X: 0, Z: 7},
			b:    entity.Vec3{X: 0, Z: 10},
			want: false,
		},
		{
			name: "diagonal through obstacle",
			a:    entity.Vec3{X: -2, Z: 3},
			b:    entity.Vec3{X: 2, Z: 7},
			want: true,
		},
		{
			name: "zero length segment",
			a:    entity.Vec3{X: 0, Z: 0},
			b:    entity.Vec3{X: 0, Z: 0},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SegmentHitsObstacle(tt.a, tt.b, obstacles)
			if got != tt.want {
				t.Errorf("SegmentHitsObstacle = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectileHitsObstacle(t *testing.T) {
	obstacles := []Obstacle{
		{CX: 0, CZ: 5, HX: 1.0, HZ: 1.0},
	}

	tests := []struct {
		name   string
		pos    entity.Vec3
		radius float32
		want   bool
	}{
		{
			name:   "inside obstacle",
			pos:    entity.Vec3{X: 0, Z: 5},
			radius: 0.3,
			want:   true,
		},
		{
			name:   "near edge - radius reaches in",
			pos:    entity.Vec3{X: 1.2, Z: 5},
			radius: 0.3,
			want:   true,
		},
		{
			name:   "outside obstacle",
			pos:    entity.Vec3{X: 3, Z: 5},
			radius: 0.3,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectileHitsObstacle(tt.pos, tt.radius, obstacles)
			if got != tt.want {
				t.Errorf("ProjectileHitsObstacle = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckProjectileHit(t *testing.T) {
	tests := []struct {
		name      string
		projPos   entity.Vec3
		targetPos entity.Vec3
		hitRadius float32
		want      bool
	}{
		{
			name:      "direct hit - same XZ, center mass Y",
			projPos:   entity.Vec3{X: 5, Y: 1.0, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			hitRadius: 1.0,
			want:      true,
		},
		{
			name:      "hit within radius on XZ plane",
			projPos:   entity.Vec3{X: 5.5, Y: 1.0, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			hitRadius: 1.0,
			want:      true,
		},
		{
			name:      "miss - too far on XZ plane",
			projPos:   entity.Vec3{X: 7, Y: 1.0, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			hitRadius: 1.0,
			want:      false,
		},
		{
			name:      "Y tolerance - projectile at +2 above center mass",
			projPos:   entity.Vec3{X: 5, Y: 3.0, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			// center mass = targetPos.Y + 1.0 = 1.0; dy = 3.0 - 1.0 = 2.0; within tolerance
			hitRadius: 1.0,
			want:      true,
		},
		{
			name:      "Y tolerance exceeded - too high",
			projPos:   entity.Vec3{X: 5, Y: 3.5, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			// center mass = 1.0; dy = 3.5 - 1.0 = 2.5 > 2.0
			hitRadius: 1.0,
			want:      false,
		},
		{
			name:      "Y tolerance exceeded - too low",
			projPos:   entity.Vec3{X: 5, Y: -1.5, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			// center mass = 1.0; dy = -1.5 - 1.0 = -2.5 < -2.0
			hitRadius: 1.0,
			want:      false,
		},
		{
			name:      "edge of radius - exactly at boundary",
			projPos:   entity.Vec3{X: 6, Y: 1.0, Z: 5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			hitRadius: 1.0,
			// flatDistSq = 1.0, hitRadius^2 = 1.0, 1.0 <= 1.0 => hit
			want: true,
		},
		{
			name:      "diagonal XZ just within radius",
			projPos:   entity.Vec3{X: 5.5, Y: 1.0, Z: 5.5},
			targetPos: entity.Vec3{X: 5, Y: 0, Z: 5},
			hitRadius: 1.0,
			// flatDistSq = 0.25+0.25 = 0.5 <= 1.0
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckProjectileHit(tt.projPos, tt.targetPos, tt.hitRadius)
			if got != tt.want {
				t.Errorf("CheckProjectileHit = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSegmentHitsExpandedObstacle(t *testing.T) {
	obstacle := Obstacle{CX: 5, CZ: 5, HX: 1.0, HZ: 1.0}
	// Actual box: (4,4) to (6,6). With radius 0.5 expansion: (3.5,3.5) to (6.5,6.5)

	tests := []struct {
		name      string
		a, b      entity.Vec3
		obstacles []Obstacle
		radius    float32
		want      bool
	}{
		{
			name:      "hits expanded obstacle",
			a:         entity.Vec3{X: 3.6, Z: 0},
			b:         entity.Vec3{X: 3.6, Z: 10},
			obstacles: []Obstacle{obstacle},
			radius:    0.5,
			// 3.6 is within expanded X range [3.5, 6.5] => hit
			want: true,
		},
		{
			name:      "misses original but hits expanded",
			a:         entity.Vec3{X: 3.8, Z: 0},
			b:         entity.Vec3{X: 3.8, Z: 10},
			obstacles: []Obstacle{obstacle},
			radius:    0.5,
			// 3.8 within [3.5, 6.5] => hit
			want: true,
		},
		{
			name:      "misses even expanded obstacle",
			a:         entity.Vec3{X: 3.0, Z: 0},
			b:         entity.Vec3{X: 3.0, Z: 10},
			obstacles: []Obstacle{obstacle},
			radius:    0.5,
			// 3.0 outside [3.5, 6.5] => miss
			want: false,
		},
		{
			name:      "zero-length segment - no hit",
			a:         entity.Vec3{X: 5, Z: 5},
			b:         entity.Vec3{X: 5, Z: 5},
			obstacles: []Obstacle{obstacle},
			radius:    0.5,
			want:      false,
		},
		{
			name:      "does not skip obstacles containing origin",
			a:         entity.Vec3{X: 5, Z: 5},
			b:         entity.Vec3{X: 5, Z: 10},
			obstacles: []Obstacle{obstacle},
			radius:    0.5,
			// Unlike SegmentHitsObstacle, expanded version does NOT skip origin-containing obstacles.
			// But t range is [0,1], and origin is inside so tMin stays 0.
			want: true,
		},
		{
			name:      "segment before obstacle",
			a:         entity.Vec3{X: 5, Z: 0},
			b:         entity.Vec3{X: 5, Z: 3},
			obstacles: []Obstacle{obstacle},
			radius:    0.5,
			// segment ends at Z=3, expanded starts at Z=3.5 => miss
			want: false,
		},
		{
			name:      "no obstacles",
			a:         entity.Vec3{X: 0, Z: 0},
			b:         entity.Vec3{X: 10, Z: 10},
			obstacles: nil,
			radius:    0.5,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SegmentHitsExpandedObstacle(tt.a, tt.b, tt.obstacles, tt.radius)
			if got != tt.want {
				t.Errorf("SegmentHitsExpandedObstacle = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSegmentHitsExpandedObstacle_tMaxZSlabBug(t *testing.T) {
	// A tall, narrow obstacle: wide in X (HX=5), narrow in Z (HZ=0.5).
	// BaseY=0, Height=3 => occupies Y [0, 3].
	// Expanded by radius 0.5: X [-5.5, 5.5], Z [4.5, 6.5]
	//
	// Segment: starts above the obstacle (Y=4) and descends to Y=0
	//   a = (0, 4, 0),  b = (0, 0, 6)
	//
	// X slab: dx=0, origin X=0 is inside [-5.5, 5.5] => tMin=0, tMax=1
	// Z slab: dz=6, Z goes from 0 to 6. Slab [4.5, 6.5].
	//   tMin_z = (4.5 - 0) / 6 = 0.75
	//   tMax_z = (6.5 - 0) / 6 = 1.083 => clamped to 1.0
	//   So refined tMax should be 1.0 (from Z), tMin = 0.75
	//
	// Height check with CORRECT tMax (1.0):
	//   yAtEntry = 4 + (0-4)*0.75 = 4 - 3 = 1.0  => inside [0, 3]
	//   yAtExit  = 4 + (0-4)*1.0  = 0.0           => inside [0, 3]
	//   => HIT (correct, segment enters obstacle from above)
	//
	// Height check with STALE tMax (1.0 from X slab):
	//   Same result here. Let me construct a better case.
	//
	// Better case: obstacle at height Y [5, 8], segment goes Y=4 down to Y=0
	// With correct tMax=1.0: yAtExit = 0 => below [5,8] => MISS
	// With stale tMax=1.0: same here.
	//
	// The real issue: the Z slab can NARROW tMax below the X slab's tMax.
	// When that happens, yAtExit is computed at the wrong t.
	//
	// Obstacle: X [-5.5, 5.5], Z [9.5, 11.5], Y [0, 2]
	// Segment: a = (0, 3, 8), b = (0, 0, 12)
	//   dz=4, Z goes from 8 to 12
	//   X slab: dx=0, origin inside => tMin=0, tMax=1
	//   Z slab: (9.5-8)/4 = 0.375 = tMin_z, (11.5-8)/4 = 0.875 = tMax_z
	//   Correct tMax = 0.875
	//
	// Height check with CORRECT tMax=0.875:
	//   yAtEntry = 3 + (0-3)*0.375 = 3 - 1.125 = 1.875 => inside [0, 2] => hit
	//   yAtExit  = 3 + (0-3)*0.875 = 3 - 2.625 = 0.375 => inside [0, 2] => hit
	//   => HIT (correct)
	//
	// Height check with STALE tMax=1.0 (from X slab):
	//   yAtExit  = 3 + (0-3)*1.0  = 0.0 => inside [0, 2] => hit
	//   Same result again. The stale value is LARGER so yAtExit is LOWER.
	//
	// The false positive case: stale tMax makes yAtExit appear inside when it's
	// actually outside. That happens when the correct tMax is SMALLER (exits
	// the Z slab earlier) and the Y at that point is ABOVE the obstacle.
	//
	// Obstacle: X [-5.5,5.5], Z [4.5,6.5], Y [0, 1]  (short, 1m tall)
	// Segment: a = (0, 2, 0), b = (0, 2, 10)  (horizontal, Y=2 constant)
	//   Y=2 is ABOVE obstacle top Y=1 => should MISS
	//   X slab: dx=0, inside => tMin=0, tMax=1
	//   Z slab: dz=10, (4.5-0)/10=0.45, (6.5-0)/10=0.65 => tMin=0.45, tMax=0.65
	//   Correct: yAtEntry = 2, yAtExit = 2 => both > 1 (obsTop) => MISS
	//   Stale:   yAtEntry = 2, yAtExit = 2 => both > 1 => MISS
	//   (No bug here because Y is constant)
	//
	// For the bug to matter: Y must CHANGE between entry and exit, and the
	// stale tMax must place yAtExit inside the obstacle when the correct one
	// places it outside.
	//
	// Obstacle: X [-5.5,5.5], Z [4.5,6.5], Y [0, 2]
	// Segment: a = (0, 3, 0), b = (0, 0, 10)  (descends from Y=3 to Y=0)
	//   X slab: dx=0, inside => tMin=0, tMax=1
	//   Z slab: dz=10, (4.5)/10=0.45=tMin_z, (6.5)/10=0.65=tMax_z
	//   Correct tMax=0.65:
	//     yAtEntry = 3 + (-3)*0.45 = 1.65 => inside [0,2]
	//     yAtExit  = 3 + (-3)*0.65 = 1.05 => inside [0,2] => HIT
	//   Stale tMax=1.0:
	//     yAtExit  = 3 + (-3)*1.0  = 0.0  => inside [0,2] => HIT (same result)
	//
	// OK, the stale tMax always produces a WIDER window, making yAtExit more
	// extreme. For false positives, we need yAtExit to cross INTO the
	// obstacle with stale tMax but stay OUTSIDE with correct tMax. That means
	// the segment must be going UP (dy > 0), and the obstacle is above.
	//
	// Obstacle: X [-5.5,5.5], Z [4.5,6.5], Y [3, 6]  (floating, 3m above ground)
	// Segment: a = (0, 0, 0), b = (0, 5, 10)  (ascends from Y=0 to Y=5)
	//   dy=5, dz=10
	//   X slab: dx=0, inside => tMin=0, tMax=1
	//   Z slab: (4.5)/10=0.45, (6.5)/10=0.65
	//   Correct tMax=0.65:
	//     yAtEntry = 0 + 5*0.45 = 2.25 => below 3 (BaseY)
	//     yAtExit  = 0 + 5*0.65 = 3.25 => inside [3,6]
	//     Not both below BaseY => HIT (correct - segment clips corner)
	//   Stale tMax=1.0:
	//     yAtEntry = 2.25 => below 3
	//     yAtExit  = 0 + 5*1.0 = 5.0 => inside [3,6] => HIT (same)
	//
	// Hmm. Let me try a case where segment RISES ABOVE the obstacle:
	//
	// Obstacle: X [-5.5,5.5], Z [4.5,6.5], Y [0, 2]
	// Segment: a = (0, 0, 0), b = (0, 10, 10)  (rises from Y=0 to Y=10)
	//   dy=10, dz=10
	//   X slab: dx=0, inside => tMin=0, tMax=1
	//   Z slab: 0.45, 0.65
	//   Correct tMax=0.65:
	//     yAtEntry = 10*0.45 = 4.5 => above obsTop=2
	//     yAtExit  = 10*0.65 = 6.5 => above obsTop=2
	//     Both above => MISS (correct! segment flies over)
	//   Stale tMax=1.0:
	//     yAtEntry = 4.5 => above
	//     yAtExit  = 10.0 => above
	//     Both above => MISS (same)
	//
	// The stale tMax always makes the window wider, pushing yAtExit further.
	// This means the Y range seen is WIDER with stale tMax. For a false
	// positive, we'd need the wider range to include the obstacle when the
	// narrower range doesn't.
	//
	// Case: segment descends through the obstacle, but exits Z slab early
	// enough that it hasn't descended into the obstacle yet.
	//
	// Obstacle: X [-5.5,5.5], Z [4.5,6.5], Y [0, 1]  (1m tall)
	// Segment: a = (0, 3, 0), b = (0, -3, 10)  (descends from Y=3 to Y=-3)
	//   dy=-6, dz=10
	//   X slab: dx=0, inside => tMin=0, tMax=1
	//   Z slab: 0.45, 0.65
	//   Correct tMax=0.65:
	//     yAtEntry = 3 + (-6)*0.45 = 3 - 2.7 = 0.3 => inside [0,1]
	//     yAtExit  = 3 + (-6)*0.65 = 3 - 3.9 = -0.9 => below 0
	//     Not both below BaseY=0 (entry is inside) => HIT
	//   We need both above or both below for a MISS.
	//
	// Let me try: segment descends but enters Z slab at a high Y, exits at
	// a Y still above the obstacle. But stale tMax extends far enough to
	// bring yAtExit into the obstacle.
	//
	// Obstacle: X [-5.5,5.5], Z [1.5,3.5], Y [0, 1]  (1m tall, at ground)
	// Segment: a = (0, 5, 0), b = (0, 0, 10)  (descends Y=5 to Y=0)
	//   dy=-5, dz=10
	//   X slab: inside => tMin=0, tMax=1
	//   Z slab: (1.5)/10=0.15, (3.5)/10=0.35
	//   Correct tMax=0.35:
	//     yAtEntry = 5 + (-5)*0.15 = 4.25 => above 1
	//     yAtExit  = 5 + (-5)*0.35 = 3.25 => above 1
	//     Both above obsTop=1 => MISS (correct!)
	//   Stale tMax=1.0:
	//     yAtEntry = 4.25 => above 1
	//     yAtExit  = 5 + (-5)*1.0 = 0.0 => inside [0,1]
	//     Not both above => FALSE POSITIVE HIT
	//
	// THIS IS THE BUG! Let me write this test.
	obs := Obstacle{CX: 0, CZ: 2.5, HX: 5.0, HZ: 1.0, BaseY: 0, Height: 1}
	// Expanded by 0.5: X [-5.5,5.5], Z [1.0,4.0]
	// Segment descends from Y=5 to Y=0 over Z [0,10]
	a := entity.Vec3{X: 0, Y: 5, Z: 0}
	b := entity.Vec3{X: 0, Y: 0, Z: 10}
	// At Z slab exit (t=0.4 for Z=4.0), Y = 5+(-5)*0.4 = 3.0 => above obstacle (top=1)
	// At Z slab entry (t=0.1 for Z=1.0), Y = 5+(-5)*0.1 = 4.5 => above obstacle
	// Both above => should MISS (segment flies over the 1m obstacle)
	got := SegmentHitsExpandedObstacle(a, b, []Obstacle{obs}, 0.5)
	if got {
		t.Error("SegmentHitsExpandedObstacle false positive: segment passes above 1m obstacle but tMax bug reports hit")
	}
}

func TestNearestObstacleOnSegment_tMaxZSlabBug(t *testing.T) {
	// Same geometry: segment descends but passes above short obstacle
	obs := Obstacle{CX: 0, CZ: 2.5, HX: 5.0, HZ: 1.0, BaseY: 0, Height: 1}
	a := entity.Vec3{X: 0, Y: 5, Z: 0}
	b := entity.Vec3{X: 0, Y: 0, Z: 10}
	_, found := NearestObstacleOnSegment(a, b, []Obstacle{obs}, 0.5)
	if found {
		t.Error("NearestObstacleOnSegment false positive: segment passes above 1m obstacle but tMax bug reports hit")
	}
}

func TestIsPillarLike(t *testing.T) {
	for _, tc := range []struct {
		name string
		obs  Obstacle
		want bool
	}{
		// Tall column with a small footprint: the canonical pillar you orbit.
		{"tall pillar", Obstacle{HX: 0.75, HZ: 0.75, Height: 4.0}, true},
		// Infinitely tall obstacle (Height==0) with a small footprint.
		{"infinite pillar", Obstacle{HX: 1.0, HZ: 1.0, Height: 0}, true},
		// Short cover crate you shoot over: small footprint but not a pillar.
		{"short cover crate", Obstacle{HX: 1.5, HZ: 0.5, Height: 1.2}, false},
		// Boundary wall: large footprint.
		{"boundary wall", Obstacle{HX: 20.0, HZ: 0.25, Height: 5.0}, false},
	} {
		if got := IsPillarLike(tc.obs); got != tc.want {
			t.Errorf("%s: IsPillarLike = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestNearestObstacleOnSegment(t *testing.T) {
	obs1 := Obstacle{CX: 5, CZ: 5, HX: 1.0, HZ: 1.0}
	obs2 := Obstacle{CX: 5, CZ: 15, HX: 1.0, HZ: 1.0}

	tests := []struct {
		name      string
		a, b      entity.Vec3
		obstacles []Obstacle
		radius    float32
		wantFound bool
		wantCX    float32
		wantCZ    float32
	}{
		{
			name:      "finds single obstacle",
			a:         entity.Vec3{X: 5, Z: 0},
			b:         entity.Vec3{X: 5, Z: 20},
			obstacles: []Obstacle{obs1},
			radius:    0.5,
			wantFound: true,
			wantCX:    5,
			wantCZ:    5,
		},
		{
			name:      "finds nearest of two obstacles",
			a:         entity.Vec3{X: 5, Z: 0},
			b:         entity.Vec3{X: 5, Z: 20},
			obstacles: []Obstacle{obs2, obs1}, // obs2 is farther but listed first
			radius:    0.5,
			wantFound: true,
			wantCX:    5,
			wantCZ:    5, // obs1 is closer (Z=5 vs Z=15)
		},
		{
			name:      "no hit - segment misses",
			a:         entity.Vec3{X: 0, Z: 0},
			b:         entity.Vec3{X: 0, Z: 20},
			obstacles: []Obstacle{obs1},
			radius:    0.5,
			wantFound: false,
		},
		{
			name:      "zero-length segment",
			a:         entity.Vec3{X: 5, Z: 5},
			b:         entity.Vec3{X: 5, Z: 5},
			obstacles: []Obstacle{obs1},
			radius:    0.5,
			wantFound: false,
		},
		{
			name:      "no obstacles",
			a:         entity.Vec3{X: 0, Z: 0},
			b:         entity.Vec3{X: 10, Z: 10},
			obstacles: nil,
			radius:    0.5,
			wantFound: false,
		},
		{
			name:      "hits second obstacle only",
			a:         entity.Vec3{X: 5, Z: 10},
			b:         entity.Vec3{X: 5, Z: 20},
			obstacles: []Obstacle{obs1, obs2},
			radius:    0.5,
			// obs1 at Z=5 is before segment start Z=10 (but segment goes 10->20)
			// expanded obs1 Z range: [3.5, 6.5], tMin for Z: (3.5-10)/(20-10)=-0.65 => clamped to 0
			// tMax for Z: (6.5-10)/(20-10)=-0.35 => tMax < tMin(0) => skip
			// obs2 at Z=15 expanded: [13.5, 16.5] => in segment range => hit
			wantFound: true,
			wantCX:    5,
			wantCZ:    15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs, found := NearestObstacleOnSegment(tt.a, tt.b, tt.obstacles, tt.radius)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if found {
				if obs.CX != tt.wantCX || obs.CZ != tt.wantCZ {
					t.Errorf("obstacle center = (%f, %f), want (%f, %f)", obs.CX, obs.CZ, tt.wantCX, tt.wantCZ)
				}
			}
		})
	}
}
