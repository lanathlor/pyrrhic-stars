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
			target:   entity.Vec3{X: float32(math.Sin(50.0 * math.Pi / 180.0)) * 2, Z: float32(math.Cos(50.0*math.Pi/180.0)) * 2},
			rng:      3.0,
			arc:      120,
			want:     true,
		},
		{
			name:     "target at 70 deg (outside 120 arc = 60 each side)",
			attacker: entity.Vec3{},
			forward:  entity.Vec3{Z: 1},
			target:   entity.Vec3{X: float32(math.Sin(70.0 * math.Pi / 180.0)) * 2, Z: float32(math.Cos(70.0*math.Pi/180.0)) * 2},
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
