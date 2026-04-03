package entity

import "testing"

func TestAddThreat(t *testing.T) {
	e := NewEnemy(0)
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
	e := NewEnemy(0)
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
	e := NewEnemy(0)
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

func TestResetClearsThreat(t *testing.T) {
	e := NewEnemy(0)
	e.AddThreat(1, 50.0)
	e.AddThreat(3, 100.0)
	e.Reset(Vec3{})

	if e.HasThreat(1) || e.HasThreat(3) {
		t.Error("threat table not cleared after Reset()")
	}
}
