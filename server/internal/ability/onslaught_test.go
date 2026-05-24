package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestOnslaught_TierThresholds(t *testing.T) {
	tests := []struct {
		stacks int
		tier   uint8
	}{
		{0, TierStandard},
		{1, TierStandard},
		{2, TierStandard},
		{3, TierEmpowered},
		{5, TierEmpowered},
		{6, TierMaximum},
		{10, TierMaximum},
	}
	for _, tt := range tests {
		s := &OnslaughtState{Stacks: tt.stacks}
		if got := s.Tier(); got != tt.tier {
			t.Errorf("stacks=%d: tier=%d, want %d", tt.stacks, got, tt.tier)
		}
	}
}

func TestOnslaught_DamageMult(t *testing.T) {
	tests := []struct {
		stacks  int
		mastery float32
		want    float32
	}{
		{0, 0, 1.0},
		{1, 0, 1.03},
		{3, 0, 1.09},
		{6, 0, 1.18},
		{3, 100, 1.18},  // 1.0 + 3*0.03*(1.0+1.0) = 1.18
		{6, 100, 1.36},  // 1.0 + 6*0.03*(2.0) = 1.36
	}
	for _, tt := range tests {
		s := &OnslaughtState{Stacks: tt.stacks}
		got := s.DamageMult(tt.mastery)
		if math.Abs(float64(got-tt.want)) > 0.001 {
			t.Errorf("stacks=%d mastery=%.0f: mult=%.3f, want %.3f", tt.stacks, tt.mastery, got, tt.want)
		}
	}
}

func TestOnslaught_ResetOnDamage(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	ons := getOnslaughtState(p)
	ons.Stacks = 5

	p.ApplyDamage(10)
	if ons.Stacks != 0 {
		t.Errorf("stacks = %d after damage, want 0 (reset)", ons.Stacks)
	}
}

func TestOnslaught_NoResetWhenInvincible(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	ons := getOnslaughtState(p)
	ons.Stacks = 5

	p.Invincible = true
	p.ApplyDamage(10)
	if ons.Stacks != 5 {
		t.Errorf("stacks = %d after invincible damage, want 5 (no reset)", ons.Stacks)
	}
}

func TestOnslaught_IncrementOnHits(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e1 := enemyInFront(100, 1e6)
	e2 := enemyInFront(101, 1e6)
	e2.Position = entity.Vec3{X: 1, Y: 0, Z: -5}

	r := eng.Commit("cleave", commitCtx(p, e1, e2))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	ons := getOnslaughtState(p)
	if ons.Stacks != len(r.Events) {
		t.Errorf("stacks = %d, want %d (one per enemy hit)", ons.Stacks, len(r.Events))
	}
}

func TestOnslaught_NoResetDuringParry(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	ons := getOnslaughtState(p)
	ons.Stacks = 5

	// Add parry buff + block state so parry triggers
	p.AddBuff(entity.ActiveBuff{ID: "vg_parry", Type: entity.BuffDamageReduction, Value: 1.0, Duration: 0.15})
	state := &VgBlockState{}
	p.AbilityState["vg_block"] = state

	p.ApplyDamage(50)

	// Parry should prevent onslaught reset
	if ons.Stacks != 5 {
		t.Errorf("stacks = %d after parried hit, want 5 (no reset during parry)", ons.Stacks)
	}
	if !state.ParryCounterPending {
		t.Error("expected parry counter to be pending")
	}
}
