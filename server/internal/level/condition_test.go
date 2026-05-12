package level

import "testing"

func TestEvalCondition(t *testing.T) {
	dead := map[int]bool{1: true, 2: true}

	tests := []struct {
		cond  string
		state ZoneState
		want  bool
	}{
		{"", ZoneState{}, true},
		{"default", ZoneState{}, true},
		{"boss_dead", ZoneState{}, false},
		{"boss_dead", ZoneState{BossDefeated: true}, true},
		{"pack_1_cleared", ZoneState{}, false},
		{"pack_1_cleared", ZoneState{DeadGroupIDs: dead}, true},
		{"pack_2_cleared", ZoneState{DeadGroupIDs: dead}, true},
		{"pack_3_cleared", ZoneState{DeadGroupIDs: dead}, false},
		{"unknown_condition", ZoneState{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			got := EvalCondition(tt.cond, tt.state)
			if got != tt.want {
				t.Errorf("EvalCondition(%q, %+v) = %v, want %v",
					tt.cond, tt.state, got, tt.want)
			}
		})
	}
}

func TestConditionPriority(t *testing.T) {
	tests := []struct {
		cond string
		want int
	}{
		{"", 0},
		{"default", 0},
		{"pack_1_cleared", 1},
		{"pack_2_cleared", 2},
		{"boss_dead", 100},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			got := ConditionPriority(tt.cond)
			if got != tt.want {
				t.Errorf("ConditionPriority(%q) = %d, want %d", tt.cond, got, tt.want)
			}
		})
	}
}
