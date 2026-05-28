package entity

import "testing"

func TestAffinityCostMult(t *testing.T) {
	tests := []struct {
		name   string
		player *Player
		school string
		want   float32
	}{
		{
			name:   "primary school returns 1.0",
			player: &Player{PrimarySchools: []string{SchoolFrost, SchoolFire}, SecondarySchools: []string{SchoolAerokinetic}},
			school: SchoolFrost,
			want:   1.0,
		},
		{
			name:   "secondary school returns 1.25",
			player: &Player{PrimarySchools: []string{SchoolFrost}, SecondarySchools: []string{SchoolAerokinetic, SchoolPure}},
			school: SchoolAerokinetic,
			want:   1.25,
		},
		{
			name:   "off-affinity school returns 1.5",
			player: &Player{PrimarySchools: []string{SchoolFrost}, SecondarySchools: []string{SchoolAerokinetic}},
			school: SchoolShadow,
			want:   1.5,
		},
		{
			name:   "no schools configured returns 1.0",
			player: &Player{},
			school: SchoolFrost,
			want:   1.0,
		},
		{
			name:   "harmonist bioarcanotechnic is primary",
			player: NewPlayer(1, ClassArcanotechnicien),
			school: SchoolBioarcanotechnic,
			want:   1.0,
		},
		{
			name:   "harmonist aerokinetic is secondary",
			player: NewPlayer(1, ClassArcanotechnicien),
			school: SchoolAerokinetic,
			want:   1.25,
		},
		{
			name:   "harmonist shadow is off-affinity",
			player: NewPlayer(1, ClassArcanotechnicien),
			school: SchoolShadow,
			want:   1.5,
		},
		{
			name:   "gunner has no scaling",
			player: NewPlayer(1, ClassGunner),
			school: SchoolFrost,
			want:   1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.player.AffinityCostMult(tt.school)
			if got != tt.want {
				t.Errorf("AffinityCostMult(%q) = %v, want %v", tt.school, got, tt.want)
			}
		})
	}
}

func TestSpendFluxBySchool_AffinityScaling(t *testing.T) {
	// Create a harmonist with known flux pools.
	p := NewPlayer(1, ClassArcanotechnicien)
	// Harmonist default: bioarcanotechnic=50%, biometabolic=30%, frost=10%, aerokinetic=10%
	// Total flux = 160, so frost pool = 16, aerokinetic pool = 16.

	// Frost is primary (1.0x). Spending 10 base should deduct exactly 10.
	frostPool := p.FluxCommit.GetPool(SchoolFrost)
	if frostPool == nil {
		t.Fatal("frost pool not found")
	}
	startFrost := frostPool.Current
	if !p.SpendFluxBySchool(SchoolFrost, 10) {
		t.Fatal("SpendFluxBySchool(frost, 10) should succeed")
	}
	if frostPool.Current != startFrost-10 {
		t.Errorf("frost pool: got %v, want %v", frostPool.Current, startFrost-10)
	}

	// Aerokinetic is secondary (1.25x). Spending 10 base should deduct 12.5.
	aeroPool := p.FluxCommit.GetPool(SchoolAerokinetic)
	if aeroPool == nil {
		t.Fatal("aerokinetic pool not found")
	}
	startAero := aeroPool.Current
	if !p.SpendFluxBySchool(SchoolAerokinetic, 10) {
		t.Fatal("SpendFluxBySchool(aerokinetic, 10) should succeed")
	}
	expectedDeduct := float32(12.5)
	if aeroPool.Current != startAero-expectedDeduct {
		t.Errorf("aerokinetic pool: got %v, want %v", aeroPool.Current, startAero-expectedDeduct)
	}
}

func TestSpendFluxBySchool_RejectsScaledInsufficient(t *testing.T) {
	// If you have 12 flux in aerokinetic (secondary, 1.25x) and try to spend
	// 10 base cost, scaled = 12.5 > 12 → should fail.
	p := NewPlayer(1, ClassArcanotechnicien)
	aeroPool := p.FluxCommit.GetPool(SchoolAerokinetic)
	if aeroPool == nil {
		t.Fatal("aerokinetic pool not found")
	}
	aeroPool.Current = 12.0

	if p.SpendFluxBySchool(SchoolAerokinetic, 10) {
		t.Error("SpendFluxBySchool should reject: 10 * 1.25 = 12.5 > 12.0")
	}
	// Pool should be unchanged.
	if aeroPool.Current != 12.0 {
		t.Errorf("pool should be unchanged: got %v, want 12.0", aeroPool.Current)
	}
}
