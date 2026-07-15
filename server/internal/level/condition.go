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
	DeadBossNums map[int]bool // BossNum → dead (mid-run checkpoints)
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
	// "boss_N_dead" pattern (checkpoint after a mid-run boss)
	if strings.HasPrefix(cond, "boss_") && strings.HasSuffix(cond, "_dead") {
		mid := cond[len("boss_") : len(cond)-len("_dead")]
		if _, err := fmt.Sscanf(mid, "%d", &n); err == nil {
			return state.DeadBossNums[n]
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
	case strings.HasPrefix(cond, "boss_") && strings.HasSuffix(cond, "_dead"):
		var n int
		mid := cond[len("boss_") : len(cond)-len("_dead")]
		if _, err := fmt.Sscanf(mid, "%d", &n); err == nil {
			return 50 + n // between pack checkpoints and the final boss_dead
		}
		return 0
	default:
		return 0
	}
}
