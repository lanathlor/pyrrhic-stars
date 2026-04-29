package combat

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestPushOutOfObstacles(t *testing.T) {
	tests := []struct {
		name      string
		pos       entity.Vec3
		obstacles []Obstacle
		radius    float32
		wantX     float32
		wantZ     float32
	}{
		{
			name:      "push right (+X)",
			pos:       entity.Vec3{X: 0.8, Z: 0},
			obstacles: []Obstacle{{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0}},
			radius:    0.5,
			wantX:     1.5,
			wantZ:     0,
		},
		{
			name:      "push left (-X)",
			pos:       entity.Vec3{X: -0.8, Z: 0},
			obstacles: []Obstacle{{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0}},
			radius:    0.5,
			wantX:     -1.5,
			wantZ:     0,
		},
		{
			name:      "push forward (+Z)",
			pos:       entity.Vec3{X: 0, Z: 0.8},
			obstacles: []Obstacle{{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0}},
			radius:    0.5,
			wantX:     0,
			wantZ:     1.5,
		},
		{
			name:      "push back (-Z)",
			pos:       entity.Vec3{X: 0, Z: -0.8},
			obstacles: []Obstacle{{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0}},
			radius:    0.5,
			wantX:     0,
			wantZ:     -1.5,
		},
		{
			name: "multiple obstacles push sequentially",
			pos:  entity.Vec3{X: 2.3, Z: 0},
			obstacles: []Obstacle{
				{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0},
				{CX: 3, CZ: 0, HX: 1.0, HZ: 1.0},
			},
			radius: 0.5,
			// X=2.3 is inside obstacle 2 expanded by 0.5 (3-1.5=1.5 to 3+1.5=4.5).
			// pushX = 1.5 - |2.3-3| = 1.5-0.7 = 0.8
			// pushZ = 1.5 - |0-0| = 1.5
			// pushX < pushZ => push X. Since 2.3 < 3, push left => X = 3 - 1.5 = 1.5
			wantX: 1.5,
			wantZ: 0,
		},
		{
			name:      "no overlap - position unchanged",
			pos:       entity.Vec3{X: 5, Z: 5},
			obstacles: []Obstacle{{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0}},
			radius:    0.5,
			wantX:     5,
			wantZ:     5,
		},
		{
			name:      "zero radius push",
			pos:       entity.Vec3{X: 0.5, Z: 0},
			obstacles: []Obstacle{{CX: 0, CZ: 0, HX: 1.0, HZ: 1.0}},
			radius:    0,
			wantX:     1.0,
			wantZ:     0,
		},
		{
			name:      "empty obstacles",
			pos:       entity.Vec3{X: 1, Z: 1},
			obstacles: nil,
			radius:    0.5,
			wantX:     1,
			wantZ:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := tt.pos
			PushOutOfObstacles(&pos, tt.obstacles, tt.radius)
			if !approxEq(pos.X, tt.wantX, 0.001) {
				t.Errorf("X = %f, want %f", pos.X, tt.wantX)
			}
			if !approxEq(pos.Z, tt.wantZ, 0.001) {
				t.Errorf("Z = %f, want %f", pos.Z, tt.wantZ)
			}
		})
	}
}

func TestIsAtWall(t *testing.T) {
	// Arena bounds: minX=-20, maxX=20, minZ=-15, maxZ=15
	const (
		minX float32 = -20
		maxX float32 = 20
		minZ float32 = -15
		maxZ float32 = 15
	)

	tests := []struct {
		name string
		pos  entity.Vec3
		want bool
	}{
		{
			name: "center of arena",
			pos:  entity.Vec3{X: 0, Z: 0},
			want: false,
		},
		{
			name: "at left wall exactly at margin",
			pos:  entity.Vec3{X: -19.5, Z: 0},
			want: true,
		},
		{
			name: "at left wall inside margin",
			pos:  entity.Vec3{X: -19.6, Z: 0},
			want: true,
		},
		{
			name: "just outside left margin",
			pos:  entity.Vec3{X: -19.4, Z: 0},
			want: false,
		},
		{
			name: "at right wall",
			pos:  entity.Vec3{X: 19.5, Z: 0},
			want: true,
		},
		{
			name: "at front wall (+Z)",
			pos:  entity.Vec3{X: 0, Z: 14.5},
			want: true,
		},
		{
			name: "at back wall (-Z)",
			pos:  entity.Vec3{X: 0, Z: -14.5},
			want: true,
		},
		{
			name: "corner - both walls",
			pos:  entity.Vec3{X: 19.6, Z: 14.6},
			want: true,
		},
		{
			name: "just inside all margins",
			pos:  entity.Vec3{X: 19.4, Z: 14.4},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAtWall(tt.pos, minX, maxX, minZ, maxZ)
			if got != tt.want {
				t.Errorf("IsAtWall(%v) = %v, want %v", tt.pos, got, tt.want)
			}
		})
	}
}

func TestIsAtObstacle(t *testing.T) {
	obstacles := []Obstacle{
		{CX: 5, CZ: 5, HX: 1.0, HZ: 1.0},
	}

	tests := []struct {
		name      string
		pos       entity.Vec3
		obstacles []Obstacle
		radius    float32
		want      bool
	}{
		{
			name:      "inside obstacle",
			pos:       entity.Vec3{X: 5, Z: 5},
			obstacles: obstacles,
			radius:    0.5,
			want:      true,
		},
		{
			name:      "on expanded edge",
			pos:       entity.Vec3{X: 6.5, Z: 5},
			obstacles: obstacles,
			radius:    0.5,
			// margin=0.1, exHx = 1.0 + 0.5 + 0.1 = 1.6
			// dx = 6.5-5 = 1.5, 1.5 < 1.6 => inside
			want: true,
		},
		{
			name:      "outside expanded boundary",
			pos:       entity.Vec3{X: 7, Z: 5},
			obstacles: obstacles,
			radius:    0.5,
			// dx = 7-5 = 2.0, exHx = 1.6 => 2.0 > 1.6 => outside
			want: false,
		},
		{
			name:      "zero radius - still uses margin",
			pos:       entity.Vec3{X: 6.0, Z: 5},
			obstacles: obstacles,
			radius:    0,
			// exHx = 1.0 + 0 + 0.1 = 1.1, dx = 1.0 < 1.1 => inside
			want: true,
		},
		{
			name:      "empty obstacles",
			pos:       entity.Vec3{X: 5, Z: 5},
			obstacles: nil,
			radius:    0.5,
			want:      false,
		},
		{
			name:      "far away",
			pos:       entity.Vec3{X: 50, Z: 50},
			obstacles: obstacles,
			radius:    0.5,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAtObstacle(tt.pos, tt.obstacles, tt.radius)
			if got != tt.want {
				t.Errorf("IsAtObstacle(%v, radius=%.1f) = %v, want %v", tt.pos, tt.radius, got, tt.want)
			}
		})
	}
}

func TestRotateVecY(t *testing.T) {
	forward := entity.Vec3{X: 0, Y: 0, Z: 1}

	tests := []struct {
		name  string
		v     entity.Vec3
		angle float32
		wantX float32
		wantY float32
		wantZ float32
	}{
		{
			name:  "0 degrees - no rotation",
			v:     forward,
			angle: 0,
			wantX: 0,
			wantY: 0,
			wantZ: 1,
		},
		{
			name:  "90 degrees CW",
			v:     forward,
			angle: math.Pi / 2,
			wantX: 1,
			wantY: 0,
			wantZ: 0,
		},
		{
			name:  "180 degrees",
			v:     forward,
			angle: math.Pi,
			wantX: 0,
			wantY: 0,
			wantZ: -1,
		},
		{
			name:  "270 degrees CW (= -90)",
			v:     forward,
			angle: 3 * math.Pi / 2,
			wantX: -1,
			wantY: 0,
			wantZ: 0,
		},
		{
			name:  "Y component preserved",
			v:     entity.Vec3{X: 0, Y: 5.0, Z: 1},
			angle: math.Pi / 2,
			wantX: 1,
			wantY: 5.0,
			wantZ: 0,
		},
		{
			name:  "arbitrary vector at 90 degrees",
			v:     entity.Vec3{X: 1, Y: 0, Z: 0},
			angle: math.Pi / 2,
			// X*cos + Z*sin = 1*0 + 0*1 = 0
			// -X*sin + Z*cos = -1*1 + 0*0 = -1
			wantX: 0,
			wantY: 0,
			wantZ: -1,
		},
		{
			name:  "negative angle (-90 degrees)",
			v:     forward,
			angle: -math.Pi / 2,
			wantX: -1,
			wantY: 0,
			wantZ: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RotateVecY(tt.v, tt.angle)
			if !approxEq(got.X, tt.wantX, 0.001) {
				t.Errorf("X = %f, want %f", got.X, tt.wantX)
			}
			if !approxEq(got.Y, tt.wantY, 0.001) {
				t.Errorf("Y = %f, want %f", got.Y, tt.wantY)
			}
			if !approxEq(got.Z, tt.wantZ, 0.001) {
				t.Errorf("Z = %f, want %f", got.Z, tt.wantZ)
			}
		})
	}
}

func approxEq(a, b, eps float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < eps
}
