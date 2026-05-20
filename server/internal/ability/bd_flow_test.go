package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestFlow_TierThresholds(t *testing.T) {
	tests := []struct {
		chainLen int
		tier     uint8
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
		s := &FlowState{ChainLen: tt.chainLen}
		if got := s.Tier(); got != tt.tier {
			t.Errorf("chainLen=%d: tier=%d, want %d", tt.chainLen, got, tt.tier)
		}
	}
}

func TestFlow_DamageMult(t *testing.T) {
	tests := []struct {
		chainLen int
		mastery  float32
		want     float32
	}{
		{0, 0, 1.0},
		{1, 0, 1.05},
		{3, 0, 1.15},
		{6, 0, 1.30},
		{3, 100, 1.30},  // 1.0 + 3*0.05*(1.0+1.0) = 1.30
		{6, 100, 1.60},  // 1.0 + 6*0.05*(2.0) = 1.60
		{6, 50, 1.45},   // 1.0 + 6*0.05*(1.5) = 1.45
	}
	for _, tt := range tests {
		s := &FlowState{ChainLen: tt.chainLen}
		got := s.DamageMult(tt.mastery)
		if math.Abs(float64(got-tt.want)) > 0.001 {
			t.Errorf("chainLen=%d mastery=%.0f: mult=%.3f, want %.3f",
				tt.chainLen, tt.mastery, got, tt.want)
		}
	}
}

func TestFlow_UniqueTransitionExtendsChain(t *testing.T) {
	s := &FlowState{}

	// Three different transitions: 0→1, 1→2, 2→3
	s.RecordTransition(0, 1, 0)
	if s.ChainLen != 1 {
		t.Errorf("chainLen = %d, want 1", s.ChainLen)
	}

	s.RecordTransition(1, 2, 0)
	if s.ChainLen != 2 {
		t.Errorf("chainLen = %d, want 2", s.ChainLen)
	}

	s.RecordTransition(2, 3, 0)
	if s.ChainLen != 3 {
		t.Errorf("chainLen = %d, want 3", s.ChainLen)
	}
}

func TestFlow_RepeatTransitionBreaksChain(t *testing.T) {
	s := &FlowState{}

	s.RecordTransition(0, 1, 0) // unique
	s.RecordTransition(1, 2, 0) // unique

	mult := s.RecordTransition(0, 1, 0) // repeat!
	if mult != 1.0 {
		t.Errorf("repeat mult = %f, want 1.0", mult)
	}
	if s.ChainLen != 0 {
		t.Errorf("chainLen = %d, want 0 (reset)", s.ChainLen)
	}
	if s.UsedTransitions != 0 {
		t.Errorf("usedTransitions = %d, want 0 (reset)", s.UsedTransitions)
	}
	if s.Timer != 0 {
		t.Errorf("timer = %f, want 0 (reset)", s.Timer)
	}
}

func TestFlow_TimerExpires(t *testing.T) {
	s := &FlowState{}
	s.RecordTransition(0, 1, 0)
	if s.Timer <= 0 {
		t.Fatal("timer should be positive after transition")
	}

	// Tick past the window
	s.Tick(20.0)
	if s.ChainLen != 0 {
		t.Errorf("chainLen = %d, want 0 after timer expiry", s.ChainLen)
	}
}

func TestFlow_WindowExtends(t *testing.T) {
	s := &FlowState{}

	// After 1 transition: window = 4 + 2*1 = 6
	s.RecordTransition(0, 1, 0)
	if math.Abs(float64(s.Timer-6.0)) > 0.01 {
		t.Errorf("timer = %f, want 6.0 after 1 transition", s.Timer)
	}

	// After 2 transitions: window = 4 + 2*2 = 8
	s.RecordTransition(1, 2, 0)
	if math.Abs(float64(s.Timer-8.0)) > 0.01 {
		t.Errorf("timer = %f, want 8.0 after 2 transitions", s.Timer)
	}

	// After 3: 4 + 2*3 = 10
	s.RecordTransition(2, 3, 0)
	if math.Abs(float64(s.Timer-10.0)) > 0.01 {
		t.Errorf("timer = %f, want 10.0 after 3 transitions", s.Timer)
	}

	// After 5: 4 + 2*5 = 14, but capped at 12
	s.RecordTransition(3, 0, 0)
	s.RecordTransition(0, 2, 0)
	if math.Abs(float64(s.Timer-12.0)) > 0.01 {
		t.Errorf("timer = %f, want 12.0 (capped) after 5 transitions", s.Timer)
	}
}

func TestFlow_AllTwentyTransitions(t *testing.T) {
	s := &FlowState{}

	count := 0
	for origin := 0; origin < 5; origin++ {
		for dest := 0; dest < 5; dest++ {
			if origin == dest {
				continue
			}
			s.RecordTransition(origin, dest, 0)
			count++
		}
	}

	if s.ChainLen != 20 {
		t.Errorf("chainLen = %d, want 20", s.ChainLen)
	}
	if count != 20 {
		t.Errorf("total valid transitions = %d, want 20", count)
	}
}

func TestFlow_DamageMultScalesPerStep(t *testing.T) {
	s := &FlowState{}

	// First transition: bonus from chainLen=0 → mult = 1.0
	mult1 := s.RecordTransition(0, 1, 0)
	if math.Abs(float64(mult1-1.0)) > 0.001 {
		t.Errorf("first transition mult = %f, want 1.0", mult1)
	}

	// Second transition: bonus from chainLen=1 → mult = 1.05
	mult2 := s.RecordTransition(1, 2, 0)
	if math.Abs(float64(mult2-1.05)) > 0.001 {
		t.Errorf("second transition mult = %f, want 1.05", mult2)
	}

	// Third transition: bonus from chainLen=2 → mult = 1.10
	mult3 := s.RecordTransition(2, 3, 0)
	if math.Abs(float64(mult3-1.10)) > 0.001 {
		t.Errorf("third transition mult = %f, want 1.10", mult3)
	}
}

func TestFlow_IntegrationWithCast(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	e := enemyInFront(100, 1e6)

	// Cast 1: shielded_sweep (Orbit→Fan) — first transition, no flow bonus
	p.Config = entity.ConfigOrbit
	p.GCDTimer = 0
	r1 := eng.Cast("shielded_sweep", castCtx(p, e))
	if !r1.OK {
		t.Fatalf("cast 1 failed: %s", r1.Reason)
	}

	flow := getFlowState(p)
	if flow.ChainLen != 1 {
		t.Errorf("after cast 1: chainLen = %d, want 1", flow.ChainLen)
	}

	// Cast 2: cleaving_pierce (Fan→Lance) — second transition, 5% bonus
	p.GCDTimer = 0
	hpBefore := e.Health
	r2 := eng.Cast("cleaving_pierce", castCtx(p, e))
	if !r2.OK {
		t.Fatalf("cast 2 failed: %s", r2.Reason)
	}
	dmg2 := hpBefore - e.Health
	// BaseDamage=30, flowMult=1.05, so dealt should be ~31.5
	if dmg2 < 31.0 || dmg2 > 32.0 {
		t.Errorf("cast 2 damage = %f, want ~31.5 (30 * 1.05)", dmg2)
	}

	if flow.ChainLen != 2 {
		t.Errorf("after cast 2: chainLen = %d, want 2", flow.ChainLen)
	}

	// Cast 3: repeat shielded_sweep (Orbit→Fan) — already used! Chain breaks.
	// But wait — we're in Lance config now, not Orbit. We need to be in Orbit.
	// Let's do a different chain: piercing_barrier (Lance→Orbit) first.
	p.GCDTimer = 0
	eng.Cast("piercing_barrier", castCtx(p, e)) // Lance→Orbit, chainLen=3
	if flow.ChainLen != 3 {
		t.Errorf("after cast 3: chainLen = %d, want 3", flow.ChainLen)
	}

	// Now in Orbit — cast shielded_sweep again (Orbit→Fan) which IS a repeat
	p.GCDTimer = 0
	eng.Cast("shielded_sweep", castCtx(p, e))
	if flow.ChainLen != 0 {
		t.Errorf("after repeat: chainLen = %d, want 0 (chain broken)", flow.ChainLen)
	}
}
