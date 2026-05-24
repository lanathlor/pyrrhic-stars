package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestMetabolicBurst(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name             string
		setup            func() (*entity.Player, []entity.Target, map[uint16]*entity.Player)
		wantOK           bool
		wantReason       string
		wantDamageCount  int
		wantHealCount    int
		wantHealTargetID uint16
		wantHealFraction float32 // expected heal as fraction of damage dealt
		wantFlux         float32 // expected flux after commit (-1 to skip)
		wantCooldown     bool
		wantGCD          bool
		wantConfluence   int
	}{
		{
			name: "deals damage to nearest enemy",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1) // seed threat so resolveNearestN picks it up
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster}
				return caster, targets, allies
			},
			wantOK:          true,
			wantDamageCount: 1,
			wantHealCount:   1, // caster is within 8m of enemy, gets overheal
			wantFlux:        -1,
			wantCooldown:    true,
			wantGCD:         true,
			wantConfluence:  1,
		},
		{
			name: "heals allies near enemy",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				ally := newHarmonist(2)
				ally.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // 2m from enemy at (0,0,-5)
				ally.Health = 50
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1)
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, targets, allies
			},
			wantOK:           true,
			wantDamageCount:  1,
			wantHealCount:    2, // ally near enemy + caster (everyone within 8m of enemy)
			wantHealTargetID: 2, // ally should be among healed targets
			wantHealFraction: 0.5,
			wantFlux:         -1,
			wantCooldown:     true,
			wantGCD:          true,
			wantConfluence:   1,
		},
		{
			name: "does not heal allies far from enemy",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				caster.Position = entity.Vec3{X: 0, Y: 0, Z: 15} // far from enemy
				ally := newHarmonist(2)
				ally.Position = entity.Vec3{X: 0, Y: 0, Z: 20} // far from enemy at (0,0,-5)
				ally.Health = 50
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1)
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, targets, allies
			},
			wantOK:          true,
			wantDamageCount: 1,
			wantHealCount:   0, // no allies within 8m of enemy
			wantFlux:        -1,
			wantCooldown:    true,
			wantGCD:         true,
			wantConfluence:  1,
		},
		{
			name: "spends biometabolic flux (40)",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1)
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster}
				return caster, targets, allies
			},
			wantOK:          true,
			wantDamageCount: 1,
			wantHealCount:   1,   // caster within 8m of enemy
			wantFlux:        120, // 160 initial - 40 cost
			wantCooldown:    true,
			wantGCD:         true,
			wantConfluence:  1,
		},
		{
			name: "sets GCD (0.8) and cooldown (12.0)",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1)
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster}
				return caster, targets, allies
			},
			wantOK:          true,
			wantDamageCount: 1,
			wantHealCount:   1, // caster within 8m of enemy
			wantFlux:        -1,
			wantCooldown:    true,
			wantGCD:         true,
			wantConfluence:  1,
		},
		{
			name: "grants confluence stack",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1)
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster}
				return caster, targets, allies
			},
			wantOK:          true,
			wantDamageCount: 1,
			wantHealCount:   1, // caster within 8m of enemy
			wantFlux:        -1,
			wantCooldown:    true,
			wantGCD:         true,
			wantConfluence:  1,
		},
		{
			name: "rejects with no enemy target",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				targets := []entity.Target{} // no enemies
				allies := map[uint16]*entity.Player{1: caster}
				return caster, targets, allies
			},
			wantOK:     false,
			wantReason: "no target",
			wantFlux:   160, // unchanged
		},
		{
			name: "rejects on insufficient flux",
			setup: func() (*entity.Player, []entity.Target, map[uint16]*entity.Player) {
				caster := newHarmonist(1)
				caster.SetAllFluxPoolsCurrent(5) // need 40
				enemy := enemyInFront(100, 200)
				enemy.AddThreat(1, 1)
				targets := []entity.Target{enemy}
				allies := map[uint16]*entity.Player{1: caster}
				return caster, targets, allies
			},
			wantOK:     false,
			wantReason: "insufficient biometabolic flux",
			wantFlux:   -1, // pool-managed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caster, targets, allies := tt.setup()

			result := eng.Commit("metabolic_burst", &CommitContext{
				Committer: caster,
				Targets:   targets,
				Allies:    allies,
			})

			if result.OK != tt.wantOK {
				t.Fatalf("OK = %v, want %v (reason: %q)", result.OK, tt.wantOK, result.Reason)
			}
			if !tt.wantOK {
				if result.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
				}
			}

			if tt.wantDamageCount > 0 {
				if len(result.Events) != tt.wantDamageCount {
					t.Errorf("Events count = %d, want %d", len(result.Events), tt.wantDamageCount)
				}
				if len(result.Events) > 0 && result.Events[0].Amount <= 0 {
					t.Error("expected positive damage amount")
				}
			}

			if tt.wantHealCount > 0 {
				if len(result.Heals) < tt.wantHealCount {
					t.Errorf("Heals count = %d, want >= %d", len(result.Heals), tt.wantHealCount)
				}
				// Check that the expected ally target is among the healed
				if tt.wantHealTargetID > 0 {
					found := false
					for _, h := range result.Heals {
						if h.TargetID == tt.wantHealTargetID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected heal for target %d, not found in heals", tt.wantHealTargetID)
					}
				}
				// Check heal fraction relative to damage
				if tt.wantHealFraction > 0 && len(result.Events) > 0 && len(result.Heals) > 0 {
					totalDmg := float32(0)
					for _, ev := range result.Events {
						totalDmg += ev.Amount
					}
					expectedHeal := totalDmg * tt.wantHealFraction
					// Each healed ally gets the same amount; check any one
					for _, h := range result.Heals {
						actualHeal := h.Amount + h.Overheal
						if math.Abs(float64(actualHeal-expectedHeal)) > 1.0 {
							t.Errorf("Heal for target %d: Amount+Overheal = %.1f, want ~%.1f (50%% of %.1f damage)",
								h.TargetID, actualHeal, expectedHeal, totalDmg)
						}
					}
				}
			} else if tt.wantOK && tt.wantHealCount == 0 {
				if len(result.Heals) != 0 {
					t.Errorf("Heals count = %d, want 0", len(result.Heals))
				}
			}

			// Flux check
			if tt.wantFlux >= 0 {
				flux := caster.Resources[entity.ResourceFlux]
				if flux != nil && math.Abs(float64(flux.Current-tt.wantFlux)) > 0.5 {
					t.Errorf("Flux = %.1f, want %.1f", flux.Current, tt.wantFlux)
				}
			}

			// Cooldown check
			if tt.wantCooldown {
				if cd, ok := caster.Cooldowns["metabolic_burst"]; !ok || cd <= 0 {
					t.Error("expected cooldown to be set")
				} else if math.Abs(float64(cd-12.0)) > 0.01 {
					t.Errorf("Cooldown = %.1f, want 12.0", cd)
				}
			}

			// GCD check
			if tt.wantGCD {
				if caster.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				}
				if math.Abs(float64(caster.GCDTimer-0.8)) > 0.01 {
					t.Errorf("GCDTimer = %.3f, want 0.8", caster.GCDTimer)
				}
			}

			// Confluence check
			if tt.wantConfluence > 0 {
				if caster.Confluence == nil {
					t.Fatal("expected Confluence to be non-nil")
				}
				if caster.Confluence.Stacks != tt.wantConfluence {
					t.Errorf("Confluence.Stacks = %d, want %d", caster.Confluence.Stacks, tt.wantConfluence)
				}
			}
		})
	}
}

func TestMetabolicBurstRegistered(t *testing.T) {
	eng := NewEngine(nil)
	def := eng.GetAbility("metabolic_burst")
	if def == nil {
		t.Fatal("metabolic_burst not registered in engine")
	}
	if def.School != "biometabolic" {
		t.Errorf("School = %q, want %q", def.School, "biometabolic")
	}
	if def.BaseDamage != 25 {
		t.Errorf("BaseDamage = %v, want 25", def.BaseDamage)
	}
	if def.Cooldown != 12.0 {
		t.Errorf("Cooldown = %v, want 12.0", def.Cooldown)
	}
	if def.GCD != 0.8 {
		t.Errorf("GCD = %v, want 0.8", def.GCD)
	}
	if len(def.Costs) != 1 || def.Costs[0].Resource != entity.ResourceFlux || def.Costs[0].Amount != 40 {
		t.Errorf("Costs = %+v, want [{flux 40}]", def.Costs)
	}
	if def.Handler != "metabolic_burst" {
		t.Errorf("Handler = %q, want %q", def.Handler, "metabolic_burst")
	}
}

func TestMetabolicBurstInAbilitiesList(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	found := false
	for _, a := range p.AllowedAbilities() {
		if a == "metabolic_burst" {
			found = true
			break
		}
	}
	if !found {
		t.Error("metabolic_burst not in AllowedAbilities for Harmonist")
	}
}
