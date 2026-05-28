package entity

import "testing"

func TestGetMoveSpeed(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 4.0},
		{testPhase2, 2, 5.0},
		{testPhase3, 3, 6.0},
		{"default falls to phase 1", 0, 4.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetMoveSpeed(); got != tt.want {
				t.Errorf("GetMoveSpeed() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetMeleeDamage(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 30.0},
		{testPhase2, 2, 30.0},
		{testPhase3, 3, 35.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetMeleeDamage(); got != tt.want {
				t.Errorf("GetMeleeDamage() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetMeleeTelegraphTime(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.2},
		{testPhase2, 2, 0.9},
		{testPhase3, 3, 0.7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.getMeleeTelegraphTime(); got != tt.want {
				t.Errorf("getMeleeTelegraphTime() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetRangedTelegraphTime(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.0},
		{testPhase2, 2, 0.8},
		{testPhase3, 3, 0.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.getRangedTelegraphTime(); got != tt.want {
				t.Errorf("getRangedTelegraphTime() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetRangedPerProjectileDamage(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 20.0},
		{testPhase2, 2, 15.0},
		{testPhase3, 3, 12.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetRangedPerProjectileDamage(); got != tt.want {
				t.Errorf("GetRangedPerProjectileDamage() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetRangedBurstCount(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  int
	}{
		{testPhase1, 1, 1},
		{testPhase2, 2, 2},
		{testPhase3, 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetRangedBurstCount(); got != tt.want {
				t.Errorf("GetRangedBurstCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetAoEDamage(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 40.0},
		{testPhase2, 2, 40.0},
		{testPhase3, 3, 45.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetAoEDamage(); got != tt.want {
				t.Errorf("GetAoEDamage() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetAoERadius(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 5.0},
		{testPhase2, 2, 6.0},
		{testPhase3, 3, 7.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetAoERadius(); got != tt.want {
				t.Errorf("GetAoERadius() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetAoETelegraphTime(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.5},
		{testPhase2, 2, 1.2},
		{testPhase3, 3, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.getAoETelegraphTime(); got != tt.want {
				t.Errorf("getAoETelegraphTime() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetChargeDamage(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 35.0},
		{testPhase2, 2, 35.0},
		{testPhase3, 3, 40.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetChargeDamage(); got != tt.want {
				t.Errorf("GetChargeDamage() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetChargeSpeed(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 12.0},
		{testPhase2, 2, 14.0},
		{testPhase3, 3, 16.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetChargeSpeed(); got != tt.want {
				t.Errorf("GetChargeSpeed() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetChargeTelegraphTime(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.0},
		{testPhase2, 2, 0.8},
		{testPhase3, 3, 0.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.getChargeTelegraphTime(); got != tt.want {
				t.Errorf("getChargeTelegraphTime() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetChargeMaxDistance(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 15.0},
		{testPhase2, 2, 18.0},
		{testPhase3, 3, 20.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.GetChargeMaxDistance(); got != tt.want {
				t.Errorf("GetChargeMaxDistance() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestGetCooldownTime(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  float32
	}{
		{testPhase1, 1, 1.5},
		{testPhase2, 2, 1.2},
		{testPhase3, 3, 0.9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.getCooldownTime(); got != tt.want {
				t.Errorf("getCooldownTime() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestPhaseWeights(t *testing.T) {
	tests := []struct {
		name  string
		phase int
		want  [4]int
	}{
		{testPhase1, 1, [4]int{30, 30, 20, 20}},
		{testPhase2, 2, [4]int{25, 25, 25, 25}},
		{testPhase3, 3, [4]int{20, 20, 25, 35}},
		{"default falls to phase 1", 0, [4]int{30, 30, 20, 20}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEnemy(0, 1000, "test")
			e.Phase = tt.phase
			if got := e.PhaseWeights(); got != tt.want {
				t.Errorf("PhaseWeights() = %v, want %v", got, tt.want)
			}
		})
	}
}
