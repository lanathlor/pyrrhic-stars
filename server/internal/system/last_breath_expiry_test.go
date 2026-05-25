package system

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestLastBreath_ExpiryDamagesCaster(t *testing.T) {
	caster := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	target := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")

	// Target has last_breath buff about to expire
	target.AddBuff(entity.ActiveBuff{
		ID:       "last_breath",
		Type:     entity.BuffDeathPrevention,
		Duration: 0.1, // expires quickly
	})
	target.LastBreathCasterID = 1
	target.LastBreathPrevented = 100 // 100 damage was prevented

	players := map[uint16]*entity.Player{1: caster, 2: target}
	w := makeWorld(players, nil)

	casterHPBefore := caster.Health

	sys := CombatSystem{}
	// Tick enough for the buff to expire
	for range 10 {
		sys.Tick(w, 0.05)
	}

	// Caster should take 50% of prevented damage = 50 HP
	if !target.HasBuff("last_breath") {
		// Buff expired — check caster damage
		expectedDamage := float32(50.0) // 50% of 100
		actualDamage := casterHPBefore - caster.Health
		if actualDamage < expectedDamage*0.8 || actualDamage > expectedDamage*1.2 {
			t.Errorf("caster took %.1f damage, want ~%.1f (50%% of prevented)", actualDamage, expectedDamage)
		}
	}

	// Target fields should be reset
	if target.LastBreathPrevented != 0 {
		t.Errorf("LastBreathPrevented = %.1f, want 0", target.LastBreathPrevented)
	}
	if target.LastBreathCasterID != 0 {
		t.Errorf("LastBreathCasterID = %d, want 0", target.LastBreathCasterID)
	}
}

func TestLastBreath_NoDamageWhenNothingPrevented(t *testing.T) {
	caster := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	target := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")

	target.AddBuff(entity.ActiveBuff{
		ID:       "last_breath",
		Type:     entity.BuffDeathPrevention,
		Duration: 0.1,
	})
	target.LastBreathCasterID = 1
	target.LastBreathPrevented = 0 // nothing prevented

	players := map[uint16]*entity.Player{1: caster, 2: target}
	w := makeWorld(players, nil)

	casterHPBefore := caster.Health

	sys := CombatSystem{}
	for range 10 {
		sys.Tick(w, 0.05)
	}

	if caster.Health != casterHPBefore {
		t.Errorf("caster HP = %.1f, want %.1f (no self-damage when nothing prevented)", caster.Health, casterHPBefore)
	}
}
