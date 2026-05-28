package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestNeuralFortification(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name         string
		setup        func() (*entity.Player, map[uint16]*entity.Player, uint16)
		wantOK       bool
		wantReason   string
		wantDRBuff   bool
		wantDRValue  float32
		wantCCBuff   bool
		wantCCValue  float32
		wantFlux     float32
		wantCooldown bool
		wantGCD      bool
	}{
		{
			name: "applies DR buff to target",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:       true,
			wantDRBuff:   true,
			wantDRValue:  0.8,
			wantCCBuff:   true,
			wantCCValue:  1.0,
			wantFlux:     120, // 160 - 40
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "applies CC immunity buff to target",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:      true,
			wantCCBuff:  true,
			wantCCValue: 1.0,
			wantDRBuff:  true,
			wantDRValue: 0.8,
			wantFlux:    120,
		},
		{
			name: "spends bioarcanotechnic flux",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:       true,
			wantFlux:     120, // 160 - 40
			wantDRBuff:   true,
			wantDRValue:  0.8,
			wantCCBuff:   true,
			wantCCValue:  1.0,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "sets GCD and cooldown",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:       true,
			wantFlux:     120,
			wantDRBuff:   true,
			wantDRValue:  0.8,
			wantCCBuff:   true,
			wantCCValue:  1.0,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: tcGrantsConfluence,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:      true,
			wantFlux:    120,
			wantDRBuff:  true,
			wantDRValue: 0.8,
			wantCCBuff:  true,
			wantCCValue: 1.0,
		},
		{
			name: "falls back to self-buff if target invalid",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 999
			},
			wantOK:       true,
			wantDRBuff:   true,
			wantDRValue:  0.8,
			wantCCBuff:   true,
			wantCCValue:  1.0,
			wantFlux:     120,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: tcRejectsInsufficientFlux,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.SetAllFluxPoolsCurrent(5)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: tcInsufficientBioarcanotechnicFlux,
			wantFlux:   -1, // skip flux check
		},
		{
			name: tcRejectsGCD,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.GCDTimer = 0.5
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: ReasonGCD,
			wantFlux:   160,
		},
		{
			name: tcRejectsCooldown,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.Cooldowns["neural_fortification"] = 5.0
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: ReasonCooldown,
			wantFlux:   160,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caster, allies, targetPeer := tt.setup()

			// Record confluence stacks before commit.
			var stacksBefore int
			if caster.Confluence != nil {
				stacksBefore = caster.Confluence.Stacks
			}

			result := eng.Commit("neural_fortification", &CommitContext{
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
			}

			// Determine buff target: ally if valid, otherwise self.
			var buffTarget *entity.Player
			if tt.wantOK {
				if a, ok := allies[targetPeer]; ok && a.ID != caster.ID {
					buffTarget = a
				} else {
					buffTarget = caster
				}
			}

			if tt.wantDRBuff && buffTarget != nil {
				if !buffTarget.HasBuff("neural_fort_dr") {
					t.Error("expected neural_fort_dr buff on target")
				} else {
					b := buffTarget.GetBuff("neural_fort_dr")
					if b.Type != entity.BuffDamageReduction {
						t.Errorf("DR buff type = %q, want %q", b.Type, entity.BuffDamageReduction)
					}
					if math.Abs(float64(b.Value-tt.wantDRValue)) > 0.01 {
						t.Errorf("DR buff value = %.2f, want %.2f", b.Value, tt.wantDRValue)
					}
					if b.Duration < 5.9 || b.Duration > 6.1 {
						t.Errorf("DR buff duration = %.1f, want ~6.0", b.Duration)
					}
				}
			}

			if tt.wantCCBuff && buffTarget != nil {
				if !buffTarget.HasBuff("neural_fort_cc") {
					t.Error("expected neural_fort_cc buff on target")
				} else {
					b := buffTarget.GetBuff("neural_fort_cc")
					if b.Type != entity.BuffCCImmunity {
						t.Errorf("CC buff type = %q, want %q", b.Type, entity.BuffCCImmunity)
					}
					if math.Abs(float64(b.Value-tt.wantCCValue)) > 0.01 {
						t.Errorf("CC buff value = %.2f, want %.2f", b.Value, tt.wantCCValue)
					}
					if b.Duration < 5.9 || b.Duration > 6.1 {
						t.Errorf("CC buff duration = %.1f, want ~6.0", b.Duration)
					}
				}
			}

			// Check flux spending.
			if tt.wantFlux >= 0 {
				flux := caster.Resources[entity.ResourceFlux]
				if flux != nil && math.Abs(float64(flux.Current-tt.wantFlux)) > 0.5 {
					t.Errorf("Flux = %.1f, want %.1f", flux.Current, tt.wantFlux)
				}
			}

			if tt.wantCooldown {
				if cd, ok := caster.Cooldowns["neural_fortification"]; !ok || cd <= 0 {
					t.Error("expected cooldown to be set")
				} else if math.Abs(float64(cd-20.0)) > 0.1 {
					t.Errorf("cooldown = %.1f, want 20.0", cd)
				}
			}

			if tt.wantGCD {
				if caster.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				} else if math.Abs(float64(caster.GCDTimer-0.8)) > 0.01 {
					t.Errorf("GCD = %.2f, want 0.8", caster.GCDTimer)
				}
			}

			// Check confluence grant for successful commits.
			if tt.wantOK && tt.name == tcGrantsConfluence {
				if caster.Confluence == nil {
					t.Error("expected Confluence to be non-nil")
				} else if caster.Confluence.Stacks != stacksBefore+1 {
					t.Errorf("Confluence stacks = %d, want %d", caster.Confluence.Stacks, stacksBefore+1)
				}
			}
		})
	}
}
