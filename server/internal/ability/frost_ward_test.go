package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

// newFrostCaster creates a harmonist with a frost flux pool for testing Frost Ward.
// The default harmonist commitment only has bioarcanotechnic/biometabolic,
// so we add a frost pool explicitly.
func newFrostCaster(id uint16) *entity.Player {
	p := newHarmonist(id)
	p.FluxCommit.SetCommitment(map[string]float32{
		entity.SchoolBioarcanotechnic: 0.4,
		entity.SchoolBiometabolic:     0.3,
		entity.SchoolFrost:            0.3,
	})
	p.SyncFluxAggregate()
	return p
}

func TestFrostWardHandler(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name         string
		setup        func() (*entity.Player, map[uint16]*entity.Player, uint16)
		wantOK       bool
		wantReason   string
		wantShield   float32
		wantBuff     bool
		wantActive   bool
		wantFlux     float32 // -1 to skip check
		wantCooldown bool
		wantGCD      bool
	}{
		{
			name: "applies shield and buff to target ally",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				ally := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:       true,
			wantShield:   30,
			wantBuff:     true,
			wantActive:   true,
			wantFlux:     -1, // pool-based, checked separately
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "falls back to self when target invalid",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 999
			},
			wantOK:       true,
			wantShield:   30,
			wantBuff:     true,
			wantActive:   true,
			wantFlux:     -1,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "caps shield at 50",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				ally := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
				ally.Resources["shield"] = &entity.Resource{Current: 35, Max: 50}
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:       true,
			wantShield:   50, // 35 + 30 = 65, capped at 50
			wantBuff:     true,
			wantActive:   true,
			wantFlux:     -1,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "creates shield resource if absent",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				ally := entity.NewPlayerWithSpec(2, entity.ClassArcanotechnicien, "harmonist")
				delete(ally.Resources, "shield") // ensure absent
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:       true,
			wantShield:   30,
			wantBuff:     true,
			wantActive:   true,
			wantFlux:     -1,
			wantCooldown: true,
			wantGCD:      true,
		},
		{
			name: "spends frost flux",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 1
			},
			wantOK:     true,
			wantShield: 30,
			wantBuff:   true,
			wantActive: true,
			wantFlux:   28, // 48 - 20 = 28 remaining in frost pool
		},
		{
			name: tcGrantsConfluence,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 1
			},
			wantOK:     true,
			wantShield: 30,
			wantBuff:   true,
			wantActive: true,
			wantFlux:   -1,
		},
		{
			name: tcRejectsInsufficientFlux,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newHarmonist(1)
				caster.SetAllFluxPoolsCurrent(5)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 1
			},
			wantOK:     false,
			wantReason: "insufficient " + entity.SchoolFrost + " flux",
			wantFlux:   -1,
		},
		{
			name: tcRejectsGCD,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				caster.GCDTimer = 0.5
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 1
			},
			wantOK:     false,
			wantReason: ReasonGCD,
			wantFlux:   -1,
		},
		{
			name: tcRejectsCooldown,
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := newFrostCaster(1)
				caster.Cooldowns[IDFrostWard] = 5.0
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 1
			},
			wantOK:     false,
			wantReason: ReasonCooldown,
			wantFlux:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caster, allies, targetPeer := tt.setup()

			var stacksBefore int
			if caster.Confluence != nil {
				stacksBefore = caster.Confluence.Stacks
			}

			result := eng.Commit(IDFrostWard, &CommitContext{
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

			// Determine buff target: ally if valid, otherwise self.
			var buffTarget *entity.Player
			if a, ok := allies[targetPeer]; ok && a.ID != caster.ID {
				buffTarget = a
			} else {
				buffTarget = caster
			}

			// Check shield.
			shield := buffTarget.Resources["shield"]
			if shield == nil {
				t.Fatal("expected shield resource on target")
			}
			if math.Abs(float64(shield.Current-tt.wantShield)) > 0.1 {
				t.Errorf("shield = %.1f, want %.1f", shield.Current, tt.wantShield)
			}

			// Check buff.
			if tt.wantBuff {
				b := buffTarget.GetBuff(IDFrostWard)
				if b == nil {
					t.Error("expected frost_ward buff on target")
				} else {
					if b.Type != entity.BuffDamageReduction {
						t.Errorf("buff type = %q, want %q", b.Type, entity.BuffDamageReduction)
					}
					if math.Abs(float64(b.Value-1.0)) > 0.01 {
						t.Errorf("buff value = %.2f, want 1.0", b.Value)
					}
					if b.Duration < 5.9 || b.Duration > 6.1 {
						t.Errorf("buff duration = %.1f, want ~6.0", b.Duration)
					}
				}
			}

			// Check ability state.
			if tt.wantActive {
				active, ok := buffTarget.AbilityState["frost_ward_active"].(bool)
				if !ok || !active {
					t.Error("expected frost_ward_active = true on target")
				}
			}

			// Check frost pool flux spending.
			if tt.wantFlux >= 0 {
				pool := caster.FluxCommit.GetPool("frost")
				if pool == nil {
					t.Fatal("expected frost flux pool to exist")
				}
				if math.Abs(float64(pool.Current-tt.wantFlux)) > 0.5 {
					t.Errorf("frost flux = %.1f, want %.1f", pool.Current, tt.wantFlux)
				}
			}

			// Check cooldown.
			if tt.wantCooldown {
				cd, ok := caster.Cooldowns[IDFrostWard]
				if !ok || cd <= 0 {
					t.Error("expected cooldown to be set")
				} else if math.Abs(float64(cd-12.0)) > 0.1 {
					t.Errorf("cooldown = %.1f, want 12.0", cd)
				}
			}

			// Check GCD.
			if tt.wantGCD {
				if caster.GCDTimer <= 0 {
					t.Error("expected GCD to be set")
				} else if math.Abs(float64(caster.GCDTimer-0.8)) > 0.01 {
					t.Errorf("GCD = %.2f, want 0.8", caster.GCDTimer)
				}
			}

			// Check confluence for the dedicated test case.
			if tt.name == tcGrantsConfluence {
				if caster.Confluence == nil {
					t.Error("expected Confluence to be non-nil")
				} else if caster.Confluence.Stacks != stacksBefore+1 {
					t.Errorf("Confluence stacks = %d, want %d", caster.Confluence.Stacks, stacksBefore+1)
				}
			}
		})
	}
}

func TestFrostWardTick(t *testing.T) {
	eng := NewEngine(nil)

	t.Run("no explosion when ward is not active", func(t *testing.T) {
		p := newHarmonist(1)
		results := frostWardTick(eng, p, 0.05, &TickContext{})
		if len(results) > 0 {
			t.Errorf("expected no results, got %d", len(results))
		}
	})

	t.Run("no explosion while buff has time remaining", func(t *testing.T) {
		p := newHarmonist(1)
		p.AbilityState["frost_ward_active"] = true
		p.AddBuff(entity.ActiveBuff{
			ID:       IDFrostWard,
			Type:     entity.BuffDamageReduction,
			Value:    1.0,
			Duration: 5.0,
		})

		results := frostWardTick(eng, p, 0.05, &TickContext{})
		if len(results) > 0 {
			t.Errorf("expected no results mid-duration, got %d", len(results))
		}
	})

	t.Run("explodes when buff expires this tick", func(t *testing.T) {
		p := newHarmonist(1)
		p.AbilityState["frost_ward_active"] = true
		p.Resources["shield"] = &entity.Resource{Current: 15, Max: 50}
		p.AddBuff(entity.ActiveBuff{
			ID:       IDFrostWard,
			Type:     entity.BuffDamageReduction,
			Value:    1.0,
			Duration: 0.04, // less than dt, will expire this tick
		})

		enemy := entity.NewEnemy(100, 200, "test_mob")
		enemy.Position = entity.Vec3{X: 3, Y: 0, Z: 0} // within 5m

		results := frostWardTick(eng, p, 0.05, &TickContext{
			Targets: []entity.Target{enemy},
		})

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].TargetID != 100 {
			t.Errorf("target ID = %d, want 100", results[0].TargetID)
		}
		if results[0].SourceID != p.ID {
			t.Errorf("source ID = %d, want %d", results[0].SourceID, p.ID)
		}
		if results[0].Amount <= 0 {
			t.Errorf("damage = %.1f, want > 0", results[0].Amount)
		}
		if results[0].AbilityID != IDFrostWard {
			t.Errorf("ability ID = %q, want %q", results[0].AbilityID, IDFrostWard)
		}

		// Check frostbite debuff on enemy.
		if !enemy.HasDebuff(entity.DebuffSlow) {
			t.Error("expected frostbite (slow) debuff on enemy")
		}

		// Ward active should be cleared.
		if active, ok := p.AbilityState["frost_ward_active"].(bool); ok && active {
			t.Error("expected frost_ward_active = false after explosion")
		}

		// Shield should be cleared.
		if shield := p.Resources["shield"]; shield != nil && shield.Current > 0 {
			t.Errorf("shield = %.1f, want 0 after explosion", shield.Current)
		}
	})

	t.Run("explodes when buff removed externally", func(t *testing.T) {
		p := newHarmonist(1)
		p.AbilityState["frost_ward_active"] = true
		p.Resources["shield"] = &entity.Resource{Current: 10, Max: 50}
		// No buff on the player -- simulates external removal.

		enemy := entity.NewEnemy(100, 200, "test_mob")
		enemy.Position = entity.Vec3{X: 2, Y: 0, Z: 0}

		results := frostWardTick(eng, p, 0.05, &TickContext{
			Targets: []entity.Target{enemy},
		})

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		if active, ok := p.AbilityState["frost_ward_active"].(bool); ok && active {
			t.Error("expected frost_ward_active = false after explosion")
		}
	})

	t.Run("does not hit enemies beyond 5m", func(t *testing.T) {
		p := newHarmonist(1)
		p.AbilityState["frost_ward_active"] = true
		p.AddBuff(entity.ActiveBuff{
			ID:       IDFrostWard,
			Type:     entity.BuffDamageReduction,
			Value:    1.0,
			Duration: 0.03,
		})

		farEnemy := entity.NewEnemy(100, 200, "test_mob")
		farEnemy.Position = entity.Vec3{X: 6, Y: 0, Z: 0} // beyond 5m

		results := frostWardTick(eng, p, 0.05, &TickContext{
			Targets: []entity.Target{farEnemy},
		})

		if len(results) != 0 {
			t.Errorf("expected 0 results for out-of-range enemy, got %d", len(results))
		}

		// Active should still be cleared despite no hits.
		if active, ok := p.AbilityState["frost_ward_active"].(bool); ok && active {
			t.Error("expected frost_ward_active = false after explosion (no hits)")
		}
	})

	t.Run("hits multiple enemies in range", func(t *testing.T) {
		p := newHarmonist(1)
		p.AbilityState["frost_ward_active"] = true
		p.AddBuff(entity.ActiveBuff{
			ID:       IDFrostWard,
			Type:     entity.BuffDamageReduction,
			Value:    1.0,
			Duration: 0.03,
		})

		e1 := entity.NewEnemy(101, 200, "mob1")
		e1.Position = entity.Vec3{X: 2, Y: 0, Z: 0}
		e2 := entity.NewEnemy(102, 200, "mob2")
		e2.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		e3 := entity.NewEnemy(103, 200, "mob3")
		e3.Position = entity.Vec3{X: 10, Y: 0, Z: 0} // out of range

		results := frostWardTick(eng, p, 0.05, &TickContext{
			Targets: []entity.Target{e1, e2, e3},
		})

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		// Both in-range enemies should have frostbite.
		if !e1.HasDebuff(entity.DebuffSlow) {
			t.Error("expected frostbite on enemy 101")
		}
		if !e2.HasDebuff(entity.DebuffSlow) {
			t.Error("expected frostbite on enemy 102")
		}
		if e3.HasDebuff(entity.DebuffSlow) {
			t.Error("enemy 103 should NOT have frostbite (out of range)")
		}
	})

	t.Run("no damage to dead enemies", func(t *testing.T) {
		p := newHarmonist(1)
		p.AbilityState["frost_ward_active"] = true
		// Buff already gone (external removal).

		dead := entity.NewEnemy(100, 200, "test_mob")
		dead.Position = entity.Vec3{X: 1, Y: 0, Z: 0}
		dead.Alive = false

		results := frostWardTick(eng, p, 0.05, &TickContext{
			Targets: []entity.Target{dead},
		})

		if len(results) != 0 {
			t.Errorf("expected 0 results for dead enemy, got %d", len(results))
		}
	})
}
