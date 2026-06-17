package progression

import "testing"

func TestScripReward(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		maxScore int
		want     int
	}{
		{"zero score", 0, 100, 100},
		{"max score", 100, 100, 400},
		{"half score", 50, 100, 250},
		{"quarter score", 25, 100, 175},
		{"three quarter", 75, 100, 325},
		{"maxScore zero", 0, 0, 100},
		{"maxScore negative", 0, -1, 100},
		{"real max 125", 125, 125, 400},
		{"real mid 75", 75, 125, 280},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScripReward(tt.score, tt.maxScore)
			if got != tt.want {
				t.Errorf("ScripReward(%d, %d) = %d, want %d", tt.score, tt.maxScore, got, tt.want)
			}
		})
	}
}

func TestScripRewardClamped(t *testing.T) {
	// Negative score should clamp to 100.
	got := ScripReward(-10, 100)
	if got < 100 {
		t.Errorf("ScripReward(-10, 100) = %d, want >= 100", got)
	}
}
