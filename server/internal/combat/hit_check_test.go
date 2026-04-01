package combat

import (
	"codex-online/server/internal/entity"
	"math"
	"testing"
)

func TestCheckHitscanDirect(t *testing.T) {
	tests := []struct {
		name     string
		origin   entity.Vec3
		dir      entity.Vec3
		target   entity.Vec3
		radius   float32
		maxRange float32
		want     bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckHitscan(tt.origin, tt.dir, tt.target, tt.radius, tt.maxRange)
			if got != tt.want {
				t.Errorf("CheckHitscan = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckMeleeArc(t *testing.T) {
	tests := []struct {
		name     string
		attacker entity.Vec3
		forward  entity.Vec3
		target   entity.Vec3
		rng      float32
		arc      float32
		want     bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckMeleeArc(tt.attacker, tt.forward, tt.target, tt.rng, tt.arc)
			if got != tt.want {
				t.Errorf("CheckMeleeArc = %v, want %v", got, tt.want)
			}
		})
	}
}
