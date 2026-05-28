package entity

import (
	"cmp"
	"math"
	"slices"
	"testing"
)

func approxEq(a, b float32) bool {
	return math.Abs(float64(a-b)) < 0.001
}

func TestSetCommitmentDistributes(t *testing.T) {
	tests := []struct {
		name       string
		totalMax   float32
		totalRegen float32
		schools    map[string]float32
		wantPools  []FluxPool
	}{
		{
			name:       "harmonist default 4 schools",
			totalMax:   160,
			totalRegen: 7,
			schools: map[string]float32{
				SchoolBioarcanotechnic: 0.5,
				SchoolBiometabolic:     0.3,
				SchoolFrost:            0.1,
				SchoolAerokinetic:      0.1,
			},
			wantPools: []FluxPool{
				{School: SchoolAerokinetic, Percentage: 0.1, Current: 16, Max: 16, Regen: 0.7},
				{School: SchoolBioarcanotechnic, Percentage: 0.5, Current: 80, Max: 80, Regen: 3.5},
				{School: SchoolBiometabolic, Percentage: 0.3, Current: 48, Max: 48, Regen: 2.1},
				{School: SchoolFrost, Percentage: 0.1, Current: 16, Max: 16, Regen: 0.7},
			},
		},
		{
			name:       "single school 100%",
			totalMax:   200,
			totalRegen: 10,
			schools: map[string]float32{
				SchoolFire: 1.0,
			},
			wantPools: []FluxPool{
				{School: SchoolFire, Percentage: 1.0, Current: 200, Max: 200, Regen: 10},
			},
		},
		{
			name:       "two schools 50/50",
			totalMax:   100,
			totalRegen: 4,
			schools: map[string]float32{
				"alpha": 0.5,
				"beta":  0.5,
			},
			wantPools: []FluxPool{
				{School: "alpha", Percentage: 0.5, Current: 50, Max: 50, Regen: 2},
				{School: "beta", Percentage: 0.5, Current: 50, Max: 50, Regen: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FluxCommitment{TotalMax: tt.totalMax, TotalRegen: tt.totalRegen}
			fc.SetCommitment(tt.schools)

			if len(fc.Pools) != len(tt.wantPools) {
				t.Fatalf("got %d pools, want %d", len(fc.Pools), len(tt.wantPools))
			}

			// Sort both by school name for stable comparison (map iteration order is random).
			slices.SortFunc(fc.Pools, func(a, b FluxPool) int { return cmp.Compare(a.School, b.School) })
			slices.SortFunc(tt.wantPools, func(a, b FluxPool) int { return cmp.Compare(a.School, b.School) })

			for i, want := range tt.wantPools {
				got := fc.Pools[i]
				if got.School != want.School {
					t.Errorf("pool[%d].School = %q, want %q", i, got.School, want.School)
				}
				if !approxEq(got.Percentage, want.Percentage) {
					t.Errorf("pool[%d].Percentage = %f, want %f", i, got.Percentage, want.Percentage)
				}
				if !approxEq(got.Current, want.Current) {
					t.Errorf("pool[%d].Current = %f, want %f", i, got.Current, want.Current)
				}
				if !approxEq(got.Max, want.Max) {
					t.Errorf("pool[%d].Max = %f, want %f", i, got.Max, want.Max)
				}
				if !approxEq(got.Regen, want.Regen) {
					t.Errorf("pool[%d].Regen = %f, want %f", i, got.Regen, want.Regen)
				}
			}
		})
	}
}

func TestSpendFromSchool(t *testing.T) {
	tests := []struct {
		name        string
		school      string
		amount      float32
		wantOK      bool
		wantCurrent float32
	}{
		{"spend within budget", SchoolBioarcanotechnic, 30, true, 50},
		{"spend exact amount", SchoolBioarcanotechnic, 80, true, 0},
		{"spend too much", SchoolBioarcanotechnic, 81, false, 80},
		{"spend from smaller pool", SchoolFrost, 10, true, 6},
		{"spend too much from small pool", SchoolFrost, 17, false, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FluxCommitment{TotalMax: 160, TotalRegen: 7}
			fc.SetCommitment(map[string]float32{
				SchoolBioarcanotechnic: 0.5,
				SchoolBiometabolic:     0.3,
				SchoolFrost:            0.1,
				SchoolAerokinetic:      0.1,
			})

			ok := fc.SpendFromSchool(tt.school, tt.amount)
			if ok != tt.wantOK {
				t.Errorf("SpendFromSchool(%q, %f) = %v, want %v", tt.school, tt.amount, ok, tt.wantOK)
			}

			pool := fc.GetPool(tt.school)
			if pool == nil {
				t.Fatalf("pool %q should exist", tt.school)
			}
			if !approxEq(pool.Current, tt.wantCurrent) {
				t.Errorf("pool %q current = %f, want %f", tt.school, pool.Current, tt.wantCurrent)
			}
		})
	}
}

func TestSpendFromSchoolUnknown(t *testing.T) {
	fc := &FluxCommitment{TotalMax: 100, TotalRegen: 5}
	fc.SetCommitment(map[string]float32{SchoolFire: 1.0})

	ok := fc.SpendFromSchool("nonexistent", 10)
	if ok {
		t.Error("SpendFromSchool on unknown school should return false")
	}
}

func TestTickRegen(t *testing.T) {
	tests := []struct {
		name        string
		school      string
		spendFirst  float32
		dt          float32
		wantCurrent float32
	}{
		{"regen partial", SchoolBioarcanotechnic, 40, 2.0, 47},            // 80-40=40, +3.5*2=47
		{"regen caps at max", SchoolBioarcanotechnic, 5, 10.0, 80},        // 80-5=75, +3.5*10=110, capped at 80
		{"regen from empty", SchoolFrost, 16, 1.0, 0.7},                   // 16-16=0, +0.7*1=0.7
		{"no spend full pool stays full", SchoolBiometabolic, 0, 5.0, 48}, // already at max 48
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FluxCommitment{TotalMax: 160, TotalRegen: 7}
			fc.SetCommitment(map[string]float32{
				SchoolBioarcanotechnic: 0.5,
				SchoolBiometabolic:     0.3,
				SchoolFrost:            0.1,
				SchoolAerokinetic:      0.1,
			})

			if tt.spendFirst > 0 {
				fc.SpendFromSchool(tt.school, tt.spendFirst)
			}

			fc.TickRegen(tt.dt)

			pool := fc.GetPool(tt.school)
			if pool == nil {
				t.Fatalf("pool %q should exist", tt.school)
			}
			if !approxEq(pool.Current, tt.wantCurrent) {
				t.Errorf("pool %q after regen = %f, want %f", tt.school, pool.Current, tt.wantCurrent)
			}
		})
	}
}

func TestGetPoolNil(t *testing.T) {
	tests := []struct {
		name   string
		school string
	}{
		{"unknown school", "gravity"},
		{"empty string", ""},
		{"similar name", "bioarcanotechni"}, // off by one char
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FluxCommitment{TotalMax: 160, TotalRegen: 7}
			fc.SetCommitment(map[string]float32{
				SchoolBioarcanotechnic: 0.5,
				SchoolBiometabolic:     0.3,
			})

			pool := fc.GetPool(tt.school)
			if pool != nil {
				t.Errorf("GetPool(%q) should return nil, got %+v", tt.school, pool)
			}
		})
	}
}

func TestSetCommitmentReplacesExisting(t *testing.T) {
	fc := &FluxCommitment{TotalMax: 100, TotalRegen: 10}
	fc.SetCommitment(map[string]float32{"fire": 0.5, "ice": 0.5})

	if len(fc.Pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(fc.Pools))
	}

	// Redistribute to 3 pools.
	fc.SetCommitment(map[string]float32{"fire": 0.4, "ice": 0.4, "wind": 0.2})

	if len(fc.Pools) != 3 {
		t.Fatalf("expected 3 pools after redistribution, got %d", len(fc.Pools))
	}

	wind := fc.GetPool("wind")
	if wind == nil {
		t.Fatal("wind pool should exist after redistribution")
	}
	if !approxEq(wind.Max, 20) {
		t.Errorf("wind.Max = %f, want 20", wind.Max)
	}
}

func TestNewPlayerHarmonistHasFluxCommitment(t *testing.T) {
	p := NewPlayerWithSpec(1, ClassArcanotechnicien, SpecHarmonist)

	if p.FluxCommit == nil {
		t.Fatal("Harmonist should have FluxCommitment initialized")
	}
	if !approxEq(p.FluxCommit.TotalMax, 160) {
		t.Errorf("TotalMax = %f, want 160", p.FluxCommit.TotalMax)
	}
	if !approxEq(p.FluxCommit.TotalRegen, 3) {
		t.Errorf("TotalRegen = %f, want 3", p.FluxCommit.TotalRegen)
	}
	if len(p.FluxCommit.Pools) != 4 {
		t.Fatalf("expected 4 pools, got %d", len(p.FluxCommit.Pools))
	}

	// Verify bioarcanotechnic is the largest pool (50%).
	bio := p.FluxCommit.GetPool(SchoolBioarcanotechnic)
	if bio == nil {
		t.Fatal("bioarcanotechnic pool should exist")
	}
	if !approxEq(bio.Max, 80) {
		t.Errorf("bioarcanotechnic.Max = %f, want 80", bio.Max)
	}
	if !approxEq(bio.Regen, 1.5) {
		t.Errorf("bioarcanotechnic.Regen = %f, want 1.5", bio.Regen)
	}
}

func TestNewPlayerDestroyerHasFluxCommitmentNoPools(t *testing.T) {
	p := NewPlayerWithSpec(1, ClassArcanotechnicien, SpecDestroyer)

	if p.FluxCommit == nil {
		t.Fatal("Destroyer should have FluxCommitment initialized (pools set later)")
	}
	if !approxEq(p.FluxCommit.TotalMax, 200) {
		t.Errorf("TotalMax = %f, want 200", p.FluxCommit.TotalMax)
	}
	if len(p.FluxCommit.Pools) != 0 {
		t.Errorf("Destroyer should have 0 pools initially, got %d", len(p.FluxCommit.Pools))
	}
}

func TestNewPlayerNonArcanotechnicienNoFluxCommitment(t *testing.T) {
	classes := []string{ClassGunner, ClassVanguard, ClassBladeDancer}
	for _, cls := range classes {
		t.Run(cls, func(t *testing.T) {
			p := NewPlayer(1, cls)
			if p.FluxCommit != nil {
				t.Errorf("%s should not have FluxCommitment", cls)
			}
		})
	}
}
