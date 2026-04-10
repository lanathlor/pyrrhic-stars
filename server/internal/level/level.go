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

// ElevatorVolume describes a moving platform that allows vertical player movement.
// The server uses this to whitelist Y-axis changes inside the volume footprint.
type ElevatorVolume struct {
	CenterX, CenterZ float32
	HalfX, HalfZ     float32
	BottomY, TopY    float32
	Speed            float32 // max upward m/s (average; smoothstep peaks at ~1.5x)
}

// Level holds static geometry and spatial data for a zone.
type Level struct {
	// Player boundaries (for clamping player positions)
	PlayerBoundsMinX, PlayerBoundsMaxX float32
	PlayerBoundsMinY, PlayerBoundsMaxY float32 // Y bounds (0,0 = disabled)
	PlayerBoundsMinZ, PlayerBoundsMaxZ float32

	// Enemy boundaries (for clamping enemy positions)
	EnemyBoundsMinX, EnemyBoundsMaxX float32
	EnemyBoundsMinZ, EnemyBoundsMaxZ float32

	// Obstacles for collision and LoS
	Obstacles []combat.Obstacle

	// Elevator volumes for Y-axis validation
	Elevators []ElevatorVolume

	// Spawn points
	PlayerSpawns []entity.Vec3
	SpawnYaw     float32 // initial facing direction (radians) for spawned players
	EnemySpawns  []EnemySpawnPoint

	// Arena entry trigger Z threshold (0 = disabled)
	ArenaEntryZ float32

	// Boss room entry Z threshold (gate between hallway and boss room)
	BossRoomEntryZ float32

	// Default enemy collision radius
	EnemyRadius float32

	// NPC spawn points (hub only)
	NPCSpawns []NPCSpawnPoint
}

// NPCSpawnPoint defines a hub NPC with patrol waypoints.
type NPCSpawnPoint struct {
	DefName      string      // visual definition name
	Speed        float32     // walk speed (m/s)
	IdleDuration float32     // seconds to idle at each waypoint
	Waypoints    []entity.Vec3
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
