package level

import (
	"fmt"
	"strings"
)

// Condition name constants.
const (
	CondDefault  = "default"
	CondBossDead = "boss_dead"
)

// ZoneState captures the zone progression needed to evaluate spawn conditions.
type ZoneState struct {
	BossDefeated bool
	DeadGroupIDs map[int]bool
}

// EvalCondition checks a spawn condition tag against zone state.
// Empty string or "default" always returns true.
func EvalCondition(cond string, state ZoneState) bool {
	switch cond {
	case "", CondDefault:
		return true
	case CondBossDead:
		return state.BossDefeated
	}
	// "pack_N_cleared" pattern
	var n int
	if strings.HasPrefix(cond, "pack_") && strings.HasSuffix(cond, "_cleared") {
		mid := cond[len("pack_") : len(cond)-len("_cleared")]
		if _, err := fmt.Sscanf(mid, "%d", &n); err == nil {
			return state.DeadGroupIDs[n]
		}
	}
	return false
}

// ConditionPriority returns a rank for spawn condition progression.
// Higher rank = further into the dungeon. Used to pick the best checkpoint.
func ConditionPriority(cond string) int {
	switch {
	case cond == "" || cond == CondDefault:
		return 0
	case strings.HasPrefix(cond, "pack_") && strings.HasSuffix(cond, "_cleared"):
		var n int
		mid := cond[len("pack_") : len(cond)-len("_cleared")]
		if _, err := fmt.Sscanf(mid, "%d", &n); err == nil {
			return n // pack_1 = 1, pack_2 = 2, etc.
		}
		return 0
	case cond == CondBossDead:
		return 100
	default:
		return 0
	}
}
