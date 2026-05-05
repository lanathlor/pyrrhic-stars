package entity

import (
	"math"
	"testing"
)

// Compile-time interface compliance checks.
var (
	_ Caster      = (*Player)(nil)
	_ Target      = (*Player)(nil)
	_ Caster      = (*Enemy)(nil)
	_ Target      = (*Enemy)(nil)
	_ Threateable = (*Enemy)(nil)
)

// --- Enemy Caster interface ---

func TestEnemyCasterID(t *testing.T) {
	e := NewEnemy(42, 100, "test")
	if got := e.CasterID(); got != 42 {
		t.Errorf("CasterID() = %d, want 42", got)
	}
}

func TestEnemyCasterPos(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	e.Position = Vec3{X: 3, Y: 1, Z: -7}
	if got := e.CasterPos(); got != e.Position {
		t.Errorf("CasterPos() = %v, want %v", got, e.Position)
	}
}

func TestEnemyCasterForward(t *testing.T) {
	tests := []struct {
		name  string
		rotY  float32
		wantX float32
		wantZ float32
	}{
		{"facing -Z (default)", 0, 0, -1},
		{"facing -X", float32(math.Pi / 2), -1, 0},
		{"facing +Z", float32(math.Pi), 0, 1},
		{"facing +X", float32(-math.Pi / 2), 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(1, 100, "test")
			e.RotationY = tt.rotY
			fwd := e.CasterForward()

			if math.Abs(float64(fwd.X-tt.wantX)) > 0.001 {
				t.Errorf("forward.X = %f, want %f", fwd.X, tt.wantX)
			}
			if fwd.Y != 0 {
				t.Errorf("forward.Y = %f, want 0", fwd.Y)
			}
			if math.Abs(float64(fwd.Z-tt.wantZ)) > 0.001 {
				t.Errorf("forward.Z = %f, want %f", fwd.Z, tt.wantZ)
			}
		})
	}
}

func TestEnemyCasterForwardIsUnitVector(t *testing.T) {
	for _, rotY := range []float32{0, 0.5, 1.0, 2.0, float32(math.Pi), -1.5} {
		fwd := (&Enemy{Combatant: Combatant{RotationY: rotY}}).CasterForward()
		length := fwd.Length()
		if math.Abs(float64(length-1.0)) > 0.001 {
			t.Errorf("rotY=%f: forward length = %f, want 1.0", rotY, length)
		}
	}
}

func TestEnemyCasterEyePos(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	e.Position = Vec3{X: 5, Y: 0, Z: 10}
	eye := e.CasterEyePos()
	want := Vec3{X: 5, Y: 1.5, Z: 10}
	if eye != want {
		t.Errorf("CasterEyePos() = %v, want %v", eye, want)
	}
}

func TestEnemyCasterAimDirMatchesForward(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	e.RotationY = 1.23
	if e.CasterAimDir() != e.CasterForward() {
		t.Error("CasterAimDir() should equal CasterForward() for enemies")
	}
}

func TestEnemyCasterAlive(t *testing.T) {
	tests := []struct {
		name  string
		alive bool
		state EnemyState
		want  bool
	}{
		{"alive and chasing", true, EnemyChase, true},
		{"alive but dead state", true, EnemyDead, false},
		{"not alive but chase state", false, EnemyChase, false},
		{"not alive and dead state", false, EnemyDead, false},
		{"alive and in cooldown", true, EnemyCooldown, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(1, 100, "test")
			e.Alive = tt.alive
			e.State = tt.state
			if got := e.CasterAlive(); got != tt.want {
				t.Errorf("CasterAlive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnemyCasterDamageMult(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	if got := e.CasterDamageMult(); got != 1.0 {
		t.Errorf("CasterDamageMult() = %f, want 1.0", got)
	}
}

// --- Enemy Target interface ---

func TestEnemyTargetID(t *testing.T) {
	e := NewEnemy(99, 100, "test")
	if got := e.TargetID(); got != 99 {
		t.Errorf("TargetID() = %d, want 99", got)
	}
}

func TestEnemyTargetPos(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	e.Position = Vec3{X: 7, Y: 2, Z: -3}
	if got := e.TargetPos(); got != e.Position {
		t.Errorf("TargetPos() = %v, want %v", got, e.Position)
	}
}

func TestEnemyTargetAlive(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	if !e.TargetAlive() {
		t.Error("TargetAlive() should be true for new enemy")
	}
	e.State = EnemyDead
	if e.TargetAlive() {
		t.Error("TargetAlive() should be false for dead enemy")
	}
}

func TestEnemyTargetApplyDamage(t *testing.T) {
	e := NewEnemy(1, 200, "test")
	dealt := e.TargetApplyDamage(50)
	if dealt != 50 {
		t.Errorf("dealt = %f, want 50", dealt)
	}
	if e.Health != 150 {
		t.Errorf("health = %f, want 150", e.Health)
	}
}

func TestEnemyTargetApplyDamage_DeadReturnsZero(t *testing.T) {
	e := NewEnemy(1, 100, "test")
	e.State = EnemyDead
	e.Alive = false
	dealt := e.TargetApplyDamage(50)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 for dead enemy", dealt)
	}
}

// --- Player Caster/Target interface ---

func TestPlayerCasterID(t *testing.T) {
	p := NewPlayer(7, ClassGunner)
	if got := p.CasterID(); got != 7 {
		t.Errorf("CasterID() = %d, want 7", got)
	}
}

func TestPlayerCasterPos(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	p.Position = Vec3{X: 1, Y: 2, Z: 3}
	if got := p.CasterPos(); got != p.Position {
		t.Errorf("CasterPos() = %v, want %v", got, p.Position)
	}
}

func TestPlayerCasterAlive(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	if !p.CasterAlive() {
		t.Error("CasterAlive() should be true")
	}
	p.Alive = false
	if p.CasterAlive() {
		t.Error("CasterAlive() should be false when dead")
	}
}

func TestPlayerTargetApplyDamage(t *testing.T) {
	p := NewPlayer(1, ClassGunner)
	hp := p.Health
	dealt := p.TargetApplyDamage(25)
	if dealt != 25 {
		t.Errorf("dealt = %f, want 25", dealt)
	}
	if p.Health != hp-25 {
		t.Errorf("health = %f, want %f", p.Health, hp-25)
	}
}

// --- Benchmarks ---

func BenchmarkEnemyCasterForward(b *testing.B) {
	e := NewEnemy(1, 100, "test")
	e.RotationY = 1.23
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = e.CasterForward()
	}
}

func BenchmarkEnemyTargetApplyDamage(b *testing.B) {
	e := NewEnemy(1, 1e12, "test")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.TargetApplyDamage(10)
	}
}
