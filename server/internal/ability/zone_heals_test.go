package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestZoneHeals(t *testing.T) {
	eng := NewEngine(nil)

	t.Run(IDVitalBloom, func(t *testing.T) {
		tests := []struct {
			name          string
			setup         func() (*entity.Player, []*entity.HealingZone)
			wantOK        bool
			wantReason    string
			wantHP        float32
			wantFlux      float32
			wantZoneCount int
			wantHealTick  float32
		}{
			{
				name: "subtracts HP and spawns zone",
				setup: func() (*entity.Player, []*entity.HealingZone) {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.Health = 100
					return p, nil
				},
				wantOK:        true,
				wantHP:        85,  // 100 - 15% of 100
				wantFlux:      152, // 160 - 8
				wantZoneCount: 1,
				wantHealTick:  4.5, // 15 * 0.3
			},
			{
				name: "sacrifice clamped when HP would drop below 1",
				setup: func() (*entity.Player, []*entity.HealingZone) {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.Health = 1.1 // 15% of 1.1 = 0.165, would leave 0.935 < 1, so clamped
					return p, nil
				},
				wantOK:        true,
				wantHP:        1, // clamped: sacrifice = 1.1 - 1 = 0.1
				wantFlux:      152,
				wantZoneCount: 1,
				wantHealTick:  0.03, // 0.1 * 0.3 = 0.03
			},
			{
				name: "fails when HP is 1",
				setup: func() (*entity.Player, []*entity.HealingZone) {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.Health = 1
					return p, nil
				},
				wantOK:     false,
				wantReason: "too low HP",
				wantHP:     1,
				wantFlux:   160, // unchanged
			},
			{
				name: tcInsufficientFluxBeforeHandler,
				setup: func() (*entity.Player, []*entity.HealingZone) {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.Health = 100
					p.SetAllFluxPoolsCurrent(3) // need 8 biometabolic
					return p, nil
				},
				wantOK:     false,
				wantReason: tcInsufficientBiometabolicFlux,
				wantHP:     100,
				wantFlux:   -1, // skip (pool-managed)
			},
			{
				name: "sets GCD",
				setup: func() (*entity.Player, []*entity.HealingZone) {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.Health = 100
					return p, nil
				},
				wantOK:        true,
				wantHP:        85,
				wantFlux:      152,
				wantZoneCount: 1,
				wantHealTick:  4.5,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p, _ := tt.setup()
				var spawned []*entity.HealingZone

				ctx := &CommitContext{
					Committer: p,
					SpawnZone: func(zone *entity.HealingZone) {
						spawned = append(spawned, zone)
					},
				}

				result := eng.Commit(IDVitalBloom, ctx)

				if result.OK != tt.wantOK {
					t.Fatalf("OK = %v, want %v (reason: %q)", result.OK, tt.wantOK, result.Reason)
				}
				if !tt.wantOK && result.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
				}
				if math.Abs(float64(p.Health-tt.wantHP)) > 0.1 {
					t.Errorf("Health = %.1f, want %.1f", p.Health, tt.wantHP)
				}
				flux := p.Resources[entity.ResourceFlux]
				if tt.wantFlux >= 0 && flux != nil && math.Abs(float64(flux.Current-tt.wantFlux)) > 0.5 {
					t.Errorf("Flux = %.1f, want %.1f", flux.Current, tt.wantFlux)
				}
				if tt.wantZoneCount > 0 {
					if len(spawned) != tt.wantZoneCount {
						t.Fatalf("spawned zones = %d, want %d", len(spawned), tt.wantZoneCount)
					}
					z := spawned[0]
					if z.OwnerID != p.ID {
						t.Errorf("zone OwnerID = %d, want %d", z.OwnerID, p.ID)
					}
					if z.AbilityID != IDVitalBloom {
						t.Errorf("zone AbilityID = %q, want %q", z.AbilityID, IDVitalBloom)
					}
					if math.Abs(float64(z.HealPerTick-tt.wantHealTick)) > 0.1 {
						t.Errorf("zone HealPerTick = %.1f, want %.1f", z.HealPerTick, tt.wantHealTick)
					}
					if z.Radius != vitalBloomDef.ZoneRadius {
						t.Errorf("zone Radius = %.1f, want %.1f", z.Radius, vitalBloomDef.ZoneRadius)
					}
					if z.Duration != vitalBloomDef.ZoneDuration {
						t.Errorf("zone Duration = %.1f, want %.1f", z.Duration, vitalBloomDef.ZoneDuration)
					}
				}
				if tt.wantOK && p.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				}
			})
		}
	})

	t.Run(IDRestorationMatrix, func(t *testing.T) {
		tests := []struct {
			name          string
			setup         func() *entity.Player
			wantOK        bool
			wantReason    string
			wantFlux      float32
			wantZoneCount int
			wantHealTick  float32
			wantCooldown  bool
		}{
			{
				name: "spends flux and spawns zone",
				setup: func() *entity.Player {
					return entity.NewPlayer(1, entity.ClassArcanotechnicien)
				},
				wantOK:        true,
				wantFlux:      110, // 160 - 50
				wantZoneCount: 1,
				wantHealTick:  8, // base 8, no identity
				wantCooldown:  true,
			},
			{
				name: "identity scales heal per tick",
				setup: func() *entity.Player {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.GearStats.Identity = 50
					p.RecalcStats()
					return p
				},
				wantOK:        true,
				wantFlux:      110, // initial flux stays 160 (RecalcStats changes Max, not Current); 160 - 50
				wantZoneCount: 1,
				wantHealTick:  12, // 8 * (1 + 50/100) = 12
				wantCooldown:  true,
			},
			{
				name: tcInsufficientFluxBeforeHandler,
				setup: func() *entity.Player {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.SetAllFluxPoolsCurrent(20) // need 50 bioarcanotechnic
					return p
				},
				wantOK:     false,
				wantReason: tcInsufficientBioarcanotechnicFlux,
				wantFlux:   -1, // skip (pool-managed)
			},
			{
				name: "rejected on GCD",
				setup: func() *entity.Player {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.GCDTimer = 0.5
					return p
				},
				wantOK:     false,
				wantReason: ReasonGCD,
				wantFlux:   160,
			},
			{
				name: "rejected on cooldown",
				setup: func() *entity.Player {
					p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
					p.Cooldowns[IDRestorationMatrix] = 5.0
					return p
				},
				wantOK:     false,
				wantReason: ReasonCooldown,
				wantFlux:   160,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := tt.setup()
				var spawned []*entity.HealingZone

				ctx := &CommitContext{
					Committer: p,
					SpawnZone: func(zone *entity.HealingZone) {
						spawned = append(spawned, zone)
					},
				}

				result := eng.Commit(IDRestorationMatrix, ctx)

				if result.OK != tt.wantOK {
					t.Fatalf("OK = %v, want %v (reason: %q)", result.OK, tt.wantOK, result.Reason)
				}
				if !tt.wantOK && result.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
				}
				flux := p.Resources[entity.ResourceFlux]
				if tt.wantFlux >= 0 && flux != nil && math.Abs(float64(flux.Current-tt.wantFlux)) > 0.5 {
					t.Errorf("Flux = %.1f, want %.1f", flux.Current, tt.wantFlux)
				}
				if tt.wantZoneCount > 0 {
					if len(spawned) != tt.wantZoneCount {
						t.Fatalf("spawned zones = %d, want %d", len(spawned), tt.wantZoneCount)
					}
					z := spawned[0]
					if z.OwnerID != p.ID {
						t.Errorf("zone OwnerID = %d, want %d", z.OwnerID, p.ID)
					}
					if z.AbilityID != IDRestorationMatrix {
						t.Errorf("zone AbilityID = %q, want %q", z.AbilityID, IDRestorationMatrix)
					}
					if math.Abs(float64(z.HealPerTick-tt.wantHealTick)) > 0.1 {
						t.Errorf("zone HealPerTick = %.1f, want %.1f", z.HealPerTick, tt.wantHealTick)
					}
					if z.Radius != restorationMatrixDef.ZoneRadius {
						t.Errorf("zone Radius = %.1f, want %.1f", z.Radius, restorationMatrixDef.ZoneRadius)
					}
					if z.Duration != restorationMatrixDef.ZoneDuration {
						t.Errorf("zone Duration = %.1f, want %.1f", z.Duration, restorationMatrixDef.ZoneDuration)
					}
				}
				if tt.wantCooldown {
					if cd, ok := p.Cooldowns[IDRestorationMatrix]; !ok || cd <= 0 {
						t.Error("expected cooldown to be set")
					}
				}
				if tt.wantOK && p.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				}
			})
		}
	})
}
