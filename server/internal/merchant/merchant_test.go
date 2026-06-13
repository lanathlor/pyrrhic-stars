package merchant

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

func TestIsTierUnlocked(t *testing.T) {
	tests := []struct {
		name      string
		tier      int
		bestScore int
		maxScore  int
		want      bool
	}{
		{"tier 0 always unlocked", 0, 0, 125, true},
		{"tier 1 unlocked at 20%", 1, 25, 125, true},
		{"tier 1 locked below 20%", 1, 24, 125, false},
		{"tier 2 unlocked at 45%", 2, 57, 125, true},
		{"tier 2 locked below 45%", 2, 55, 125, false},
		{"tier 3 unlocked at 80%", 3, 100, 125, true},
		{"tier 3 locked below 80%", 3, 99, 125, false},
		{"invalid tier negative", -1, 100, 125, false},
		{"invalid tier too high", 4, 100, 125, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTierUnlocked(tt.tier, tt.bestScore, tt.maxScore)
			if got != tt.want {
				t.Errorf("IsTierUnlocked(%d, %d, %d) = %v, want %v", tt.tier, tt.bestScore, tt.maxScore, got, tt.want)
			}
		})
	}
}

func TestRequiredScore(t *testing.T) {
	tests := []struct {
		name     string
		tier     int
		maxScore int
		want     int
	}{
		{"tier 0 always free", 0, 125, 0},
		{"tier 1 at 20%", 1, 125, 25},
		{"tier 2 at 45%", 2, 125, 56},
		{"tier 3 at 80%", 3, 125, 100},
		{"invalid tier negative", -1, 125, 0},
		{"invalid tier too high", 4, 125, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RequiredScore(tt.tier, tt.maxScore); got != tt.want {
				t.Errorf("RequiredScore(%d, %d) = %d, want %d", tt.tier, tt.maxScore, got, tt.want)
			}
		})
	}
}

func TestMerchantItemsCount(t *testing.T) {
	if got := len(MerchantItems); got != 6 {
		t.Errorf("len(MerchantItems) = %d, want 6", got)
	}
}

func TestTiersCount(t *testing.T) {
	if got := len(Tiers); got != 4 {
		t.Errorf("len(Tiers) = %d, want 4", got)
	}
}
