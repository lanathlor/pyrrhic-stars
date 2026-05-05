package combatlog

import "strconv"

// FormatPlayerID returns a stable string identifier for a player peer ID.
func FormatPlayerID(id uint16) string {
	return "player_" + strconv.FormatUint(uint64(id), 10)
}

// FormatEnemyID returns a stable string identifier for an enemy entity ID.
func FormatEnemyID(id uint16) string {
	return "enemy_" + strconv.FormatUint(uint64(id), 10)
}
