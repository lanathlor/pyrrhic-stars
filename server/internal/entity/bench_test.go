package entity

import "testing"

// --- Player stat-read hot path (called every tick or every ability cast) ---

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

func BenchmarkCasterDamageMult(b *testing.B) {
	p := benchPlayerWithGear()
	p.AddBuff(ActiveBuff{ID: "overclock", Type: BuffDamageMult, Value: 1.3, Duration: 5.0})
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = p.CasterDamageMult()
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
	p.AddBuff(ActiveBuff{ID: "vg_block", Type: BuffDamageReduction, Value: 0.5, Duration: -1})
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
	if s := p.Resources["shield"]; s != nil {
		s.Current = 100
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.Health = p.MaxHealth
		p.Alive = true
		p.State = PlayerStateMove
		if s := p.Resources["shield"]; s != nil {
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
