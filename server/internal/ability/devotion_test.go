package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func newShieldVanguard() *entity.Player {
	p := entity.NewPlayerWithSpec(10, entity.ClassVanguard, "shield")
	p.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p.RotationY = 0
	return p
}

func TestDevotion_StartsAtZero(t *testing.T) {
	p := newShieldVanguard()
	dev := getDevotionState(p)
	if dev.Charges != 0 {
		t.Errorf("charges = %f, want 0", dev.Charges)
	}
	if dev.Tier() != TierStandard {
		t.Errorf("tier = %d, want %d (standard)", dev.Tier(), TierStandard)
	}
}

func TestDevotion_AddCharges(t *testing.T) {
	dev := &DevotionState{}
	// 50 damage absorbed, 0 mastery → 50 * 0.15 = 7.5 charges
	dev.AddCharges(50, 0)
	if dev.Charges < 7.4 || dev.Charges > 7.6 {
		t.Errorf("charges = %f, want 7.5", dev.Charges)
	}
}

func TestDevotion_AddChargesWithMastery(t *testing.T) {
	dev := &DevotionState{}
	// 50 damage, 100 mastery → 50 * (0.15 + 100/500) = 50 * 0.35 = 17.5
	dev.AddCharges(50, 100)
	if dev.Charges < 17.4 || dev.Charges > 17.6 {
		t.Errorf("charges = %f, want 17.5", dev.Charges)
	}
}

func TestDevotion_Tiers(t *testing.T) {
	tests := []struct {
		charges float32
		tier    uint8
	}{
		{0, TierStandard},
		{15, TierStandard},
		{29.9, TierStandard},
		{30, TierEmpowered},
		{45, TierEmpowered},
		{59.9, TierEmpowered},
		{60, TierMaximum},
		{100, TierMaximum},
	}
	for _, tc := range tests {
		dev := &DevotionState{Charges: tc.charges}
		if got := dev.Tier(); got != tc.tier {
			t.Errorf("Tier() at %.1f charges = %d, want %d", tc.charges, got, tc.tier)
		}
	}
}

func TestDevotion_ConsumeAll(t *testing.T) {
	dev := &DevotionState{Charges: 42.5}
	got := dev.ConsumeAll()
	if got < 42.4 || got > 42.6 {
		t.Errorf("ConsumeAll() = %f, want 42.5", got)
	}
	if dev.Charges != 0 {
		t.Errorf("charges after consume = %f, want 0", dev.Charges)
	}
}

func TestDevotion_ConsumeAllWhenEmpty(t *testing.T) {
	dev := &DevotionState{}
	got := dev.ConsumeAll()
	if got != 0 {
		t.Errorf("ConsumeAll() when empty = %f, want 0", got)
	}
}

func TestDevotion_Reset(t *testing.T) {
	dev := &DevotionState{Charges: 50}
	dev.Reset()
	if dev.Charges != 0 {
		t.Errorf("charges after reset = %f, want 0", dev.Charges)
	}
}

func TestDevotion_StackCount(t *testing.T) {
	dev := &DevotionState{Charges: 42.7}
	if got := dev.StackCount(); got != 42 {
		t.Errorf("StackCount() = %d, want 42", got)
	}
}

func TestDevotion_GetDevotionState_CreatesIfMissing(t *testing.T) {
	p := newShieldVanguard()
	dev := getDevotionState(p)
	if dev == nil {
		t.Fatal("getDevotionState returned nil")
	}
	// Second call returns same instance
	dev2 := getDevotionState(p)
	if dev != dev2 {
		t.Error("getDevotionState should return same instance")
	}
}
