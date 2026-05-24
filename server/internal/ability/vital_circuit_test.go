package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestVitalCircuit(t *testing.T) {
	eng := NewEngine(nil)

	t.Run("creates damage link between caster and target", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		var spawnedLink *entity.DamageLink
		result := eng.Commit("vital_circuit", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
			SpawnLink: func(link *entity.DamageLink) {
				spawnedLink = link
			},
		})

		if !result.OK {
			t.Fatalf("OK = false, reason: %q", result.Reason)
		}
		if spawnedLink == nil {
			t.Fatal("SpawnLink was not called")
		}
		if spawnedLink.PeerA != 1 || spawnedLink.PeerB != 2 {
			t.Errorf("link peers = (%d, %d), want (1, 2)", spawnedLink.PeerA, spawnedLink.PeerB)
		}
		if spawnedLink.Duration < 7.5 || spawnedLink.Duration > 8.5 {
			t.Errorf("link Duration = %f, want ~8.0", spawnedLink.Duration)
		}
		if spawnedLink.SourcePeer != 1 {
			t.Errorf("link SourcePeer = %d, want 1", spawnedLink.SourcePeer)
		}
	})

	t.Run("spends biometabolic flux", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		before := caster.Resources[entity.ResourceFlux].Current
		eng.Commit("vital_circuit", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
			SpawnLink:    func(*entity.DamageLink) {},
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

		eng.Commit("vital_circuit", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
			SpawnLink:    func(*entity.DamageLink) {},
		})

		if caster.GCDTimer <= 0 {
			t.Error("GCD not set")
		}
		if cd := caster.Cooldowns["vital_circuit"]; cd <= 0 {
			t.Error("cooldown not set")
		}
	})

	t.Run("grants confluence stack", func(t *testing.T) {
		caster := newHarmonist(1)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		eng.Commit("vital_circuit", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
			SpawnLink:    func(*entity.DamageLink) {},
		})

		if caster.Confluence.Stacks != 1 {
			t.Errorf("Confluence.Stacks = %d, want 1", caster.Confluence.Stacks)
		}
	})

	t.Run("rejects without valid target", func(t *testing.T) {
		caster := newHarmonist(1)
		allies := map[uint16]*entity.Player{1: caster}

		result := eng.Commit("vital_circuit", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 999,
			SpawnLink:    func(*entity.DamageLink) {},
		})

		if result.OK {
			t.Fatal("should reject without valid target")
		}
		if result.Reason != "no valid target" {
			t.Errorf("Reason = %q, want %q", result.Reason, "no valid target")
		}
	})

	t.Run("rejects on insufficient flux", func(t *testing.T) {
		caster := newHarmonist(1)
		caster.SetAllFluxPoolsCurrent(2)
		ally := newHarmonist(2)
		allies := map[uint16]*entity.Player{1: caster, 2: ally}

		result := eng.Commit("vital_circuit", &CommitContext{
			Committer:    caster,
			Allies:       allies,
			TargetPeerID: 2,
			SpawnLink:    func(*entity.DamageLink) {},
		})

		if result.OK {
			t.Fatal("should reject on insufficient flux")
		}
	})
}
