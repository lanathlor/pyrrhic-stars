package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestRegenProtocol(t *testing.T) {
	eng := NewEngine(nil)

	t.Run("applies HoT to target ally", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		ally.Health = 100
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		result := eng.Commit(IDRegenProtocol, &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if !result.OK {
			t.Fatalf("OK = false, reason: %q", result.Reason)
		}
		if len(ally.HoTs) != 1 {
			t.Fatalf("ally HoTs count = %d, want 1", len(ally.HoTs))
		}
		hot := ally.HoTs[0]
		if hot.ID != IDRegenProtocol {
			t.Errorf("HoT ID = %q, want %q", hot.ID, IDRegenProtocol)
		}
		if hot.SourcePeer != 1 {
			t.Errorf("HoT SourcePeer = %d, want 1", hot.SourcePeer)
		}
		if hot.Remaining < 11.5 || hot.Remaining > 12.5 {
			t.Errorf("HoT Remaining = %f, want ~12.0", hot.Remaining)
		}
		if hot.BurstThreshold < 0.29 || hot.BurstThreshold > 0.31 {
			t.Errorf("HoT BurstThreshold = %f, want ~0.3", hot.BurstThreshold)
		}
	})

	t.Run("spends bioarcanotechnic flux", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		ally.Health = 100
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		before := caster.Resources[entity.ResourceFlux].Current
		eng.Commit(IDRegenProtocol, &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		after := caster.Resources[entity.ResourceFlux].Current
		if after >= before {
			t.Errorf("flux not spent: before=%.1f after=%.1f", before, after)
		}
	})

	t.Run("sets GCD and cooldown", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		eng.Commit(IDRegenProtocol, &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if caster.GCDTimer <= 0 {
			t.Error("GCD not set")
		}
		if cd := caster.Cooldowns[IDRegenProtocol]; cd <= 0 {
			t.Error("cooldown not set")
		}
	})

	t.Run(tcGrantsConfluence, func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		eng.Commit(IDRegenProtocol, &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if caster.Confluence.Stacks != 1 {
			t.Errorf("Confluence.Stacks = %d, want 1", caster.Confluence.Stacks)
		}
	})

	t.Run("falls back to self when target invalid", func(t *testing.T) {
		caster := newHarmonist(1)
		caster.Health = 100
		allies := map[uint16]*entity.Player{1: caster}

		result := eng.Commit(IDRegenProtocol, &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 999,
		})

		if !result.OK {
			t.Fatalf("OK = false, reason: %q", result.Reason)
		}
		if len(caster.HoTs) != 1 {
			t.Fatalf("self HoTs count = %d, want 1", len(caster.HoTs))
		}
	})

	t.Run(tcRejectsInsufficientFlux, func(t *testing.T) {
		caster := newHarmonist(1)
		caster.SetAllFluxPoolsCurrent(2)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		result := eng.Commit(IDRegenProtocol, &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
		})

		if result.OK {
			t.Fatal("should reject on insufficient flux")
		}
		if result.Reason != tcInsufficientBioarcanotechnicFlux {
			t.Errorf("Reason = %q, want %q", result.Reason, tcInsufficientBioarcanotechnicFlux)
		}
	})
}
