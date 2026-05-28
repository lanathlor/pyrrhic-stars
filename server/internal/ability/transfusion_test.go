package ability

import (
	"slices"
	"testing"

	"codex-online/server/internal/entity"
)

func TestTransfusion(t *testing.T) {
	eng := NewEngine(nil)

	t.Run("definition", func(t *testing.T) {
		def := eng.GetAbility(IDTransfusion)
		if def == nil {
			t.Fatal("transfusion not registered in engine")
		}
		if def.CommitTime != 4.0 {
			t.Errorf("CommitTime = %v, want 4.0", def.CommitTime)
		}
		if def.ExecuteTime != 0.1 {
			t.Errorf("ExecuteTime = %v, want 0.1", def.ExecuteTime)
		}
		wantCancel := uint8(CancelOnMove) | uint8(CancelOnDamage)
		if def.CancelConditions != wantCancel {
			t.Errorf("CancelConditions = %d, want %d", def.CancelConditions, wantCancel)
		}
		if def.OnCommitTick != IDTransfusion {
			t.Errorf("OnCommitTick = %q, want %q", def.OnCommitTick, IDTransfusion)
		}
		if def.Hit.Type != HitAllyTarget {
			t.Errorf("Hit.Type = %d, want HitAllyTarget (%d)", def.Hit.Type, HitAllyTarget)
		}
		if def.Hit.Range != 15 {
			t.Errorf("Hit.Range = %v, want 15", def.Hit.Range)
		}
		if def.GCD != 0.5 {
			t.Errorf("GCD = %v, want 0.5", def.GCD)
		}
		if len(def.Costs) != 1 || def.Costs[0].Resource != entity.ResourceFlux || def.Costs[0].Amount != 3 {
			t.Errorf("Costs = %+v, want [{flux 3}]", def.Costs)
		}
		if def.Delivery != uint8(entity.DeliveryBeam) {
			t.Errorf("Delivery = %d, want DeliveryBeam (%d)", def.Delivery, entity.DeliveryBeam)
		}
	})

	t.Run("handler", func(t *testing.T) {
		tests := []struct {
			name         string
			setup        func() (*entity.Player, map[uint16]*entity.Player, uint16)
			wantOK       bool
			wantReason   string
			wantTargetID uint16
		}{
			{
				name: "rejects when flux below 3",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.SetAllFluxPoolsCurrent(2) // need 3 biometabolic
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:     false,
				wantReason: tcInsufficientBiometabolicFlux,
			},
			{
				name: "rejects when flux is zero",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.SetAllFluxPoolsCurrent(0)
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:     false,
				wantReason: tcInsufficientBiometabolicFlux,
			},
			{
				name: "accepts when flux exactly 3",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					caster.Resources[entity.ResourceFlux].Current = 3
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:       true,
				wantTargetID: 2,
			},
			{
				name: "accepts when flux is plentiful",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					// default flux is 160 for harmonist
					ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
					allies := map[uint16]*entity.Player{1: caster, 2: ally}
					return caster, allies, 2
				},
				wantOK:       true,
				wantTargetID: 2,
			},
			{
				name: "stores ChannelTargetID on success",
				setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
					caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					ally := entity.NewPlayer(5, entity.ClassArcanotechnicien)
					allies := map[uint16]*entity.Player{1: caster, 5: ally}
					return caster, allies, 5
				},
				wantOK:       true,
				wantTargetID: 5,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				caster, allies, targetPeer := tt.setup()

				result := eng.Commit(IDTransfusion, &CommitContext{
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
				if tt.wantOK && tt.wantTargetID != 0 {
					if caster.ChannelTargetID != tt.wantTargetID {
						t.Errorf("ChannelTargetID = %d, want %d", caster.ChannelTargetID, tt.wantTargetID)
					}
				}
			})
		}
	})

	t.Run("harmonist spec includes transfusion in abilities list", func(t *testing.T) {
		p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
		found := slices.Contains(p.AllowedAbilities(), IDTransfusion)
		if !found {
			t.Error("transfusion not in AllowedAbilities (may not be in default loadout but should be equippable)")
		}
	})
}
