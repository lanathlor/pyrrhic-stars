package level

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// EnemySpawnPoint defines where and what enemy spawns in a level.
type EnemySpawnPoint struct {
	Position    entity.Vec3
	DefName     string // maps to enemyai.DefRegistry key
	IsBoss      bool
	PatrolA     entity.Vec3 // first patrol waypoint
	PatrolB     entity.Vec3 // second patrol waypoint
	AggroRadius float32     // detection distance
	LeashRadius float32     // max distance from spawn before reset
	GroupID     int         // mobs with the same GroupID aggro together (0 = no group)
}

// Level holds static geometry and spatial data for a zone.
type Level struct {
	// Player boundaries (for clamping player positions)
	PlayerBoundsMinX, PlayerBoundsMaxX float32
	PlayerBoundsMinZ, PlayerBoundsMaxZ float32

	// Enemy boundaries (for clamping enemy positions)
	EnemyBoundsMinX, EnemyBoundsMaxX float32
	EnemyBoundsMinZ, EnemyBoundsMaxZ float32

	// Obstacles for collision and LoS
	Obstacles []combat.Obstacle

	// Spawn points
	PlayerSpawns []entity.Vec3
	EnemySpawns  []EnemySpawnPoint

	// Arena entry trigger Z threshold (0 = disabled)
	ArenaEntryZ float32

	// Boss room entry Z threshold (gate between hallway and boss room)
	BossRoomEntryZ float32

	// Default enemy collision radius
	EnemyRadius float32
}

// ClampPlayer restricts a position within player bounds.
func (l *Level) ClampPlayer(pos *entity.Vec3) {
	pos.X = entity.Clamp(pos.X, l.PlayerBoundsMinX, l.PlayerBoundsMaxX)
	pos.Z = entity.Clamp(pos.Z, l.PlayerBoundsMinZ, l.PlayerBoundsMaxZ)
}

// ClampEnemy restricts a position within enemy bounds.
func (l *Level) ClampEnemy(pos *entity.Vec3) {
	pos.X = entity.Clamp(pos.X, l.EnemyBoundsMinX, l.EnemyBoundsMaxX)
	pos.Z = entity.Clamp(pos.Z, l.EnemyBoundsMinZ, l.EnemyBoundsMaxZ)
	if pos.Y < 0.1 {
		pos.Y = 0.1
	}
}
