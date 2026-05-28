package entity

import "testing"

// --- Player stat-read hot path (called every tick or every ability commit) ---

func benchPlayerWithGear() *Player {
	p := NewPlayer(1, ClassGunner)
	p.GearStats = GearStats{
		Hull:     80,
		Output:   55,
		Plating:  15,
		Tempo:    20,
		Identity: 12,
		Mastery:  8,
	}
	p.RecalcStats()
	p.Health = p.MaxHealth * 0.8
	return p
}

func BenchmarkRecalcStats_Gunner(b *testing.B) {
	p := benchPlayerWithGear()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.RecalcStats()
	}
}

func BenchmarkRecalcStats_Vanguard(b *testing.B) {
	p := NewPlayer(1, ClassVanguard)
	p.GearStats = GearStats{Hull: 80, Output: 55, Plating: 15, Tempo: 20, Identity: 30, Mastery: 8}
	p.RecalcStats()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.RecalcStats()
	}
}

func BenchmarkRecalcStats_BladeDancer(b *testing.B) {
	p := NewPlayer(1, ClassBladeDancer)
	p.GearStats = GearStats{Hull: 80, Output: 55, Plating: 15, Tempo: 20, Identity: 30, Mastery: 8}
	p.RecalcStats()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.RecalcStats()
	}
}

// --- Hot-path stat reads (called every tick in CombatSystem/AbilityEngine) ---

func BenchmarkTempoMult(b *testing.B) {
	p := benchPlayerWithGear()
	b.ReportAllocs()
	for b.Loop() {
		_ = p.TempoMult()
	}
}

func BenchmarkCommitterDamageMult(b *testing.B) {
	p := benchPlayerWithGear()
	p.AddBuff(ActiveBuff{ID: "overclock", Type: BuffDamageMult, Value: 1.3, Duration: 5.0})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = p.CommitterDamageMult()
	}
}

func BenchmarkTenacityEfficiency(b *testing.B) {
	p := NewPlayer(1, ClassVanguard)
	p.GearStats.Identity = 30
	b.ReportAllocs()
	for b.Loop() {
		_ = p.TenacityEfficiency()
	}
}

func BenchmarkApplyDamage_WithGear(b *testing.B) {
	p := benchPlayerWithGear()
	p.AddBuff(ActiveBuff{ID: AbilityVgBlock, Type: BuffDamageReduction, Value: 0.5, Duration: -1})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset health to avoid killing the player
		p.Health = p.MaxHealth
		p.Alive = true
		p.State = PlayerStateMove
		p.ApplyDamage(25)
	}
}

func BenchmarkApplyDamage_ShieldAbsorb(b *testing.B) {
	p := benchPlayerWithGear()
	if s := p.Resources[ResourceShield]; s != nil {
		s.Current = 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Health = p.MaxHealth
		p.Alive = true
		p.State = PlayerStateMove
		if s := p.Resources[ResourceShield]; s != nil {
			s.Current = 100
		}
		p.ApplyDamage(25)
	}
}

func BenchmarkDamageReduction(b *testing.B) {
	p := benchPlayerWithGear()
	p.AddBuff(ActiveBuff{ID: "armor_up", Type: BuffDamageReduction, Value: 0.8, Duration: -1})
	p.AddBuff(ActiveBuff{ID: "fortify", Type: BuffDamageReduction, Value: 0.9, Duration: -1})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = p.DamageReduction()
	}
}

// --- NewPlayer allocation cost ---

func BenchmarkNewPlayer_Gunner(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = NewPlayer(1, ClassGunner)
	}
}

func BenchmarkNewPlayer_Vanguard(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = NewPlayer(1, ClassVanguard)
	}
}

func BenchmarkNewPlayer_BladeDancer(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = NewPlayer(1, ClassBladeDancer)
	}
}

func BenchmarkNewPlayer_Arcanotechnicien(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = NewPlayer(1, ClassArcanotechnicien)
	}
}

// --- Arcanotechnicien / Harmonist benchmarks ---

func benchHarmonistWithGear() *Player {
	p := NewPlayer(1, ClassArcanotechnicien)
	p.GearStats = GearStats{
		Hull:     80,
		Output:   55,
		Plating:  15,
		Tempo:    20,
		Identity: 12,
		Mastery:  8,
	}
	p.RecalcStats()
	p.Health = p.MaxHealth * 0.8
	return p
}

func BenchmarkRecalcStats_Arcanotechnicien(b *testing.B) {
	p := benchHarmonistWithGear()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.RecalcStats()
	}
}

func BenchmarkFluxCommitment_TickRegen(b *testing.B) {
	p := benchHarmonistWithGear()
	// Spend some flux so regen has work to do
	for i := range p.FluxCommit.Pools {
		p.FluxCommit.Pools[i].Current = p.FluxCommit.Pools[i].Max * 0.5
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset pools so regen doesn't cap out
		for i := range p.FluxCommit.Pools {
			p.FluxCommit.Pools[i].Current = p.FluxCommit.Pools[i].Max * 0.5
		}
		p.FluxCommit.TickRegen(0.05)
	}
}

func BenchmarkFluxCommitment_GetPool(b *testing.B) {
	p := benchHarmonistWithGear()
	fc := p.FluxCommit

	b.Run("first", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = fc.GetPool(fc.Pools[0].School)
		}
	})
	b.Run("last", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = fc.GetPool(fc.Pools[len(fc.Pools)-1].School)
		}
	})
	b.Run("miss", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = fc.GetPool(SchoolShadow)
		}
	})
}

func BenchmarkFluxCommitment_SpendFromSchool(b *testing.B) {
	p := benchHarmonistWithGear()
	fc := p.FluxCommit

	b.Run("first", func(b *testing.B) {
		school := fc.Pools[0].School
		b.ReportAllocs()
		for b.Loop() {
			fc.Pools[0].Current = fc.Pools[0].Max
			fc.SpendFromSchool(school, 5)
		}
	})
	b.Run("last", func(b *testing.B) {
		last := len(fc.Pools) - 1
		school := fc.Pools[last].School
		b.ReportAllocs()
		for b.Loop() {
			fc.Pools[last].Current = fc.Pools[last].Max
			fc.SpendFromSchool(school, 5)
		}
	})
	b.Run("miss", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			fc.SpendFromSchool(SchoolShadow, 5)
		}
	})
}

func BenchmarkConfluence_Tick(b *testing.B) {
	b.Run("idle", func(b *testing.B) {
		c := &ConfluenceState{Stacks: 3, MaxStacks: 5, IdleTimer: 1.0}
		b.ReportAllocs()
		for b.Loop() {
			c.IdleTimer = 1.0
			c.Stacks = 3
			c.Tick(0.05)
		}
	})
	b.Run("decaying", func(b *testing.B) {
		c := &ConfluenceState{Stacks: 5, MaxStacks: 5, IdleTimer: 4.5, DecayTimer: 0.9}
		b.ReportAllocs()
		for b.Loop() {
			c.Stacks = 5
			c.IdleTimer = 4.5
			c.DecayTimer = 0.9
			c.Tick(0.05)
		}
	})
}

func BenchmarkConfluence_AbilityPowerMult(b *testing.B) {
	c := &ConfluenceState{Stacks: 3, MaxStacks: 5}
	b.ReportAllocs()
	for b.Loop() {
		_ = c.AbilityPowerMult()
	}
}

func BenchmarkApplyLoadout(b *testing.B) {
	p := NewPlayer(1, ClassArcanotechnicien)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.ApplyLoadout()
	}
}

func BenchmarkAffinityCostMult(b *testing.B) {
	p := benchHarmonistWithGear()

	b.Run("primary", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = p.AffinityCostMult(SchoolBioarcanotechnic)
		}
	})
	b.Run("secondary", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = p.AffinityCostMult(SchoolAerokinetic)
		}
	})
	b.Run("off", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = p.AffinityCostMult(SchoolShadow)
		}
	})
}

func BenchmarkSpendFluxBySchool(b *testing.B) {
	p := benchHarmonistWithGear()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset pool so spend succeeds each iteration
		pool := p.FluxCommit.GetPool(SchoolBiometabolic)
		pool.Current = pool.Max
		p.SpendFluxBySchool(SchoolBiometabolic, 5)
	}
}

func BenchmarkSyncFluxAggregate(b *testing.B) {
	p := benchHarmonistWithGear()
	b.ReportAllocs()
	for b.Loop() {
		p.SyncFluxAggregate()
	}
}

func BenchmarkSympatheticFieldRadius(b *testing.B) {
	p := benchHarmonistWithGear()
	b.ReportAllocs()
	for b.Loop() {
		_ = p.SympatheticFieldRadius()
	}
}
