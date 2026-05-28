package entity

import "testing"

func TestEnemy_AddDebuff(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.3, Duration: 2.0})
	if !e.HasDebuff(DebuffSlow) {
		t.Fatal("expected slow debuff after AddDebuff")
	}
	if got := e.GetDebuffValue(DebuffSlow); got != 0.3 {
		t.Errorf("slow value = %f, want 0.3", got)
	}
}

func TestEnemy_AddDebuff_ReplaceByID(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.3, Duration: 2.0})
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.5, Duration: 3.0})
	if len(e.Debuffs) != 1 {
		t.Fatalf("expected 1 debuff after replace, got %d", len(e.Debuffs))
	}
	if got := e.GetDebuffValue(DebuffSlow); got != 0.5 {
		t.Errorf("slow value = %f, want 0.5 after replace", got)
	}
}

func TestEnemy_RemoveDebuff(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.3, Duration: 2.0})
	e.RemoveDebuff(testSlow1)
	if e.HasDebuff(DebuffSlow) {
		t.Fatal("expected no slow after RemoveDebuff")
	}
}

func TestEnemy_GetDebuffValue_ReturnsHighest(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.2, Duration: 2.0})
	e.AddDebuff(ActiveDebuff{ID: "slow2", Type: DebuffSlow, Value: 0.4, Duration: 2.0})
	if got := e.GetDebuffValue(DebuffSlow); got != 0.4 {
		t.Errorf("slow value = %f, want 0.4 (highest)", got)
	}
}

func TestEnemy_GetDebuffValue_ZeroWhenNone(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	if got := e.GetDebuffValue(DebuffSlow); got != 0 {
		t.Errorf("slow value = %f, want 0 when no debuffs", got)
	}
}

func TestEnemy_TickDebuffs_Expires(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.3, Duration: 1.0})
	e.TickDebuffs(0.5)
	if !e.HasDebuff(DebuffSlow) {
		t.Fatal("debuff should still be active after 0.5s")
	}
	e.TickDebuffs(0.6)
	if e.HasDebuff(DebuffSlow) {
		t.Fatal("debuff should have expired after 1.1s total")
	}
}

func TestEnemy_Reset_ClearsDebuffs(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: testSlow1, Type: DebuffSlow, Value: 0.3, Duration: 5.0})
	e.Reset(Vec3{})
	if len(e.Debuffs) != 0 {
		t.Fatalf("expected 0 debuffs after Reset, got %d", len(e.Debuffs))
	}
}

func TestEnemy_Vulnerability_IncreaseDamage(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: "vuln1", Type: DebuffVulnerability, Value: 0.2, Duration: 5.0})
	dealt := e.TargetApplyDamage(100)
	// 100 * 1.2 = 120
	if dealt < 119.9 || dealt > 120.1 {
		t.Errorf("dealt = %f, want ~120 with 20%% vulnerability", dealt)
	}
}

func TestEnemy_Root_HasDebuff(t *testing.T) {
	e := NewEnemy(1, 500, "test")
	e.AddDebuff(ActiveDebuff{ID: "root1", Type: DebuffRoot, Value: 1.0, Duration: 2.0})
	if !e.HasDebuff(DebuffRoot) {
		t.Fatal("expected root debuff")
	}
}
