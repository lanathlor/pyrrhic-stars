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

func TestPlayerRunner_CancelOutsideCommitOrSustain(t *testing.T) {
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
	for range 10 {
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
		tickCount    int
		expectedLow  float32
		expectedHigh float32
	}{
		{5, 0.2, 0.3},    // ~0.25s into 1.0s = ~0.25
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
	for range 10 {
		r.Tick(dt)
	}
	if r.Phase != PRunnerExecute {
		t.Fatalf("expected PRunnerExecute after commit, got %d", r.Phase)
	}

	// Phase 2: Execute (0.2s = 4 ticks, +1 for float rounding)
	for range 5 {
		r.Tick(dt)
		if r.Phase != PRunnerExecute {
			break
		}
	}
	if r.Phase != PRunnerCooldown {
		t.Fatalf("expected PRunnerCooldown after execute, got %d", r.Phase)
	}

	// Phase 3: Cooldown (0.3s = 6 ticks, +1 for float rounding)
	for range 7 {
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
		{"sustain", PRunnerSustain, true},
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
	for range 100 {
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

func testSustainAbilityDef() *AbilityDef {
	return &AbilityDef{
		ID:                tcTestSustain,
		Name:              "Test Sustain",
		CommitTime:        1.0,
		ExecuteTime:       0.1,
		Cooldown:          2.0,
		Sustain:           true,
		SustainCostPerSec: 10,
		SustainEffect:     15,
		SustainInterval:   0.5,
		SustainScaling:    0.05,
		CancelConditions:  uint8(CancelOnMove) | uint8(CancelOnDamage),
	}
}

func TestPlayerRunner_SustainAfterExecute(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.Start(def)

	dt := float32(0.05)

	// Burn through commit (1.0s = 20 ticks)
	for range 20 {
		r.Tick(dt)
	}
	if r.Phase != PRunnerExecute {
		t.Fatalf("expected PRunnerExecute, got %d", r.Phase)
	}

	// Burn through execute (0.1s = 2 ticks) — tick until sustain is reached
	for range 10 {
		r.Tick(dt)
		if r.Phase == PRunnerSustain {
			break
		}
	}
	if r.Phase != PRunnerSustain {
		t.Fatalf("expected PRunnerSustain after execute, got %d", r.Phase)
	}
	// SustainElapsed may be small from the transition tick, but should be < 1 tick
	if r.SustainElapsed > dt+0.001 {
		t.Fatalf("expected SustainElapsed near 0 at start, got %f", r.SustainElapsed)
	}
}

func TestPlayerRunner_SustainTicksFire(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	dt := float32(0.05)
	tickCount := 0

	// Run for 1.0s (20 ticks). With 0.5s interval, expect 2 sustain ticks.
	for range 20 {
		if r.Tick(dt) {
			tickCount++
		}
	}
	if tickCount != 2 {
		t.Errorf("expected 2 sustain ticks in 1.0s (interval=0.5s), got %d", tickCount)
	}
	if r.Phase != PRunnerSustain {
		t.Fatalf("runner should still be in sustain (infinite hold), got %d", r.Phase)
	}
	if r.SustainElapsed < 0.95 || r.SustainElapsed > 1.05 {
		t.Errorf("SustainElapsed = %f, expected ~1.0", r.SustainElapsed)
	}
}

func TestPlayerRunner_SustainScaling(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	dt := float32(0.05)

	// Run for 2.0s
	for range 40 {
		r.Tick(dt)
	}

	// After 2s with 0.05 scaling: Charge = 1.0 + 2.0 * 0.05 = 1.10
	if r.Charge < 1.09 || r.Charge > 1.11 {
		t.Errorf("Charge after 2s sustain = %f, expected ~1.10", r.Charge)
	}
}

func TestPlayerRunner_CancelDuringSustain(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	// Tick a bit
	r.Tick(0.1)
	if r.Phase != PRunnerSustain {
		t.Fatalf("expected PRunnerSustain, got %d", r.Phase)
	}

	// Cancel during sustain should succeed and enter cooldown
	if !r.Cancel() {
		t.Fatal("Cancel should succeed during sustain phase")
	}
	if r.Phase != PRunnerCooldown {
		t.Fatalf("expected PRunnerCooldown after sustain cancel, got %d", r.Phase)
	}
}

func TestPlayerRunner_StartSustainDirect(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	pos := entity.Vec3{X: 1, Y: 2, Z: 3}
	r.StartSustain(def, pos, 42)

	if r.Phase != PRunnerSustain {
		t.Fatalf("expected PRunnerSustain, got %d", r.Phase)
	}
	if r.AbilityID != tcTestSustain {
		t.Fatalf("expected AbilityID 'test_sustain', got %q", r.AbilityID)
	}
	if r.SustainStartPos != pos {
		t.Fatalf("expected SustainStartPos=%v, got %v", pos, r.SustainStartPos)
	}
	if r.SustainStartTick != 42 {
		t.Fatalf("expected SustainStartTick=42, got %d", r.SustainStartTick)
	}
	if r.SustainTickTimer != def.SustainInterval {
		t.Fatalf("expected SustainTickTimer=%f, got %f", def.SustainInterval, r.SustainTickTimer)
	}
}

func TestPlayerRunner_NonSustainSkipsSustainPhase(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testAbilityDef(0.5, 0.2) // No sustain flag

	r.Start(def)
	dt := float32(0.05)

	// Burn through commit + execute
	for range 20 {
		r.Tick(dt)
	}

	// Should go directly to cooldown, not sustain
	if r.Phase == PRunnerSustain {
		t.Fatal("non-sustain ability should not enter sustain phase")
	}
	if r.Phase != PRunnerCooldown {
		t.Fatalf("expected PRunnerCooldown, got %d", r.Phase)
	}
}

func TestPlayerRunner_SustainSyncToPlayer(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	// Tick for 1.0s
	for range 20 {
		r.Tick(0.05)
	}

	p := &entity.Player{}
	r.SyncToPlayer(p)

	if p.ChannelPhase != uint8(PRunnerSustain) {
		t.Errorf("ChannelPhase = %d, want %d", p.ChannelPhase, PRunnerSustain)
	}
	if p.ChannelAbilityID != tcTestSustain {
		t.Errorf("ChannelAbilityID = %q, want 'test_sustain'", p.ChannelAbilityID)
	}
	// Charge should reflect scaling: 1.0 + 1.0 * 0.05 = 1.05
	if p.ChannelCharge < 1.04 || p.ChannelCharge > 1.06 {
		t.Errorf("ChannelCharge = %f, expected ~1.05", p.ChannelCharge)
	}
}

func TestPlayerRunner_SustainFullLifecycle(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.Start(def)

	dt := float32(0.05)

	// Phase 1: Commit (1.0s = 20 ticks)
	for range 20 {
		r.Tick(dt)
	}
	if r.Phase != PRunnerExecute {
		t.Fatalf("expected PRunnerExecute after commit, got %d", r.Phase)
	}

	// Phase 2: Execute (0.1s = ~2 ticks)
	for range 5 {
		r.Tick(dt)
		if r.Phase != PRunnerExecute {
			break
		}
	}
	if r.Phase != PRunnerSustain {
		t.Fatalf("expected PRunnerSustain after execute, got %d", r.Phase)
	}

	// Phase 3: Sustain (hold for 2.0s = 40 ticks)
	for range 40 {
		r.Tick(dt)
	}
	if r.Phase != PRunnerSustain {
		t.Fatalf("sustain should not auto-expire, got %d", r.Phase)
	}

	// Phase 4: Cancel → Cooldown
	if !r.Cancel() {
		t.Fatal("cancel during sustain should succeed")
	}
	if r.Phase != PRunnerCooldown {
		t.Fatalf("expected PRunnerCooldown after sustain cancel, got %d", r.Phase)
	}

	// Phase 5: Cooldown → Idle (0.3s = 6 ticks, +1 margin)
	for range 7 {
		r.Tick(dt)
		if r.Phase == PRunnerIdle {
			break
		}
	}
	if r.Phase != PRunnerIdle {
		t.Fatalf("expected PRunnerIdle after cooldown, got %d", r.Phase)
	}
	if r.AbilityID != "" {
		t.Errorf("AbilityID should be empty after full cycle, got %q", r.AbilityID)
	}
	if r.SustainElapsed != 0 {
		t.Errorf("SustainElapsed should be 0 after reset, got %f", r.SustainElapsed)
	}
}

func TestPlayerRunner_SustainResetFieldsAfterCancel(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{X: 5, Y: 0, Z: 5}, 100)

	// Tick to accumulate state
	for range 10 {
		r.Tick(0.05)
	}
	if r.SustainElapsed < 0.4 {
		t.Fatalf("SustainElapsed should be ~0.5, got %f", r.SustainElapsed)
	}

	// Cancel → cooldown
	r.Cancel()
	// Tick through cooldown to idle
	for range 10 {
		r.Tick(0.05)
	}
	if r.Phase != PRunnerIdle {
		t.Fatalf("expected PRunnerIdle, got %d", r.Phase)
	}

	// All sustain fields should be zeroed after reset
	if r.SustainElapsed != 0 {
		t.Errorf("SustainElapsed = %f after reset, want 0", r.SustainElapsed)
	}
	if r.SustainTickTimer != 0 {
		t.Errorf("SustainTickTimer = %f after reset, want 0", r.SustainTickTimer)
	}
	if r.SustainStartTick != 0 {
		t.Errorf("SustainStartTick = %d after reset, want 0", r.SustainStartTick)
	}
	if r.SustainStartPos != (entity.Vec3{}) {
		t.Errorf("SustainStartPos = %v after reset, want zero", r.SustainStartPos)
	}
}

func TestPlayerRunner_SustainTickIntervalPrecision(t *testing.T) {
	// Test that sustain ticks fire at correct intervals with varying dt
	r := &PlayerAbilityRunner{}
	def := &AbilityDef{
		ID:              "tick_test",
		Sustain:         true,
		SustainInterval: 0.25,
		SustainScaling:  0.0,
	}
	r.StartSustain(def, entity.Vec3{}, 0)

	// Run 1.0s with varying dt sizes
	tickCount := 0
	elapsed := float32(0)
	dts := []float32{0.1, 0.05, 0.1, 0.15, 0.1, 0.1, 0.05, 0.1, 0.05, 0.1, 0.1}
	for _, d := range dts {
		elapsed += d
		if r.Tick(d) {
			tickCount++
		}
	}
	// Over 1.0s with 0.25s interval, expect 4 ticks
	if tickCount != 4 {
		t.Errorf("expected 4 sustain ticks over 1.0s (interval=0.25), got %d (elapsed=%f)", tickCount, elapsed)
	}
}

func TestPlayerRunner_SustainChargeIncreasesMonotonically(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	dt := float32(0.05)
	prevCharge := r.Charge

	for i := range 100 {
		r.Tick(dt)
		if r.Charge < prevCharge {
			t.Fatalf("Charge decreased at tick %d: %f -> %f", i, prevCharge, r.Charge)
		}
		prevCharge = r.Charge
	}
	// After 5s with 0.05 scaling: 1.0 + 5.0 * 0.05 = 1.25
	if r.Charge < 1.24 || r.Charge > 1.26 {
		t.Errorf("Charge after 5s = %f, expected ~1.25", r.Charge)
	}
}

func TestPlayerRunner_SustainDoesNotExpire(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	dt := float32(0.05)
	// Tick for 60 seconds (1200 ticks) — should never leave sustain
	for range 1200 {
		r.Tick(dt)
	}
	if r.Phase != PRunnerSustain {
		t.Fatalf("sustain should never auto-expire, got %d after 60s", r.Phase)
	}
}

func TestPlayerRunner_SustainCancelThenRestart(t *testing.T) {
	r := &PlayerAbilityRunner{}
	def := testSustainAbilityDef()
	r.StartSustain(def, entity.Vec3{}, 0)

	// Tick and cancel
	r.Tick(0.05)
	r.Cancel()

	// Tick through cooldown
	for range 10 {
		r.Tick(0.05)
	}
	if r.Phase != PRunnerIdle {
		t.Fatalf("expected idle after cooldown, got %d", r.Phase)
	}

	// Should be able to start a new ability
	def2 := testAbilityDef(0.3, 0.1)
	if !r.Start(def2) {
		t.Fatal("Start should succeed after sustain cancel + cooldown")
	}
	if r.Phase != PRunnerCommit {
		t.Fatalf("expected PRunnerCommit, got %d", r.Phase)
	}
}

func TestPlayerRunner_SustainPhaseValueIs3(t *testing.T) {
	// Verify the phase enum ordering for wire protocol consistency
	if PRunnerIdle != 0 {
		t.Errorf("PRunnerIdle = %d, want 0", PRunnerIdle)
	}
	if PRunnerCommit != 1 {
		t.Errorf("PRunnerCommit = %d, want 1", PRunnerCommit)
	}
	if PRunnerExecute != 2 {
		t.Errorf("PRunnerExecute = %d, want 2", PRunnerExecute)
	}
	if PRunnerSustain != 3 {
		t.Errorf("PRunnerSustain = %d, want 3", PRunnerSustain)
	}
	if PRunnerCooldown != 4 {
		t.Errorf("PRunnerCooldown = %d, want 4", PRunnerCooldown)
	}
}

func TestPlayerRunner_StartSustainOverwritesBusy(t *testing.T) {
	// StartSustain doesn't check idle — it forcibly enters sustain
	r := &PlayerAbilityRunner{}
	def1 := testAbilityDef(1.0, 0.1)
	r.Start(def1)
	if r.Phase != PRunnerCommit {
		t.Fatalf("expected PRunnerCommit, got %d", r.Phase)
	}

	// Force into sustain while busy (simulates input.go cancel+StartSustain)
	def2 := testSustainAbilityDef()
	r.StartSustain(def2, entity.Vec3{}, 0)
	if r.Phase != PRunnerSustain {
		t.Fatalf("expected PRunnerSustain after forced start, got %d", r.Phase)
	}
	if r.AbilityID != tcTestSustain {
		t.Errorf("AbilityID = %q, want 'test_sustain'", r.AbilityID)
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
