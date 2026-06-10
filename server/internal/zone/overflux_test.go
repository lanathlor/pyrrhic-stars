package zone

import (
	"testing"

	"codex-online/server/internal/overflux"
)

// TestNew_NilOverflux verifies that a zone created with nil overflux has no
// overflux state accessible via OverfluxState().
func TestNew_NilOverflux(t *testing.T) {
	z := New("test_nil_oflx", testArenaLevel(t), nil)
	if got := z.OverfluxState(); got != nil {
		t.Errorf("OverfluxState() = %v, want nil", got)
	}
}

// TestNew_WithOverflux verifies that a zone created with an overflux state
// stores and returns it unchanged via OverfluxState().
func TestNew_WithOverflux(t *testing.T) {
	state := overflux.NewState([]overflux.ActiveCondition{
		{ID: overflux.CondEnemyHP, Rank: 3},
	})
	z := New("test_with_oflx", testArenaLevel(t), state)
	got := z.OverfluxState()
	if got == nil {
		t.Fatal("OverfluxState() = nil, want non-nil")
	}
	if got != state {
		t.Error("OverfluxState() returned different pointer than the one passed to New")
	}
}

// TestRescaleEnemies_NoOverflux verifies baseline group scaling without overflux.
// HP formula: 1.0 + 0.75*(n-1), so solo=1.0x, 3 players=2.5x.
func TestRescaleEnemies_NoOverflux(t *testing.T) {
	tests := []struct {
		name       string
		groupSize  int
		wantHPMult float32
	}{
		{"solo", 1, 1.0},
		{"3_players", 3, 2.5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z := New("test_rescale_no_oflx", testArenaLevel(t), nil)

			// Capture baseline HP values before any scaling call so we can
			// compute expected values against the original base.
			type baseline struct {
				baseHP float32
			}
			bases := make([]baseline, len(z.world.Enemies))
			for i, e := range z.world.Enemies {
				bases[i] = baseline{baseHP: e.BaseMaxHealth}
			}

			z.SetGroupSize(tc.groupSize)

			for i, e := range z.world.Enemies {
				want := bases[i].baseHP * tc.wantHPMult
				if !approxEqual(e.MaxHealth, want, 1e-3) {
					t.Errorf("enemy[%d] MaxHealth = %.3f, want %.3f (base=%.3f mult=%.3f)",
						i, e.MaxHealth, want, bases[i].baseHP, tc.wantHPMult)
				}
			}
		})
	}
}

// TestRescaleEnemies_WithOverflux verifies that the overflux HP multiplier
// compounds with group scaling. Combined formula:
//
//	hpMult = (1.0 + 0.75*(n-1)) * HPMultiplier()
//
// where HPMultiplier() = 1.0 + 0.2*rank for CondEnemyHP.
func TestRescaleEnemies_WithOverflux(t *testing.T) {
	tests := []struct {
		name       string
		groupSize  int
		rank       int
		wantHPMult float32
	}{
		// solo (1.0x group) * rank1 enemy_hp (1.2x) = 1.2x
		{"solo_rank1", 1, 1, 1.2},
		// solo (1.0x group) * rank5 enemy_hp (2.0x) = 2.0x
		{"solo_rank5", 1, 5, 2.0},
		// 3 players (2.5x group) * rank3 enemy_hp (1.6x) = 4.0x
		{"3p_rank3", 3, 3, 4.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := overflux.NewState([]overflux.ActiveCondition{
				{ID: overflux.CondEnemyHP, Rank: tc.rank},
			})
			z := New("test_rescale_oflx", testArenaLevel(t), state)

			type baseline struct {
				baseHP float32
			}
			bases := make([]baseline, len(z.world.Enemies))
			for i, e := range z.world.Enemies {
				bases[i] = baseline{baseHP: e.BaseMaxHealth}
			}

			z.SetGroupSize(tc.groupSize)

			for i, e := range z.world.Enemies {
				want := bases[i].baseHP * tc.wantHPMult
				if !approxEqual(e.MaxHealth, want, 1e-3) {
					t.Errorf("enemy[%d] MaxHealth = %.3f, want %.3f (base=%.3f mult=%.3f rank=%d)",
						i, e.MaxHealth, want, bases[i].baseHP, tc.wantHPMult, tc.rank)
				}
			}
		})
	}
}

// TestRescaleEnemies_PreservesRatio verifies that mid-fight rescaling preserves
// each enemy's current HP percentage, including when overflux compounds the
// HP multiplier.
func TestRescaleEnemies_PreservesRatio(t *testing.T) {
	tests := []struct {
		name      string
		groupSize int
		rank      int
		hpRatio   float32 // current HP as fraction of MaxHealth before rescale
	}{
		{"solo_no_oflx_50pct", 1, 0, 0.5},
		{"solo_rank2_50pct", 1, 2, 0.5},
		{"3p_rank3_75pct", 3, 3, 0.75},
		{"3p_no_oflx_25pct", 3, 0, 0.25},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var state *overflux.State
			if tc.rank > 0 {
				state = overflux.NewState([]overflux.ActiveCondition{
					{ID: overflux.CondEnemyHP, Rank: tc.rank},
				})
			}
			z := New("test_ratio", testArenaLevel(t), state)

			// Apply an initial scaling so enemies have a known MaxHealth.
			z.SetGroupSize(1)

			// Wound each enemy to tc.hpRatio of its current MaxHealth.
			for _, e := range z.world.Enemies {
				e.Health = e.MaxHealth * tc.hpRatio
			}

			// Now rescale for the target group size mid-fight.
			z.RescaleForPlayerCount(tc.groupSize)

			for i, e := range z.world.Enemies {
				if e.MaxHealth == 0 {
					continue
				}
				gotRatio := e.Health / e.MaxHealth
				if !approxEqual(gotRatio, tc.hpRatio, 1e-3) {
					t.Errorf("enemy[%d] HP ratio after rescale = %.4f, want %.4f",
						i, gotRatio, tc.hpRatio)
				}
			}
		})
	}
}

// approxEqual returns true if a and b differ by less than tol.
func approxEqual(a, b, tol float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tol
}
