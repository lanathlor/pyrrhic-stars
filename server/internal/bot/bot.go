package bot

import (
	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/entity"
)

// MaxBotsPerPlayer is the maximum number of bots one player can spawn.
// Group max is 5, so player + 4 bots.
const MaxBotsPerPlayer = 4

// RescaleFunc is called when bot count changes so the zone can rescale
// enemy HP/damage for the new total player count.
type RescaleFunc func(totalPlayerCount int)

// BotRespawnDelay is how long a bot stays dead before auto-respawning.
const BotRespawnDelay float32 = 5.0

// LiveBot wraps a bosstest puppet for use in a live zone.
type LiveBot struct {
	Puppet          *bosstest.PlayerPuppet
	OwnerID         uint16
	BotID           uint16
	inCombat        bool
	followOffset    entity.Vec3
	deathTimer      float32 // counts up while dead; respawns at BotRespawnDelay
	currentBossName string  // DefName of boss whose tree is active ("" = default spec tree)
}
