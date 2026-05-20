package ability

import (
	"testing"
)

func TestRetaliate_ConsumesAllDevotion(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)

	dev := getDevotionState(p)
	dev.Charges = 50

	r := eng.Cast("retaliate", castCtx(p, enemy))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}

	if dev.Charges != 0 {
		t.Errorf("Devotion charges = %f, want 0 after Retaliate", dev.Charges)
	}
}

func TestRetaliate_DamageScalesWithCharges(t *testing.T) {
	eng := NewEngine(nil)

	// Zero charges
	p0 := newShieldVanguard()
	enemy0 := enemyInFront(100, 1000)
	eng.Cast("retaliate", castCtx(p0, enemy0))
	dmg0 := float32(1000) - enemy0.Health

	// 50 charges
	p50 := newShieldVanguard()
	enemy50 := enemyInFront(101, 1000)
	dev := getDevotionState(p50)
	dev.Charges = 50
	eng.Cast("retaliate", castCtx(p50, enemy50))
	dmg50 := float32(1000) - enemy50.Health

	if dmg50 <= dmg0 {
		t.Errorf("damage with 50 charges (%f) should be > damage with 0 charges (%f)", dmg50, dmg0)
	}
}

func TestRetaliate_DamageScalesWithMastery(t *testing.T) {
	eng := NewEngine(nil)

	// 50 charges, 0 mastery
	p0 := newShieldVanguard()
	enemy0 := enemyInFront(100, 1000)
	dev0 := getDevotionState(p0)
	dev0.Charges = 50
	eng.Cast("retaliate", castCtx(p0, enemy0))
	dmg0 := float32(1000) - enemy0.Health

	// 50 charges, 100 mastery
	p100 := newShieldVanguard()
	p100.GearStats.Mastery = 100
	p100.RecalcStats()
	enemy100 := enemyInFront(101, 1000)
	dev100 := getDevotionState(p100)
	dev100.Charges = 50
	eng.Cast("retaliate", castCtx(p100, enemy100))
	dmg100 := float32(1000) - enemy100.Health

	if dmg100 <= dmg0 {
		t.Errorf("damage with mastery 100 (%f) should be > damage with mastery 0 (%f)", dmg100, dmg0)
	}
}

func TestRetaliate_DropsGuard(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("vg_shield_block", castCtx(p))
	if !p.HasBuff("vg_shield_block") {
		t.Fatal("should be blocking")
	}

	eng.Cast("retaliate", castCtx(p))
	if p.HasBuff("vg_shield_block") {
		t.Error("Retaliate should cancel shield block")
	}
}

func TestRetaliate_SetsGCD(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("retaliate", castCtx(p))

	if p.GCDTimer < 1.4 {
		t.Errorf("GCD = %f, want >= 1.5", p.GCDTimer)
	}
}

func TestRetaliate_ZeroChargesStillDealsBaseDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 1000)

	r := eng.Cast("retaliate", castCtx(p, enemy))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("should deal damage even with 0 charges")
	}
	if r.Events[0].Amount <= 0 {
		t.Error("base damage should be positive")
	}
}

func TestRetaliate_HitsWideArc(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	// Place enemies on both sides (within 180° arc)
	left := enemyInFront(100, 500)
	left.Position.X = -3
	left.Position.Z = -3
	right := enemyInFront(101, 500)
	right.Position.X = 3
	right.Position.Z = -3

	r := eng.Cast("retaliate", castCtx(p, left, right))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) < 2 {
		t.Errorf("expected 2 hits from wide arc, got %d", len(r.Events))
	}
}
