package item

import (
	"math"
	"testing"
)

func TestScalingFactor_BaselinePreserved(t *testing.T) {
	stats := []struct {
		stat StatID
		name string
	}{
		{StatHull, "Hull"},
		{StatOutput, "Output"},
		{StatPlating, "Plating"},
		{StatTempo, "Tempo"},
		{StatIdentity, "Identity"},
		{StatMastery, "Mastery"},
	}
	for _, tc := range stats {
		t.Run(tc.name, func(t *testing.T) {
			got := scalingFactor(tc.stat, 1)
			if got != 1.0 {
				t.Errorf("scalingFactor(%s, 1) = %f, want 1.0", tc.name, got)
			}
		})
	}
}

func TestScalingFactor_TargetRatiosAtIlvl50(t *testing.T) {
	tests := []struct {
		stat   StatID
		name   string
		target float64
	}{
		{StatHull, "Hull", 3.0},
		{StatOutput, "Output", 2.25},
		{StatPlating, "Plating", 1.75},
		{StatTempo, "Tempo", 1.75},
		{StatIdentity, "Identity", 1.75},
		{StatMastery, "Mastery", 1.75},
	}
	const tolerance = 0.05 // 5%
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := float64(scalingFactor(tc.stat, 50))
			ratio := math.Abs(got-tc.target) / tc.target
			if ratio > tolerance {
				t.Errorf("scalingFactor(%s, 50) = %.4f, want ~%.1f (off by %.1f%%)",
					tc.name, got, tc.target, ratio*100)
			}
		})
	}
}

func TestScalingFactor_Monotonicity(t *testing.T) {
	stats := []struct {
		stat StatID
		name string
	}{
		{StatHull, "Hull"},
		{StatOutput, "Output"},
		{StatPlating, "Plating"},
		{StatTempo, "Tempo"},
		{StatIdentity, "Identity"},
		{StatMastery, "Mastery"},
	}
	for _, tc := range stats {
		t.Run(tc.name, func(t *testing.T) {
			for n := 1; n <= 55; n++ {
				prev := scalingFactor(tc.stat, n)
				next := scalingFactor(tc.stat, n+1)
				if next <= prev {
					t.Errorf("scalingFactor(%s, %d)=%.4f >= scalingFactor(%s, %d)=%.4f — not monotonic",
						tc.name, n+1, next, tc.name, n, prev)
				}
			}
		})
	}
}

func TestScaleStatLine_FullKitTotalsAtIlvl50(t *testing.T) {
	// Starter gear base values (sum across all 6 items at ilvl 1):
	//   Hull:  30 + 5 + 10 = 45
	//   Output: 8 + 2 + 10 + 3 + 5 + 3 = 31
	//   Plating: 3 + 2 = 5
	// At ilvl 50, expected totals = base * scalingFactor(stat, 50).
	tests := []struct {
		name       string
		stat       StatID
		baseTotal  float32
		wantScaled float64
	}{
		{"Hull", StatHull, 45, 45 * 3.0},
		{"Output", StatOutput, 31, 31 * 2.25},
		{"Plating", StatPlating, 5, 5 * 1.75},
	}
	const tolerance = 0.05
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line := StatLine{Stat: tc.stat, Value: tc.baseTotal}
			got := float64(ScaleStatLine(line, 50))
			ratio := math.Abs(got-tc.wantScaled) / tc.wantScaled
			if ratio > tolerance {
				t.Errorf("ScaleStatLine(%s, base=%.1f, ilvl=50) = %.2f, want ~%.1f (off by %.1f%%)",
					tc.name, tc.baseTotal, got, tc.wantScaled, ratio*100)
			}
		})
	}
}

func TestScalingFactor_HeritageFloorIlvl15(t *testing.T) {
	// At ilvl 15, values should be meaningfully above baseline but well
	// below the ilvl-50 targets, confirming the curve is neither too flat
	// nor too steep at mid-range.
	// pow(14, 2) = 196
	tests := []struct {
		stat    StatID
		name    string
		wantMin float64
		wantMax float64
	}{
		// Hull: 1 + (2/2401)*196 = 1.163
		{StatHull, "Hull", 1.1, 1.5},
		// Output: 1 + (1.25/2401)*196 = 1.102
		{StatOutput, "Output", 1.05, 1.3},
		// Plating: 1 + (0.75/2401)*196 = 1.061
		{StatPlating, "Plating", 1.02, 1.2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := float64(scalingFactor(tc.stat, 15))
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("scalingFactor(%s, 15) = %.4f, want in [%.1f, %.1f]",
					tc.name, got, tc.wantMin, tc.wantMax)
			}
		})
	}
}
