package item

// ScaleStatLine returns the scaled stat value for a given ilvl.
// Phase 0: simple linear scaling (value * ilvl).
func ScaleStatLine(line StatLine, ilvl int) float32 {
	return line.Value * float32(ilvl)
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
