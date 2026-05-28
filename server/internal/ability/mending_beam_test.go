package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestMendingBeam(t *testing.T) {
	eng := NewEngine(nil)

	t.Run("definition", func(t *testing.T) {
		def := eng.GetAbility(IDMendingBeam)
		if def == nil {
			t.Fatal("mending_beam not registered in engine")
		}
		if def.CommitTime != 3.0 {
			t.Errorf("CommitTime = %v, want 3.0", def.CommitTime)
		}
		if def.ExecuteTime != 0.1 {
			t.Errorf("ExecuteTime = %v, want 0.1", def.ExecuteTime)
		}
		wantCancel := uint8(CancelOnMove) | uint8(CancelOnDamage)
		if def.CancelConditions != wantCancel {
			t.Errorf("CancelConditions = %d, want %d", def.CancelConditions, wantCancel)
		}
		if def.OnCommitTick != IDMendingBeam {
			t.Errorf("OnCommitTick = %q, want %q", def.OnCommitTick, IDMendingBeam)
		}
		if def.Hit.Type != HitAllyTarget {
			t.Errorf("Hit.Type = %d, want HitAllyTarget (%d)", def.Hit.Type, HitAllyTarget)
		}
		if def.Hit.Range != 20 {
			t.Errorf("Hit.Range = %v, want 20", def.Hit.Range)
		}
		if def.BaseHeal != 12 {
			t.Errorf("BaseHeal = %v, want 12", def.BaseHeal)
		}
		if def.HealScaling != "identity" {
			t.Errorf("HealScaling = %q, want %q", def.HealScaling, "identity")
		}
		if def.GCD != 0.5 {
			t.Errorf("GCD = %v, want 0.5", def.GCD)
		}
		if len(def.Costs) != 1 || def.Costs[0].Resource != entity.ResourceFlux || def.Costs[0].Amount != 8 {
			t.Errorf("Costs = %+v, want [{flux 8}]", def.Costs)
		}
	})

	t.Run("handler", func(t *testing.T) {
		tests := []struct {
			name       string
			setup      func() (*entity.Player, map[uint16]*entity.Player, uint16)
			wantOK     bool
			wantReason string
		}{
			{
				name: "rejects when flux below 10",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.SetAllFluxPoolsCurrent(5) // need 10 bioarcanotechnic
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					ally.Health = 50
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:     false,
				wantReason: tcInsufficientBioarcanotechnicFlux,
			},
			{
				name: "rejects when flux is zero",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.SetAllFluxPoolsCurrent(0)
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					ally.Health = 50
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:     false,
				wantReason: tcInsufficientBioarcanotechnicFlux,
			},
			{
				name: "accepts when flux exactly 10",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.Resources[entity.ResourceFlux].Current = 10
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					ally.Health = 50
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK: true,
			},
			{
				name: "accepts when flux is plentiful",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					// default flux is 160 for harmonist
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					ally.Health = 50
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK: true,
			},
			{
				name: "rejected on GCD by engine before handler",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.GCDTimer = 0.5
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					ally.Health = 50
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:     false,
				wantReason: ReasonGCD,
			},
			{
				name: "rejected on cooldown by engine before handler",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.Cooldowns[IDMendingBeam] = 1.0
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					ally.Health = 50
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:     false,
				wantReason: ReasonCooldown,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				caster, allies, targetPeer := tt.setup()

				result := eng.Commit(IDMendingBeam, &CommitContext{
					Committer:    caster,
					Allies:       allies,
					TargetPeerID: targetPeer,
				})

				if result.OK != tt.wantOK {
					t.Fatalf("OK = %v, want %v (reason: %q)", result.OK, tt.wantOK, result.Reason)
				}
				if !tt.wantOK && result.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
				}
			})
		}
	})

	t.Run("harmonist spec includes mending_beam", func(t *testing.T) {
		p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
		found := false
		for _, id := range []string{IDMendingBeam} {
			if abilityID, ok := p.ActionMap[51]; ok && abilityID == id {
				found = true
			}
		}
		if !found {
			t.Errorf("ActionMap[51] = %q, want %q", p.ActionMap[51], IDMendingBeam)
		}
	})
}
