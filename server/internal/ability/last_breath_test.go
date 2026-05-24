package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestLastBreath(t *testing.T) {
	eng := NewEngine(nil)

	t.Run("applies death prevention buff to target", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		result := eng.Commit("last_breath", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if !result.OK {
			t.Fatalf("OK = false, reason: %q", result.Reason)
		}
		if !ally.HasBuff("last_breath") {
			t.Fatal("ally should have last_breath buff")
		}
		buff := ally.GetBuff("last_breath")
		if buff.Type != entity.BuffDeathPrevention {
			t.Errorf("buff Type = %q, want %q", buff.Type, entity.BuffDeathPrevention)
		}
		if buff.Duration < 3.5 || buff.Duration > 4.5 {
			t.Errorf("buff Duration = %f, want ~4.0", buff.Duration)
		}
	})

	t.Run("records caster ID on target", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		eng.Commit("last_breath", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if ally.LastBreathCasterID != 1 {
			t.Errorf("LastBreathCasterID = %d, want 1", ally.LastBreathCasterID)
		}
	})

	t.Run("spends biometabolic flux and sets 60s cooldown", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		before := caster.Resources["flux"].Current
		eng.Commit("last_breath", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		after := caster.Resources["flux"].Current
		if after >= before {
			t.Errorf("flux not spent: before=%.1f after=%.1f", before, after)
		}
		if cd := caster.Cooldowns["last_breath"]; cd < 59 {
			t.Errorf("cooldown = %f, want ~60", cd)
		}
	})

	t.Run("grants confluence stack", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		eng.Commit("last_breath", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if caster.Confluence.Stacks != 1 {
			t.Errorf("Confluence.Stacks = %d, want 1", caster.Confluence.Stacks)
		}
	})

	t.Run("rejects on insufficient flux", func(t *testing.T) {
		caster := newHarmonist(1)
		caster.SetAllFluxPoolsCurrent(2)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		result := eng.Commit("last_breath", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if result.OK {
			t.Fatal("should reject on insufficient flux")
		}
	})

	t.Run("rejects on cooldown", func(t *testing.T) {
		caster := newHarmonist(1)
		caster.Cooldowns["last_breath"] = 30.0
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		result := eng.Commit("last_breath", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if result.OK {
			t.Fatal("should reject on cooldown")
		}
	})
}

func TestLastBreath_DeathPrevention(t *testing.T) {
	t.Run("prevents lethal damage", func(t *testing.T) {
		p := newHarmonist(1)
		p.Health = 50
		p.AddBuff(entity.ActiveBuff{
			ID:       "last_breath",
			Type:     entity.BuffDeathPrevention,
			Duration: 4.0,
		})

		p.ApplyDamage(200) // should be lethal

		if !p.Alive {
			t.Error("player should be alive (death prevented)")
		}
		if p.Health != 1 {
			t.Errorf("Health = %f, want 1", p.Health)
		}
		if p.LastBreathPrevented <= 0 {
			t.Error("LastBreathPrevented should be > 0")
		}
	})

	t.Run("without buff player dies normally", func(t *testing.T) {
		p := newHarmonist(1)
		p.Health = 50

		p.ApplyDamage(200) // lethal, no buff

		if p.Alive {
			t.Error("player should be dead (no death prevention)")
		}
	})
}
