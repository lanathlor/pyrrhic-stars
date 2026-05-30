package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
)

func TestTransfusionSustain_DrainsTargetHealsOthers(t *testing.T) {
	caster := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	target := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
	target.Health = 100 // will be drained
	other := entity.NewPlayerWithSpec(3, entity.ClassArcanotechnicien, "harmonist")
	other.Health = 80 // should receive healing

	players := map[uint16]*entity.Player{1: caster, 2: target, 3: other}
	w := makeWorld(t, players, nil)

	def := w.AbilityEngine.GetAbility("transfusion")
	if def == nil {
		t.Fatal("transfusion ability not registered")
	}

	runner := &ability.PlayerAbilityRunner{}
	runner.StartSustain(def, caster.Position, 50)
	w.AbilityRunners[1] = runner
	caster.ChannelTargetID = 2

	targetBefore := target.Health
	otherBefore := other.Health

	// Tick enough for one sustain tick (SustainInterval = 0.5s)
	sys := CombatSystem{}
	for range 12 {
		sys.Tick(w, 0.05)
	}

	if target.Health >= targetBefore {
		t.Errorf("target HP = %.1f, want less than %.1f (should be drained)", target.Health, targetBefore)
	}
	if other.Health <= otherBefore {
		t.Errorf("other HP = %.1f, want more than %.1f (should be healed)", other.Health, otherBefore)
	}
}

func TestTransfusionSustain_DoesNotHealTarget(t *testing.T) {
	caster := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	target := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
	target.Health = 100 // will be drained, should NOT be healed back
	// Only two players — no other ally to heal. Target should just lose HP.

	players := map[uint16]*entity.Player{1: caster, 2: target}
	w := makeWorld(t, players, nil)

	def := w.AbilityEngine.GetAbility("transfusion")
	if def == nil {
		t.Fatal("transfusion ability not registered")
	}

	runner := &ability.PlayerAbilityRunner{}
	runner.StartSustain(def, caster.Position, 50)
	w.AbilityRunners[1] = runner
	caster.ChannelTargetID = 2

	targetBefore := target.Health

	sys := CombatSystem{}
	for range 12 {
		sys.Tick(w, 0.05)
	}

	if target.Health >= targetBefore {
		t.Errorf("target HP = %.1f, should be less than %.1f (drained, not healed)", target.Health, targetBefore)
	}
}

func TestTransfusionSustain_CancelsWhenTargetDead(t *testing.T) {
	caster := entity.NewPlayerWithSpec(1, entity.ClassArcanotechnicien, "harmonist")
	target := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
	target.Alive = false // dead target
	target.Health = 0

	players := map[uint16]*entity.Player{1: caster, 2: target}
	w := makeWorld(t, players, nil)

	def := w.AbilityEngine.GetAbility("transfusion")
	if def == nil {
		t.Fatal("transfusion ability not registered")
	}

	runner := &ability.PlayerAbilityRunner{}
	runner.StartSustain(def, caster.Position, 50)
	w.AbilityRunners[1] = runner
	caster.ChannelTargetID = 2

	sys := CombatSystem{}
	for range 12 {
		sys.Tick(w, 0.05)
	}

	if runner.Phase == ability.PRunnerSustain {
		t.Error("runner should not be sustaining when target is dead")
	}
}
