package system

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestVitalCircuit_LinkExpiryHealsLowerHP(t *testing.T) {
	p1 := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	p1.Health = 100
	p2 := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
	p2.Health = 150

	players := map[uint16]*entity.Player{1: p1, 2: p2}
	w := makeWorld(t, players, nil)
	w.DamageLinks = append(w.DamageLinks, &entity.DamageLink{
		SourcePeer: 99,
		PeerA:      1,
		PeerB:      2,
		Duration:   0.1, // expires quickly
	})

	sys := CombatSystem{}
	for range 10 {
		sys.Tick(w, 0.05)
	}

	// Link expired. HP difference was 50. Lower-HP ally (p1) should be healed for 30% of 50 = 15.
	if p1.Health <= 100 {
		t.Errorf("p1 HP = %.1f, want > 100 (should be healed on link expiry)", p1.Health)
	}
	if len(w.DamageLinks) != 0 {
		t.Errorf("DamageLinks count = %d, want 0 (should have expired)", len(w.DamageLinks))
	}
}

func TestVitalCircuit_LinkDecaysDuration(t *testing.T) {
	p1 := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	p2 := entity.NewPlayer(2, entity.ClassArcanotechnicien)

	players := map[uint16]*entity.Player{1: p1, 2: p2}
	w := makeWorld(t, players, nil)
	w.DamageLinks = append(w.DamageLinks, &entity.DamageLink{
		SourcePeer: 99,
		PeerA:      1,
		PeerB:      2,
		Duration:   5.0,
	})

	sys := CombatSystem{}
	sys.Tick(w, 1.0)

	if len(w.DamageLinks) != 1 {
		t.Fatalf("DamageLinks count = %d, want 1 (should still be alive)", len(w.DamageLinks))
	}
	if w.DamageLinks[0].Duration > 4.1 {
		t.Errorf("Duration = %.1f, want ~4.0 (should have ticked down)", w.DamageLinks[0].Duration)
	}
}
