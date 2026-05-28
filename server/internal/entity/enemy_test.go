package entity

import "testing"

func TestAddThreat(t *testing.T) {
	e := NewEnemy(0, 2000.0, "guard_captain")
	e.AddThreat(1, 10.0)
	e.AddThreat(1, 5.0)
	e.AddThreat(2, 20.0)

	if got := e.ThreatTable[1]; got != 15.0 {
		t.Errorf("threat[1] = %f, want 15.0", got)
	}
	if got := e.ThreatTable[2]; got != 20.0 {
		t.Errorf("threat[2] = %f, want 20.0", got)
	}
}

func TestHasThreat(t *testing.T) {
	e := NewEnemy(0, 2000.0, "guard_captain")
	if e.HasThreat(1) {
		t.Error("HasThreat(1) = true on fresh enemy, want false")
	}
	e.AddThreat(1, 10.0)
	if !e.HasThreat(1) {
		t.Error("HasThreat(1) = false after AddThreat, want true")
	}
	if e.HasThreat(2) {
		t.Error("HasThreat(2) = true, want false")
	}
}

func TestClearThreat(t *testing.T) {
	e := NewEnemy(0, 2000.0, "guard_captain")
	e.AddThreat(1, 10.0)
	e.AddThreat(2, 20.0)
	e.ClearThreat()

	if e.HasThreat(1) {
		t.Error("HasThreat(1) = true after ClearThreat")
	}
	if len(e.ThreatTable) != 0 {
		t.Errorf("ThreatTable len = %d after ClearThreat, want 0", len(e.ThreatTable))
	}
}

// --- ChangeState ---

func TestChangeStateChase(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChaseTimer = 5.0
	e.Velocity = Vec3{X: 1, Y: 0, Z: 1}
	e.ChangeState(EnemyChase)

	if e.State != EnemyChase {
		t.Errorf("state = %d, want EnemyChase", e.State)
	}
	if e.ChaseTimer != 0 {
		t.Errorf("ChaseTimer = %f, want 0", e.ChaseTimer)
	}
	if e.Velocity != (Vec3{}) {
		t.Errorf("Velocity = %v, want zero", e.Velocity)
	}
}

func TestChangeStateMeleeTelegraph(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.2},
		{testPhase2, 2, 0.9},
		{testPhase3, 3, 0.7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			e.ChangeState(EnemyMeleeTelegraph)

			if e.State != EnemyMeleeTelegraph {
				t.Errorf("state = %d, want EnemyMeleeTelegraph", e.State)
			}
			if e.StateTimer != tt.want {
				t.Errorf("StateTimer = %f, want %f", e.StateTimer, tt.want)
			}
		})
	}
}

func TestChangeStateMeleeAttack(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChangeState(EnemyMeleeAttack)

	if e.State != EnemyMeleeAttack {
		t.Errorf("state = %d, want EnemyMeleeAttack", e.State)
	}
	if e.StateTimer != 0.3 {
		t.Errorf("StateTimer = %f, want 0.3", e.StateTimer)
	}
}

func TestChangeStateRangedTelegraph(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.0},
		{testPhase2, 2, 0.8},
		{testPhase3, 3, 0.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			e.ChangeState(EnemyRangedTelegraph)

			if e.State != EnemyRangedTelegraph {
				t.Errorf("state = %d, want EnemyRangedTelegraph", e.State)
			}
			if e.StateTimer != tt.want {
				t.Errorf("StateTimer = %f, want %f", e.StateTimer, tt.want)
			}
		})
	}
}

func TestChangeStateRangedAttack(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChangeState(EnemyRangedAttack)

	if e.State != EnemyRangedAttack {
		t.Errorf("state = %d, want EnemyRangedAttack", e.State)
	}
	if e.StateTimer != 0.1 {
		t.Errorf("StateTimer = %f, want 0.1", e.StateTimer)
	}
}

func TestChangeStateAoETelegraph(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.5},
		{testPhase2, 2, 1.2},
		{testPhase3, 3, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			e.ChangeState(EnemyAoETelegraph)

			if e.State != EnemyAoETelegraph {
				t.Errorf("state = %d, want EnemyAoETelegraph", e.State)
			}
			if e.StateTimer != tt.want {
				t.Errorf("StateTimer = %f, want %f", e.StateTimer, tt.want)
			}
		})
	}
}

func TestChangeStateAoESlam(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChangeState(EnemyAoESlam)

	if e.State != EnemyAoESlam {
		t.Errorf("state = %d, want EnemyAoESlam", e.State)
	}
	if e.StateTimer != 0.1 {
		t.Errorf("StateTimer = %f, want 0.1", e.StateTimer)
	}
}

func TestChangeStateChargeTelegraph(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.0},
		{testPhase2, 2, 0.8},
		{testPhase3, 3, 0.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			e.ChargeDirection = Vec3{X: 1, Y: 0, Z: 0}
			e.ChangeState(EnemyChargeTelegraph)

			if e.State != EnemyChargeTelegraph {
				t.Errorf("state = %d, want EnemyChargeTelegraph", e.State)
			}
			if e.StateTimer != tt.want {
				t.Errorf("StateTimer = %f, want %f", e.StateTimer, tt.want)
			}
			if e.ChargeDirection != (Vec3{}) {
				t.Errorf("ChargeDirection = %v, want zero (cleared)", e.ChargeDirection)
			}
		})
	}
}

func TestChangeStateCharge(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChargeDistance = 10.0
	e.ChargeHitPlayers = []uint16{1, 2, 3}
	e.ChargeDirection = Vec3{X: 1, Y: 0, Z: 0}
	e.ChangeState(EnemyCharge)

	if e.State != EnemyCharge {
		t.Errorf("state = %d, want EnemyCharge", e.State)
	}
	if e.ChargeDistance != 0 {
		t.Errorf("ChargeDistance = %f, want 0", e.ChargeDistance)
	}
	if len(e.ChargeHitPlayers) != 0 {
		t.Errorf("ChargeHitPlayers len = %d, want 0", len(e.ChargeHitPlayers))
	}
	// Direction was set, should be preserved
	if e.ChargeDirection.X != 1.0 {
		t.Errorf("ChargeDirection = %v, want (1,0,0) preserved", e.ChargeDirection)
	}
}

func TestChangeStateChargeFallbackDirection(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChargeDirection = Vec3{} // zero vector triggers fallback
	e.ChangeState(EnemyCharge)

	expected := Vec3{X: 0, Y: 0, Z: -1}
	if e.ChargeDirection != expected {
		t.Errorf("ChargeDirection = %v, want %v (fallback forward)", e.ChargeDirection, expected)
	}
}

func TestChangeStateCooldown(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.5},
		{testPhase2, 2, 1.2},
		{testPhase3, 3, 0.9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			e.ChangeState(EnemyCooldown)

			if e.State != EnemyCooldown {
				t.Errorf("state = %d, want EnemyCooldown", e.State)
			}
			if e.StateTimer != tt.want {
				t.Errorf("StateTimer = %f, want %f", e.StateTimer, tt.want)
			}
		})
	}
}

func TestChangeStatePhaseTransition(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.ChangeState(EnemyPhaseTransition)

	if e.State != EnemyPhaseTransition {
		t.Errorf("state = %d, want EnemyPhaseTransition", e.State)
	}
	if e.StateTimer != 1.5 {
		t.Errorf("StateTimer = %f, want 1.5", e.StateTimer)
	}
}

func TestChangeStateDead(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.Velocity = Vec3{X: 5, Y: 0, Z: 5}
	e.ChangeState(EnemyDead)

	if e.State != EnemyDead {
		t.Errorf("state = %d, want EnemyDead", e.State)
	}
	if e.Alive {
		t.Error("Alive = true, want false")
	}
	if e.Velocity != (Vec3{}) {
		t.Errorf("Velocity = %v, want zero", e.Velocity)
	}
}

func TestChangeStatePatrol(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.Velocity = Vec3{X: 3, Y: 0, Z: 3}
	e.ChaseTimer = 5.0
	e.ChangeState(EnemyPatrol)

	if e.State != EnemyPatrol {
		t.Errorf("state = %d, want EnemyPatrol", e.State)
	}
	if e.Velocity != (Vec3{}) {
		t.Errorf("Velocity = %v, want zero", e.Velocity)
	}
	if e.ChaseTimer != 0 {
		t.Errorf("ChaseTimer = %f, want 0", e.ChaseTimer)
	}
}

func TestChangeStateIdleClearsVelocity(t *testing.T) {
	e := NewEnemy(0, 1000, "test")
	e.Velocity = Vec3{X: 2, Y: 0, Z: 2}
	e.ChangeState(EnemyIdle)

	if e.State != EnemyIdle {
		t.Errorf("state = %d, want EnemyIdle", e.State)
	}
	if e.Velocity != (Vec3{}) {
		t.Errorf("Velocity = %v, want zero (all state changes clear velocity)", e.Velocity)
	}
}

func TestResetClearsThreat(t *testing.T) {
	e := NewEnemy(0, 2000.0, "guard_captain")
	e.AddThreat(1, 50.0)
	e.AddThreat(3, 100.0)
	e.Reset(Vec3{})

	if e.HasThreat(1) || e.HasThreat(3) {
		t.Error("threat table not cleared after Reset()")
	}
}

func TestResetClearsTargetPlayerID(t *testing.T) {
	e := NewEnemy(0, 2000.0, "guard_captain")
	e.TargetPlayerID = 42
	e.ActiveAbility = 2
	e.Reset(Vec3{X: 5, Y: 0, Z: 5}, EnemyPatrol)

	if e.TargetPlayerID != 0 {
		t.Errorf("TargetPlayerID = %d after Reset, want 0", e.TargetPlayerID)
	}
	if e.ActiveAbility != 0 {
		t.Errorf("ActiveAbility = %d after Reset, want 0", e.ActiveAbility)
	}
}
