package item

import "math"

// scalingCoeffs maps each stat to its quadratic-curve coefficient.
// Formula: scalingFactor = 1.0 + coeff * pow(ilvl-1, 2)
// Coefficients derived so that scalingFactor(50) hits design targets
// (ilvl 1→50 ratios from stats.md):
//
//	Hull=3.0x, Output=2.25x, secondaries=1.75x
//
// Since pow(49, 2) = 2401, coeff = (target - 1) / 2401.
var scalingCoeffs = [StatCount]float64{
	StatHull:     2.0 / 2401.0,  // target 3.0x at ilvl 50
	StatOutput:   1.25 / 2401.0, // target 2.25x at ilvl 50
	StatPlating:  0.75 / 2401.0, // target 1.75x at ilvl 50
	StatTempo:    0.75 / 2401.0, // target 1.75x at ilvl 50
	StatIdentity: 0.75 / 2401.0, // target 1.75x at ilvl 50
	StatMastery:  0.75 / 2401.0, // target 1.75x at ilvl 50
}

// scalingFactor returns the multiplicative scaling factor for a stat at
// a given item level. At ilvl 1 the factor is exactly 1.0 (baseline).
func scalingFactor(stat StatID, ilvl int) float32 {
	if ilvl <= 1 {
		return 1.0
	}
	coeff := scalingCoeffs[stat]
	return float32(1.0 + coeff*math.Pow(float64(ilvl-1), 2))
}

// ScaleStatLine returns the scaled stat value for a given ilvl.
// Uses a per-stat power curve: value * scalingFactor(stat, ilvl).
func ScaleStatLine(line StatLine, ilvl int) float32 {
	return line.Value * scalingFactor(line.Stat, ilvl)
}

// ComputeStats aggregates stats from all equipped items.
func ComputeStats(equipped [SlotCount]*Item) Stats {
	var s Stats
	for _, it := range equipped {
		if it == nil {
			continue
		}
		def := DefRegistry[it.DefID]
		if def == nil {
			continue
		}
		for _, line := range def.StatLines {
			val := ScaleStatLine(line, it.ILvl)
			switch line.Stat {
			case StatHull:
				s.Hull += val
			case StatOutput:
				s.Output += val
			case StatPlating:
				s.Plating += val
			case StatTempo:
				s.Tempo += val
			case StatIdentity:
				s.Identity += val
			case StatMastery:
				s.Mastery += val
			}
		}
	}
	return s
}

// ComputeStatsForItem returns the scaled stat lines for a single item.
func ComputeStatsForItem(it *Item) []struct {
	Stat  StatID
	Value float32
} {
	if it == nil {
		return nil
	}
	def := DefRegistry[it.DefID]
	if def == nil {
		return nil
	}
	result := make([]struct {
		Stat  StatID
		Value float32
	}, len(def.StatLines))
	for i, line := range def.StatLines {
		result[i].Stat = line.Stat
		result[i].Value = ScaleStatLine(line, it.ILvl)
	}
	return result
}
