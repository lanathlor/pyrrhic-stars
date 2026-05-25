package system

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestHoT_TicksHealTarget(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	p.Health = 100
	p.HoTs = append(p.HoTs, entity.ActiveHoT{
		ID:          "regen_protocol",
		SourcePeer:  99,
		HealPerTick: 10,
		Remaining:   5.0,
		Interval:    1.0,
		TickTimer:   1.0,
	})

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	// Tick for 1.1 seconds — should fire one HoT tick
	for range 22 {
		sys.Tick(w, 0.05)
	}

	if p.Health <= 100 {
		t.Errorf("HP = %.1f, want > 100 (HoT should have healed)", p.Health)
	}
}

func TestHoT_Expires(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	p.Health = 100
	p.HoTs = append(p.HoTs, entity.ActiveHoT{
		ID:          "regen_protocol",
		SourcePeer:  99,
		HealPerTick: 10,
		Remaining:   0.5,
		Interval:    1.0,
		TickTimer:   1.0,
	})

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}

	// Tick for 1 second — HoT should expire
	for range 20 {
		sys.Tick(w, 0.05)
	}

	if len(p.HoTs) != 0 {
		t.Errorf("HoTs count = %d, want 0 (should have expired)", len(p.HoTs))
	}
}

func TestHoT_BurstAtLowHP(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	p.MaxHealth = 200
	p.Health = 40 // 20% — below 30% threshold

	p.HoTs = append(p.HoTs, entity.ActiveHoT{
		ID:             "regen_protocol",
		SourcePeer:     99,
		HealPerTick:    10,
		Remaining:      10.0, // 10 ticks remaining
		Interval:       1.0,
		TickTimer:      1.0,
		BurstThreshold: 0.3,
	})

	w := makeWorld(map[uint16]*entity.Player{1: p}, nil)
	sys := CombatSystem{}
	sys.Tick(w, 0.05)

	// Should burst-consume: 10 remaining ticks * 10 HP = 100 HP burst heal
	// 40 + 100 = 140
	if p.Health < 100 {
		t.Errorf("HP = %.1f, want >= 100 (burst should have healed significantly)", p.Health)
	}
	if len(p.HoTs) != 0 {
		t.Errorf("HoTs count = %d, want 0 (burst should consume the HoT)", len(p.HoTs))
	}
}
