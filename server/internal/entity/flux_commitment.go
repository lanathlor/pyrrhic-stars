package entity

// FluxPool represents a single school's sub-pool within a Flux Commitment.
// The Arcanotechnicien distributes their total Flux reserve across schools;
// each school gets a proportional share of max capacity and regen rate.
type FluxPool struct {
	School     string
	Percentage float32 // fraction of total (0.0–1.0)
	Current    float32
	Max        float32
	Regen      float32 // per-second regen for this pool
}

// FluxCommitment tracks how a player's Flux reserve is distributed across
// schools. Each pool regenerates independently at its proportional rate.
type FluxCommitment struct {
	Pools      []FluxPool
	TotalMax   float32
	TotalRegen float32
}

// GetPool returns the pool for the given school, or nil if not committed.
func (fc *FluxCommitment) GetPool(school string) *FluxPool {
	for i := range fc.Pools {
		if fc.Pools[i].School == school {
			return &fc.Pools[i]
		}
	}
	return nil
}

// SpendFromSchool deducts amount from the named school's pool.
// Returns false if the school is not committed or has insufficient flux.
func (fc *FluxCommitment) SpendFromSchool(school string, amount float32) bool {
	pool := fc.GetPool(school)
	if pool == nil || pool.Current < amount {
		return false
	}
	pool.Current -= amount
	return true
}

// TickRegen advances regeneration for all pools by dt seconds.
// Each pool regenerates independently and is capped at its own max.
func (fc *FluxCommitment) TickRegen(dt float32) {
	for i := range fc.Pools {
		fc.Pools[i].Current += fc.Pools[i].Regen * dt
		if fc.Pools[i].Current > fc.Pools[i].Max {
			fc.Pools[i].Current = fc.Pools[i].Max
		}
	}
}

// SetCommitment redistributes the total flux reserve across the given schools.
// Each school gets a fraction of TotalMax and TotalRegen based on its percentage.
// Pools are initialized at full capacity.
func (fc *FluxCommitment) SetCommitment(schools map[string]float32) {
	fc.Pools = fc.Pools[:0]
	for school, pct := range schools {
		fc.Pools = append(fc.Pools, FluxPool{
			School:     school,
			Percentage: pct,
			Current:    fc.TotalMax * pct,
			Max:        fc.TotalMax * pct,
			Regen:      fc.TotalRegen * pct,
		})
	}
}
