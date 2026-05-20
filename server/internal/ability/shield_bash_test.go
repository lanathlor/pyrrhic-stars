package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestShieldBash_HitsEnemy(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	enemy.Position.Z = -3 // within shield bash range (4)

	r := eng.Cast("shield_bash", castCtx(p, enemy))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) == 0 {
		t.Error("expected damage events")
	}
	if r.Events[0].Amount <= 0 {
		t.Error("damage should be positive")
	}
}

func TestShieldBash_AppliesSlowDebuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	enemy.Position.Z = -3

	eng.Cast("shield_bash", castCtx(p, enemy))

	if !enemy.HasDebuff(entity.DebuffSlow) {
		t.Error("enemy should have slow debuff from Shield Bash")
	}
}

func TestShieldBash_CostsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	initial := p.GetResource("stamina")

	eng.Cast("shield_bash", castCtx(p))

	if p.GetResource("stamina") >= initial {
		t.Error("stamina should decrease after Shield Bash")
	}
}

func TestShieldBash_WorksDuringBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	enemy.Position.Z = -3

	// Start blocking
	eng.Cast("vg_shield_block", castCtx(p, enemy))
	if !p.HasBuff("vg_shield_block") {
		t.Fatal("should be blocking")
	}

	// Shield Bash should work without cancelling block
	r := eng.Cast("shield_bash", castCtx(p, enemy))
	if !r.OK {
		t.Fatalf("shield bash during block failed: %s", r.Reason)
	}
	if !p.HasBuff("vg_shield_block") {
		t.Error("shield block should NOT be cancelled by Shield Bash")
	}
}

func TestShieldBash_GeneratesDevotion(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)
	enemy.Position.Z = -3

	eng.Cast("shield_bash", castCtx(p, enemy))

	dev := getDevotionState(p)
	if dev.Charges <= 0 {
		t.Error("Shield Bash should generate Devotion on hit")
	}
}

func TestShieldBash_InsufficientStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 0

	r := eng.Cast("shield_bash", castCtx(p))
	if r.OK {
		t.Error("should fail with 0 stamina")
	}
}

func TestShieldBash_SetsGCD(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Cast("shield_bash", castCtx(p))

	if p.GCDTimer <= 0 {
		t.Error("GCD should be set after Shield Bash")
	}
}

func TestShieldBash_MissesEnemyBehind(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyBehind(100, 500)

	r := eng.Cast("shield_bash", castCtx(p, enemy))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("should not hit enemy behind player")
	}
}
