package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func testAbilityDef(commitTime, executeTime float32) *AbilityDef {
	return &AbilityDef{
		ID:          "test_channel",
		Name:        "Test Channel",
		CommitTime:  commitTime,
		ExecuteTime: executeTime,
		Cooldown:    2.0,
	}
}

func TestPlayerRunner_StartWhileIdle(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.5, 0.2)

	if !r.Start(def) {
		t.Fatal("Start should succeed when runner is idle")
	}
	if r.Phase != PRunnerCommit {
		t.Fatalf("expected PRunnerCommit, got %d", r.Phase)
	}
	if r.AbilityID != "test_channel" {
		t.Fatalf("expected ability ID 'test_channel', got %q", r.AbilityID)
	}
	if r.Timer != 0.5 {
		t.Fatalf("expected Timer=0.5, got %f", r.Timer)
	}
	if r.TotalCommitTime != 0.5 {
		t.Fatalf("expected TotalCommitTime=0.5, got %f", r.TotalCommitTime)
	}
	if r.Charge != 0 {
		t.Fatalf("expected Charge=0 at start, got %f", r.Charge)
	}
}

func TestPlayerRunner_StartWhileBusy(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.5, 0.2)

	r.Start(def)
	if r.Start(def) {
		t.Error("Start should fail when runner is busy")
	}
	if r.Phase != PRunnerCommit {
		t.Errorf("phase should remain PRunnerCommit, got %d", r.Phase)
	}
}

func TestPlayerRunner_CancelDuringCommit(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.5, 0.2)

	r.Start(def)
	if !r.Cancel() {
		t.Fatal("Cancel should succeed during commit phase")
	}
	if r.Phase != PRunnerIdle {
		t.Fatalf("expected PRunnerIdle after cancel, got %d", r.Phase)
	}
	if r.AbilityID != "" {
		t.Fatalf("expected empty AbilityID after cancel, got %q", r.AbilityID)
	}
	if r.Def != nil {
		t.Fatal("expected nil Def after cancel")
	}
}

func TestPlayerRunner_CancelOutsideCommit(t *testing.T) {
	tests := []struct {
		name  string
		phase PlayerRunnerPhase
	}{
		{"idle", PRunnerIdle},
		{"execute", PRunnerExecute},
		{"cooldown", PRunnerCooldown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PlayerAbilityRunner{Phase: tt.phase}
			if r.Cancel() {
				t.Errorf("Cancel should fail in phase %d", tt.phase)
			}
		})
	}
}

func TestPlayerRunner_CommitToExecute(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.5, 0.2)
	r.Start(def)

	dt := float32(0.05)
	fired := false

	// Tick 10 times at 0.05 = 0.5s total, should exhaust commit
	for i := 0; i < 10; i++ {
		if r.Tick(dt) {
			fired = true
		}
	}

	if !fired {
		t.Fatal("Tick should return true when commit expires (fire signal)")
	}
	if r.Phase != PRunnerExecute {
		t.Fatalf("expected PRunnerExecute, got %d", r.Phase)
	}
	if r.Timer != def.ExecuteTime {
		t.Fatalf("expected Timer=%f (execute time), got %f", def.ExecuteTime, r.Timer)
	}
	if r.Charge != 1.0 {
		t.Fatalf("expected Charge=1.0 after commit, got %f", r.Charge)
	}
}

func TestPlayerRunner_ChargeAccumulation(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(1.0, 0.2)
	r.Start(def)

	tests := []struct {
		tickCount     int
		expectedLow   float32
		expectedHigh  float32
	}{
		{5, 0.2, 0.3},   // ~0.25s into 1.0s = ~0.25
		{10, 0.45, 0.55}, // ~0.50s into 1.0s = ~0.50
		{15, 0.7, 0.8},   // ~0.75s into 1.0s = ~0.75
	}

	dt := float32(0.05)
	totalTicks := 0
	for _, tt := range tests {
		for totalTicks < tt.tickCount {
			r.Tick(dt)
			totalTicks++
		}
		if r.Charge < tt.expectedLow || r.Charge > tt.expectedHigh {
			t.Errorf("after %d ticks: Charge=%f, expected [%f, %f]",
				tt.tickCount, r.Charge, tt.expectedLow, tt.expectedHigh)
		}
	}
}

func TestPlayerRunner_FullLifecycle(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.5, 0.2)
	r.Start(def)

	dt := float32(0.05)

	// Phase 1: Commit (0.5s = 10 ticks)
	for i := 0; i < 10; i++ {
		r.Tick(dt)
	}
	if r.Phase != PRunnerExecute {
		t.Fatalf("expected PRunnerExecute after commit, got %d", r.Phase)
	}

	// Phase 2: Execute (0.2s = 4 ticks, +1 for float rounding)
	for i := 0; i < 5; i++ {
		r.Tick(dt)
		if r.Phase != PRunnerExecute {
			break
		}
	}
	if r.Phase != PRunnerCooldown {
		t.Fatalf("expected PRunnerCooldown after execute, got %d", r.Phase)
	}

	// Phase 3: Cooldown (0.3s = 6 ticks, +1 for float rounding)
	for i := 0; i < 7; i++ {
		r.Tick(dt)
		if r.Phase != PRunnerCooldown {
			break
		}
	}
	if r.Phase != PRunnerIdle {
		t.Fatalf("expected PRunnerIdle after cooldown, got %d", r.Phase)
	}
	if r.AbilityID != "" {
		t.Fatalf("expected empty AbilityID after full cycle, got %q", r.AbilityID)
	}
	if r.Def != nil {
		t.Fatal("expected nil Def after full cycle")
	}
}

func TestPlayerRunner_IsBusy(t *testing.T) {
	tests := []struct {
		name  string
		phase PlayerRunnerPhase
		want  bool
	}{
		{"idle", PRunnerIdle, false},
		{"commit", PRunnerCommit, true},
		{"execute", PRunnerExecute, true},
		{"cooldown", PRunnerCooldown, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PlayerAbilityRunner{Phase: tt.phase}
			if got := r.IsBusy(); got != tt.want {
				t.Errorf("IsBusy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlayerRunner_TickIdleNoop(t *testing.T) {
	r := &PlayerAbilityRunner{}
	if r.Tick(0.05) {
		t.Error("Tick on idle runner should return false")
	}
	if r.Phase != PRunnerIdle {
		t.Error("phase should remain idle")
	}
}

func TestPlayerRunner_ZeroCommitTime(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.0, 0.2)
	r.Start(def)

	// With zero commit time, first tick should immediately fire
	fired := r.Tick(0.05)
	if !fired {
		t.Fatal("expected immediate fire with zero commit time")
	}
	if r.Phase != PRunnerExecute {
		t.Fatalf("expected PRunnerExecute, got %d", r.Phase)
	}
	if r.Charge != 1.0 {
		t.Fatalf("expected Charge=1.0, got %f", r.Charge)
	}
}

func TestPlayerRunner_SyncToPlayer(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(1.0, 0.2)
	r.Start(def)

	// Tick partway through commit
	r.Tick(0.3)

	p := &entity.Player{}
	r.SyncToPlayer(p)

	if p.ChannelAbilityID != "test_channel" {
		t.Errorf("ChannelAbilityID = %q, want 'test_channel'", p.ChannelAbilityID)
	}
	if p.ChannelPhase != uint8(PRunnerCommit) {
		t.Errorf("ChannelPhase = %d, want %d", p.ChannelPhase, PRunnerCommit)
	}

	// Timer should be ~0.7
	if math.Abs(float64(p.ChannelTimer-0.7)) > 0.01 {
		t.Errorf("ChannelTimer = %f, want ~0.7", p.ChannelTimer)
	}

	// Charge should be ~0.3
	if math.Abs(float64(p.ChannelCharge-0.3)) > 0.01 {
		t.Errorf("ChannelCharge = %f, want ~0.3", p.ChannelCharge)
	}
}

func TestPlayerRunner_ReStartAfterFullCycle(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.1, 0.1)

	r.Start(def)
	dt := float32(0.05)

	// Burn through entire lifecycle
	for i := 0; i < 100; i++ {
		r.Tick(dt)
		if r.Phase == PRunnerIdle {
			break
		}
	}

	if r.Phase != PRunnerIdle {
		t.Fatalf("expected idle after full cycle, got %d", r.Phase)
	}

	// Should be able to start again
	if !r.Start(def) {
		t.Fatal("Start should succeed after full lifecycle completes")
	}
	if r.Phase != PRunnerCommit {
		t.Fatalf("expected PRunnerCommit on re-start, got %d", r.Phase)
	}
}

func TestCancelCondition_Bitmask(t *testing.T) {
	tests := []struct {
		name       string
		conditions CancelCondition
		checkMove  bool
		checkDmg   bool
		checkInput bool
	}{
		{"none", CancelNone, false, false, false},
		{"move only", CancelOnMove, true, false, false},
		{"damage only", CancelOnDamage, false, true, false},
		{"move+damage", CancelOnMove | CancelOnDamage, true, true, false},
		{"all", CancelOnMove | CancelOnDamage | CancelOnInput, true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conditions&CancelOnMove != 0; got != tt.checkMove {
				t.Errorf("CancelOnMove = %v, want %v", got, tt.checkMove)
			}
			if got := tt.conditions&CancelOnDamage != 0; got != tt.checkDmg {
				t.Errorf("CancelOnDamage = %v, want %v", got, tt.checkDmg)
			}
			if got := tt.conditions&CancelOnInput != 0; got != tt.checkInput {
				t.Errorf("CancelOnInput = %v, want %v", got, tt.checkInput)
			}
		})
	}
}
