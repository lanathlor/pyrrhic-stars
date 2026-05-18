package item

import "math"

// scalingPowers maps each stat to its power-curve exponent.
// Formula: scalingFactor = (ilvl / 50)^power
//
// Item definition values represent the stat at ilvl 50 (BiS reference).
// At ilvl 1 the factor is near zero (nearly naked).
// At ilvl 50 the factor is exactly 1.0.
//
// Powers derived so factor(50)/factor(15) matches design targets
// (heritage-floor → BiS ratios from stats.md):
//
//	Hull ≈ 3.0x, Output ≈ 2.25x, secondaries ≈ 1.75x
//
// power = ln(ratio) / ln(50/15)
var scalingPowers = [StatCount]float64{
	StatHull:     0.912, // 3.0x  from ilvl 15→50
	StatOutput:   0.674, // 2.25x from ilvl 15→50
	StatPlating:  0.465, // 1.75x from ilvl 15→50
	StatTempo:    0.465, // 1.75x from ilvl 15→50
	StatIdentity: 0.465, // 1.75x from ilvl 15→50
	StatMastery:  0.465, // 1.75x from ilvl 15→50
}

// scalingFactor returns the multiplicative scaling factor for a stat at
// a given item level. At ilvl 50 the factor is 1.0 (BiS reference).
// At ilvl 1 the factor is near zero (nearly naked).
func scalingFactor(stat StatID, ilvl int) float32 {
	if ilvl <= 0 {
		return 0
	}
	power := scalingPowers[stat]
	return float32(math.Pow(float64(ilvl)/50.0, power))
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
