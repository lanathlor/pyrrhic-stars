package item

import (
	"math"
	"testing"
)

func TestScalingFactor_Ilvl50IsUnity(t *testing.T) {
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
			got := scalingFactor(tc.stat, 50)
			if math.Abs(float64(got)-1.0) > 0.001 {
				t.Errorf("scalingFactor(%s, 50) = %f, want 1.0", tc.name, got)
			}
		})
	}
}

func TestScalingFactor_Ilvl1NearlyNaked(t *testing.T) {
	// At ilvl 1 all factors should be very small (< 15% of BiS).
	stats := []struct {
		stat   StatID
		name   string
		maxPct float64
	}{
		{StatHull, "Hull", 0.04},         // ~2.5%
		{StatOutput, "Output", 0.08},     // ~6.1%
		{StatPlating, "Plating", 0.17},   // ~16.2%
		{StatTempo, "Tempo", 0.17},       // ~16.2%
		{StatIdentity, "Identity", 0.17}, // ~16.2%
		{StatMastery, "Mastery", 0.17},   // ~16.2%
	}
	for _, tc := range stats {
		t.Run(tc.name, func(t *testing.T) {
			got := float64(scalingFactor(tc.stat, 1))
			if got > tc.maxPct {
				t.Errorf("scalingFactor(%s, 1) = %.4f, want < %.3f (nearly naked)", tc.name, got, tc.maxPct)
			}
			if got <= 0 {
				t.Errorf("scalingFactor(%s, 1) = %.4f, want > 0", tc.name, got)
			}
		})
	}
}

func TestScalingFactor_Ilvl0IsZero(t *testing.T) {
	got := scalingFactor(StatHull, 0)
	if got != 0 {
		t.Errorf("scalingFactor(Hull, 0) = %f, want 0", got)
	}
	got = scalingFactor(StatHull, -1)
	if got != 0 {
		t.Errorf("scalingFactor(Hull, -1) = %f, want 0", got)
	}
}

func TestScalingFactor_HeritageToMaxRatios(t *testing.T) {
	// The design target ratios from stats.md: factor(50)/factor(15).
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
	const tolerance = 0.02
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f50 := float64(scalingFactor(tc.stat, 50))
			f15 := float64(scalingFactor(tc.stat, 15))
			ratio := f50 / f15
			off := math.Abs(ratio-tc.target) / tc.target
			if off > tolerance {
				t.Errorf("factor(50)/factor(15) for %s = %.4f, want ~%.2f (off by %.1f%%)",
					tc.name, ratio, tc.target, off*100)
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

func TestScaleStatLine_FullKitAtKeyIlvls(t *testing.T) {
	// Starter gear ilvl 50 totals (sum across all 6 items):
	//   Hull:     90 + 20 + 40 = 150
	//   Output:   22 + 8 + 25 + 8 + 12 + 8 = 83
	//   Plating:  12 + 8 = 20
	tests := []struct {
		name  string
		stat  StatID
		total float32
		ilvl  int
	}{
		// At ilvl 50, factor = 1.0 → full budget
		{"Hull@50", StatHull, 150, 50},
		{"Output@50", StatOutput, 83, 50},
		{"Plating@50", StatPlating, 20, 50},
	}
	const tolerance = 0.02
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line := StatLine{Stat: tc.stat, Value: tc.total}
			got := float64(ScaleStatLine(line, tc.ilvl))
			want := float64(tc.total) * float64(scalingFactor(tc.stat, tc.ilvl))
			off := math.Abs(got-want) / want
			if off > tolerance {
				t.Errorf("ScaleStatLine(%s, ilvl=%d) = %.2f, want ~%.2f", tc.name, tc.ilvl, got, want)
			}
		})
	}
}
