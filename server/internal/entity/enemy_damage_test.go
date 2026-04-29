package entity

import "testing"

func TestEnemyApplyDamageBasic(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	dealt, phase := e.ApplyDamage(100)
	if dealt != 100 {
		t.Errorf("dealt = %f, want 100", dealt)
	}
	if phase != 0 {
		t.Errorf("phase trigger = %d, want 0", phase)
	}
	if e.Health != 900 {
		t.Errorf("health = %f, want 900", e.Health)
	}
}

func TestEnemyApplyDamageKills(t *testing.T) {
	e := NewEnemy(0, 100, "test")
	dealt, _ := e.ApplyDamage(200)
	if dealt != 200 {
		t.Errorf("dealt = %f, want 200", dealt)
	}
	if e.Health != 0 {
		t.Errorf("health = %f, want 0", e.Health)
	}
	if e.State != EnemyDead {
		t.Errorf("state = %d, want EnemyDead", e.State)
	}
	if e.Alive {
		t.Error("should not be alive")
	}
}

func TestEnemyApplyDamageToDeadEnemy(t *testing.T) {
	e := NewEnemy(0, 100, "test")
	e.State = EnemyDead
	dealt, _ := e.ApplyDamage(50)
	if dealt != 0 {
		t.Errorf("dealt = %f, want 0 (dead)", dealt)
	}
}

func TestEnemyPhaseTransitionAt60Percent(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	// Bring to exactly 60% → 600 HP. Need to deal 400 damage.
	dealt, phase := e.ApplyDamage(400)
	if dealt != 400 {
		t.Errorf("dealt = %f, want 400", dealt)
	}
	if phase != 2 {
		t.Errorf("phase = %d, want 2 (60%% threshold)", phase)
	}
	if e.Phase != 2 {
		t.Errorf("e.Phase = %d, want 2", e.Phase)
	}
	if e.State != EnemyPhaseTransition {
		t.Errorf("state = %d, want EnemyPhaseTransition", e.State)
	}
}

func TestEnemyPhaseTransitionAt30Percent(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	// Already in phase 2
	e.Phase = 2
	e.PhaseTransitioned = append(e.PhaseTransitioned, 2)
	e.Health = 400 // 40%
	e.State = entity_EnemyChase()

	_, phase := e.ApplyDamage(150) // → 250 HP = 25%
	if phase != 3 {
		t.Errorf("phase = %d, want 3 (30%% threshold)", phase)
	}
	if e.Phase != 3 {
		t.Errorf("e.Phase = %d, want 3", e.Phase)
	}
}

func entity_EnemyChase() EnemyState { return EnemyChase }

func TestEnemyPhaseTransitionSkipTo3(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	// Deal massive damage skipping phase 2 entirely
	_, phase := e.ApplyDamage(750) // → 250 HP = 25%, should trigger phase 3 (checked first)
	if phase != 3 {
		t.Errorf("phase = %d, want 3 (direct skip to 30%%)", phase)
	}
}

func TestEnemyPhaseTransitionNotRepeated(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.Health = 600
	e.Phase = 2
	e.PhaseTransitioned = []int{2}

	_, phase := e.ApplyDamage(50) // 550 HP = 55%, still in phase 2 zone
	if phase != 0 {
		t.Errorf("phase = %d, want 0 (already transitioned)", phase)
	}
}

// --- Reset ---

func TestEnemyResetRestoresState(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.Health = 0
	e.Alive = false
	e.State = EnemyDead
	e.Phase = 3
	e.PhaseTransitioned = []int{2, 3}
	e.LastAttack = "melee"
	e.AddThreat(1, 100)

	spawn := Vec3{X: 5, Y: 0.1, Z: 5}
	e.Reset(spawn)

	if e.Health != 1000 {
		t.Errorf("health = %f, want 1000", e.Health)
	}
	if !e.Alive {
		t.Error("should be alive")
	}
	if e.State != EnemyChase {
		t.Errorf("state = %d, want EnemyChase", e.State)
	}
	if e.Phase != 1 {
		t.Errorf("phase = %d, want 1", e.Phase)
	}
	if len(e.PhaseTransitioned) != 0 {
		t.Error("phase transitioned should be empty")
	}
	if e.Position != spawn {
		t.Errorf("position = %v, want %v", e.Position, spawn)
	}
	if e.LastAttack != "" {
		t.Error("last attack should be empty")
	}
	if len(e.ThreatTable) != 0 {
		t.Error("threat should be cleared")
	}
}

func TestEnemyResetCustomState(t *testing.T) {
	e := NewEnemy(0, 500, "test")
	e.Reset(Vec3{}, EnemyPatrol)
	if e.State != EnemyPatrol {
		t.Errorf("state = %d, want EnemyPatrol", e.State)
	}
}

// --- Projectile ---

func TestProjectileTick(t *testing.T) {
	p := NewProjectile(1, 0, 0, Vec3{}, Vec3{X: 1}, 10, 5, 2.0)
	p.Tick(0.1)
	if p.Position.X != 1.0 { // speed 10 * 0.1 * dir(1,0,0)
		t.Errorf("position.X = %f, want 1.0", p.Position.X)
	}
	if !p.Alive {
		t.Error("should still be alive")
	}
}

func TestProjectileTickExpires(t *testing.T) {
	p := NewProjectile(1, 0, 0, Vec3{}, Vec3{X: 1}, 10, 5, 1.0)
	p.Tick(1.1) // exceeds lifetime
	if p.Alive {
		t.Error("should be dead after lifetime")
	}
}

func TestProjectileTickDead(t *testing.T) {
	p := NewProjectile(1, 0, 0, Vec3{}, Vec3{X: 1}, 10, 5, 2.0)
	p.Alive = false
	startPos := p.Position
	p.Tick(0.1)
	if p.Position != startPos {
		t.Error("dead projectile should not move")
	}
}

func TestProjectileNormalizesDirection(t *testing.T) {
	p := NewProjectile(1, 0, 0, Vec3{}, Vec3{X: 3, Z: 4}, 10, 5, 2.0)
	// Direction should be normalized to (0.6, 0, 0.8)
	if p.Direction.X < 0.59 || p.Direction.X > 0.61 {
		t.Errorf("direction.X = %f, want ~0.6", p.Direction.X)
	}
}
