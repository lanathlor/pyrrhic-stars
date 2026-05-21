package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestMendingSurge(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name           string
		setup          func() (*entity.Player, map[uint16]*entity.Player, uint16)
		wantOK         bool
		wantReason     string
		wantHealCount  int
		wantHealTarget uint16
		wantHealAmount float32
		wantFlux       float32
		wantCooldown   bool
		wantGCD        bool
	}{
		{
			name: "heals injured ally",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 50
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantHealCount:  1,
			wantHealTarget: 2,
			wantHealAmount: 92, // 80 * 1.15 (Sympathetic Field: Harmonist, same pos)
			wantFlux:       120, // 160 - 40
			wantCooldown:   true,
			wantGCD:        true,
		},
		{
			name: "falls back to self-heal when target invalid",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				caster.Health = 50
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 999
			},
			wantOK:         true,
			wantHealCount:  1,
			wantHealTarget: 1,
			wantHealAmount: 92, // 80 * 1.15 (Sympathetic Field: self-heal, dist=0)
			wantFlux:       120,
			wantCooldown:   true,
			wantGCD:        true,
		},
		{
			name: "insufficient flux rejects before handler",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				caster.Resources["flux"].Current = 10 // not enough
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 50
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: "insufficient flux",
			wantFlux:   10, // unchanged
		},
		{
			name: "everyone at full HP still spends flux",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				// both at full HP by default
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:        true,
			wantHealCount: 0, // no heal emitted, everyone full
			wantFlux:      120,
			wantCooldown:  true,
			wantGCD:       true,
		},
		{
			name: "identity stat scales heal",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				caster.GearStats.Identity = 50 // +50% heal, also scales SF radius
				caster.RecalcStats()
				// Current stays at initial (160), RecalcStats only changes Max.
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 20
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:         true,
			wantHealCount:  1,
			wantHealTarget: 2,
			wantHealAmount: 138, // 80 * 1.5 (identity) * 1.15 (Sympathetic Field) = 138
			wantFlux:       120, // 160 (initial current) - 40 (cost) = 120
			wantCooldown:   true,
			wantGCD:        true,
		},
		{
			name: "rejected on GCD",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				caster.GCDTimer = 0.5 // active GCD
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 50
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: "gcd",
			wantFlux:   160, // unchanged
		},
		{
			name: "rejected on cooldown",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				caster.Cooldowns["mending_surge"] = 1.0 // on cooldown
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 50
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: "cooldown",
			wantFlux:   160, // unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caster, allies, targetPeer := tt.setup()

			result := eng.Cast("mending_surge", &CastContext{
				Caster:       caster,
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

			if tt.wantHealCount >= 0 {
				if len(result.Heals) != tt.wantHealCount {
					t.Errorf("Heals count = %d, want %d", len(result.Heals), tt.wantHealCount)
				}
			}

			if tt.wantHealCount > 0 && len(result.Heals) > 0 {
				h := result.Heals[0]
				if h.TargetID != tt.wantHealTarget {
					t.Errorf("Heal TargetID = %d, want %d", h.TargetID, tt.wantHealTarget)
				}
				if math.Abs(float64(h.Amount-tt.wantHealAmount)) > 0.5 {
					t.Errorf("Heal Amount = %.1f, want %.1f", h.Amount, tt.wantHealAmount)
				}
				if h.SourceID != caster.ID {
					t.Errorf("Heal SourceID = %d, want %d", h.SourceID, caster.ID)
				}
			}

			flux := caster.Resources["flux"]
			if flux != nil && math.Abs(float64(flux.Current-tt.wantFlux)) > 0.5 {
				t.Errorf("Flux = %.1f, want %.1f", flux.Current, tt.wantFlux)
			}

			if tt.wantCooldown {
				if cd, ok := caster.Cooldowns["mending_surge"]; !ok || cd <= 0 {
					t.Error("expected cooldown to be set")
				}
			}
			if tt.wantGCD {
				if caster.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				}
			}
		})
	}
}
