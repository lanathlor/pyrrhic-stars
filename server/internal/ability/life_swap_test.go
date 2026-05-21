package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestLifeSwap(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		name            string
		setup           func() (*entity.Player, map[uint16]*entity.Player, uint16)
		wantOK          bool
		wantReason      string
		wantAllyHealth  float32
		wantVitalCharge float32
		wantHealCount   int
		wantHealAmount  float32
	}{
		{
			name: "drains ally HP and stores vital charge",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 100
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:          true,
			wantAllyHealth:  80,  // 100 - 100*0.20
			wantVitalCharge: 20,  // 100 * 0.20
			wantHealCount:   1,
			wantHealAmount:  -20, // negative = drain
		},
		{
			name: "fails on target at 1 HP",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 1
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: "target too low",
		},
		{
			name: "fails with no valid target",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 999
			},
			wantOK:     false,
			wantReason: "no valid target",
		},
		{
			name: "fails when targeting self",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				allies := map[uint16]*entity.Player{1: caster}
				return caster, allies, 1
			},
			wantOK:     false,
			wantReason: "no valid target",
		},
		{
			name: "clamps drain so ally never drops below 1 HP",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 4 // 20% of 4 = 0.8, but 4-0.8=3.2 > 1, so drain = 0.8
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:          true,
			wantAllyHealth:  3.2,
			wantVitalCharge: 0.8,
			wantHealCount:   1,
			wantHealAmount:  -0.8,
		},
		{
			name: "spends flux on cast",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 100
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:          true,
			wantAllyHealth:  80,
			wantVitalCharge: 20,
			wantHealCount:   1,
			wantHealAmount:  -20,
		},
		{
			name: "insufficient flux rejects before handler",
			setup: func() (*entity.Player, map[uint16]*entity.Player, uint16) {
				caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
				caster.Resources["flux"].Current = 2 // need 5
				ally := entity.NewPlayer(2, entity.ClassArcanotechnicien)
				ally.Health = 100
				allies := map[uint16]*entity.Player{1: caster, 2: ally}
				return caster, allies, 2
			},
			wantOK:     false,
			wantReason: "insufficient flux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caster, allies, targetPeer := tt.setup()

			result := eng.Cast("life_swap", &CastContext{
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
				return
			}

			if len(result.Heals) != tt.wantHealCount {
				t.Fatalf("Heals count = %d, want %d", len(result.Heals), tt.wantHealCount)
			}

			if tt.wantHealCount > 0 {
				h := result.Heals[0]
				if math.Abs(float64(h.Amount-tt.wantHealAmount)) > 0.01 {
					t.Errorf("Heal Amount = %.2f, want %.2f", h.Amount, tt.wantHealAmount)
				}
			}

			ally := allies[targetPeer]
			if ally != nil && tt.wantAllyHealth > 0 {
				if math.Abs(float64(ally.Health-tt.wantAllyHealth)) > 0.01 {
					t.Errorf("ally Health = %.2f, want %.2f", ally.Health, tt.wantAllyHealth)
				}
			}

			if tt.wantVitalCharge > 0 {
				if math.Abs(float64(caster.VitalCharge-tt.wantVitalCharge)) > 0.01 {
					t.Errorf("VitalCharge = %.2f, want %.2f", caster.VitalCharge, tt.wantVitalCharge)
				}
				if caster.VitalChargeTimer <= 0 {
					t.Error("expected VitalChargeTimer > 0")
				}
			}

			// Verify flux was spent
			if tt.wantOK && tt.name == "spends flux on cast" {
				flux := caster.Resources["flux"]
				want := float32(155) // 160 - 5
				if math.Abs(float64(flux.Current-want)) > 0.5 {
					t.Errorf("Flux = %.1f, want %.1f", flux.Current, want)
				}
			}

			// Verify GCD was set
			if tt.wantOK && caster.GCDTimer <= 0 {
				t.Error("expected GCD to be set after successful cast")
			}
		})
	}
}

func TestVitalChargeEmpowersMendingSurge(t *testing.T) {
	eng := NewEngine(nil)

	// Setup: caster + ally, both arcanotechnicien
	caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	donor := entity.NewPlayer(2, entity.ClassArcanotechnicien)
	donor.Health = 100
	patient := entity.NewPlayer(3, entity.ClassArcanotechnicien)
	patient.Health = 10 // very low, lots of room to heal

	allies := map[uint16]*entity.Player{1: caster, 2: donor, 3: patient}

	// Step 1: Cast Life Swap on donor to build vital charge
	result := eng.Cast("life_swap", &CastContext{
		Caster:       caster,
		Allies:       allies,
		TargetPeerID: 2,
	})
	if !result.OK {
		t.Fatalf("Life Swap failed: %s", result.Reason)
	}

	charge := caster.VitalCharge
	if charge <= 0 {
		t.Fatalf("expected VitalCharge > 0, got %.2f", charge)
	}

	// Step 2: Clear GCD so we can cast Mending Surge
	caster.GCDTimer = 0
	// Also clear the mending_surge cooldown if set
	delete(caster.Cooldowns, "mending_surge")

	// Step 3: Cast Mending Surge on the patient
	healResult := eng.Cast("mending_surge", &CastContext{
		Caster:       caster,
		Allies:       allies,
		TargetPeerID: 3,
	})
	if !healResult.OK {
		t.Fatalf("Mending Surge failed: %s", healResult.Reason)
	}
	if len(healResult.Heals) == 0 {
		t.Fatal("expected at least one heal result")
	}

	// Base heal = 80, confluence gives +8% (1 stack from life swap), so 80*1.08=86.4
	// Plus vital charge (20 from 100*0.20)
	// Total = 86.4 + 20 = 106.4
	baseHeal := float32(80)
	confluenceMult := float32(1.08) // 1 stack from Life Swap
	expectedHeal := baseHeal*confluenceMult + charge

	actualHeal := healResult.Heals[0].Amount
	if math.Abs(float64(actualHeal-expectedHeal)) > 0.5 {
		t.Errorf("empowered heal = %.1f, want %.1f (base=%.0f, confluence=%.2f, charge=%.1f)",
			actualHeal, expectedHeal, baseHeal, confluenceMult, charge)
	}

	// Step 4: Verify vital charge was consumed
	if caster.VitalCharge != 0 {
		t.Errorf("VitalCharge = %.2f after consumption, want 0", caster.VitalCharge)
	}
	if caster.VitalChargeTimer != 0 {
		t.Errorf("VitalChargeTimer = %.2f after consumption, want 0", caster.VitalChargeTimer)
	}
}

func TestVitalChargeExpiry(t *testing.T) {
	eng := NewEngine(nil)

	caster := entity.NewPlayer(1, entity.ClassArcanotechnicien)
	caster.VitalCharge = 30
	caster.VitalChargeTimer = 4.0

	// Tick 3.5 seconds: should still have charge
	eng.TickPlayer(caster, 3.5, nil)
	if caster.VitalCharge != 30 {
		t.Errorf("VitalCharge after 3.5s = %.2f, want 30", caster.VitalCharge)
	}
	if caster.VitalChargeTimer <= 0 {
		t.Error("VitalChargeTimer should still be > 0 after 3.5s")
	}

	// Tick 0.6 more seconds (total 4.1s): charge should be expired
	eng.TickPlayer(caster, 0.6, nil)
	if caster.VitalCharge != 0 {
		t.Errorf("VitalCharge after 4.1s = %.2f, want 0", caster.VitalCharge)
	}
	if caster.VitalChargeTimer != 0 {
		t.Errorf("VitalChargeTimer after 4.1s = %.2f, want 0", caster.VitalChargeTimer)
	}
}
