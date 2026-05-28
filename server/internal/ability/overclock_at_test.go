package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestOverclockAT(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name           string
		setup          func() (*entity.Player, map[uint16]*entity.Player, uint16)
		wantOK         bool
		wantReason     string
		wantAllyBuffs  bool   // check buffs on target ally
		wantAllyBuffID uint16 // which ally to check buffs on
		wantConfluence int
		wantCooldown   bool
		wantGCD        bool
		wantFlux       float32 // -1 to skip
	}{
		{
			name: "buffs target ally with attack speed and move speed",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantAllyBuffs:  true,
			wantAllyBuffID: 2,
			wantConfluence: 1,
			wantCooldown:   true,
			wantGCD:        true,
			wantFlux:       50, // bioarcanotechnic pool: 160*0.5=80, 80-30=50
		},
		{
			name: "spends bioarcanotechnic flux",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantAllyBuffs:  true,
			wantAllyBuffID: 2,
			wantConfluence: 1,
			wantCooldown:   true,
			wantGCD:        true,
			wantFlux:       50,
		},
		{
			name: "sets GCD to 0.8",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantAllyBuffs:  true,
			wantAllyBuffID: 2,
			wantConfluence: 1,
			wantCooldown:   true,
			wantGCD:        true,
			wantFlux:       50,
		},
		{
			name: "sets cooldown",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantAllyBuffs:  true,
			wantAllyBuffID: 2,
			wantConfluence: 1,
			wantCooldown:   true,
			wantGCD:        true,
			wantFlux:       50,
		},
		{
			name: tcGrantsConfluence,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantAllyBuffs:  true,
			wantAllyBuffID: 2,
			wantConfluence: 1,
			wantCooldown:   true,
			wantGCD:        true,
			wantFlux:       50,
		},
		{
			name: "falls back to self-buff if target peer ID is invalid",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 999
			},
			wantOK:         true,
			wantAllyBuffs:  true,
			wantAllyBuffID: 1, // self
			wantConfluence: 1,
			wantCooldown:   true,
			wantGCD:        true,
			wantFlux:       50,
		},
		{
			name: tcRejectsInsufficientFlux,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.SetAllFluxPoolsCurrent(5)
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: tcInsufficientBioarcanotechnicFlux,
			wantFlux:   -1,
		},
		{
			name: tcRejectsGCD,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.GCDTimer = 0.5
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: ReasonGCD,
			wantFlux:   -1,
		},
		{
			name: tcRejectsCooldown,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.Cooldowns[IDOverclockAT] = 1.0
				ally := entity.NewPlayer(2, entity.ClassGunner)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: ReasonCooldown,
			wantFlux:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caster, allies, targetPeer := tt.setup()

			result := eng.Commit(IDOverclockAT, &CommitContext{
				Committer:    caster,
				Allies:       allies,
				TargetPeerID: targetPeer,
			})

			if result.OK != tt.wantOK {
				t.Fatalf("OK = %v, want %v (reason: %q)", result.OK, tt.wantOK, result.Reason)
			}
			if !tt.wantOK {
				if result.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
				}
				return
			}

			// Check buffs on the target ally
			if tt.wantAllyBuffs {
				target := allies[tt.wantAllyBuffID]
				if target == nil {
					t.Fatalf("ally %d not found in allies map", tt.wantAllyBuffID)
				}
				if !target.HasBuff("overclock_at_speed") {
					t.Error("expected target to have overclock_at_speed buff")
				}
				if !target.HasBuff("overclock_at_move") {
					t.Error("expected target to have overclock_at_move buff")
				}

				// Verify buff values
				speedBuff := target.GetBuff("overclock_at_speed")
				if speedBuff == nil {
					t.Fatal("overclock_at_speed buff is nil")
				}
				if speedBuff.Type != entity.BuffAttackSpeed {
					t.Errorf("overclock_at_speed type = %q, want %q", speedBuff.Type, entity.BuffAttackSpeed)
				}
				if speedBuff.Value != 1.15 {
					t.Errorf("overclock_at_speed value = %v, want 1.15", speedBuff.Value)
				}
				if speedBuff.Duration < 5.9 || speedBuff.Duration > 6.1 {
					t.Errorf("overclock_at_speed duration = %v, want ~6.0", speedBuff.Duration)
				}

				moveBuff := target.GetBuff("overclock_at_move")
				if moveBuff == nil {
					t.Fatal("overclock_at_move buff is nil")
				}
				if moveBuff.Type != entity.BuffMoveSpeed {
					t.Errorf("overclock_at_move type = %q, want %q", moveBuff.Type, entity.BuffMoveSpeed)
				}
				if moveBuff.Value != 1.10 {
					t.Errorf("overclock_at_move value = %v, want 1.10", moveBuff.Value)
				}
				if moveBuff.Duration < 5.9 || moveBuff.Duration > 6.1 {
					t.Errorf("overclock_at_move duration = %v, want ~6.0", moveBuff.Duration)
				}
			}

			// Check confluence
			if tt.wantConfluence > 0 {
				if caster.Confluence == nil {
					t.Fatal("Confluence is nil")
				}
				if caster.Confluence.Stacks != tt.wantConfluence {
					t.Errorf("Confluence.Stacks = %d, want %d", caster.Confluence.Stacks, tt.wantConfluence)
				}
			}

			// Check cooldown
			if tt.wantCooldown {
				if cd, ok := caster.Cooldowns[IDOverclockAT]; !ok || cd <= 0 {
					t.Error("expected cooldown to be set")
				} else if cd != 15.0 {
					t.Errorf("cooldown = %v, want 15.0", cd)
				}
			}

			// Check GCD
			if tt.wantGCD {
				if caster.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				}
				if caster.GCDTimer != 0.8 {
					t.Errorf("GCD = %v, want 0.8", caster.GCDTimer)
				}
			}

			// Check flux (bioarcanotechnic pool)
			if tt.wantFlux >= 0 && caster.FluxCommit != nil {
				pool := caster.FluxCommit.GetPool("bioarcanotechnic")
				if pool == nil {
					t.Fatal("bioarcanotechnic pool is nil")
				}
				if pool.Current != tt.wantFlux {
					t.Errorf("bioarcanotechnic flux = %v, want %v", pool.Current, tt.wantFlux)
				}
			}
		})
	}
}
