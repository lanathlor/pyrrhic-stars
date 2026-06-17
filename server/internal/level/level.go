package level

import (
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// PlayerSpawn defines a player spawn point with an optional condition.
type PlayerSpawn struct {
	Position  entity.Vec3
	Condition string // "", "boss_dead", "pack_N_cleared" — empty means always active
}

// EnemySpawnPoint defines where and what enemy spawns in a level.
type EnemySpawnPoint struct {
	Position        entity.Vec3
	DefName         string // maps to enemyai.DefRegistry key
	IsBoss          bool
	PatrolA         entity.Vec3   // first patrol waypoint
	PatrolB         entity.Vec3   // second patrol waypoint
	PatrolWaypoints []entity.Vec3 // multi-point patrol (overrides A/B when len >= 2)
	AggroRadius     float32       // detection distance
	LeashRadius     float32       // max distance from spawn before reset
	GroupID         int           // mobs with the same GroupID aggro together (0 = no group)
	Condition       string        // spawn condition (empty = always active)
}

// PortalDef defines a zone transition point.
type PortalDef struct {
	Name              string
	Position          entity.Vec3
	TargetZone        string
	InteractionRadius float32
	Condition         string // empty = always active
}

// GateDef defines a data-driven barrier that opens/closes on game flow events.
// When closed, the gate acts as an obstacle for collision and LoS.
type GateDef struct {
	ID            string      // unique identifier (e.g. "boss_gate")
	Position      entity.Vec3 // center of the gate geometry
	HalfExtents   entity.Vec3 // half-size on each axis (for obstacle creation)
	CloseOn       []string    // event names that close this gate (e.g. "boss_activated")
	OpenOn        []string    // event names that open this gate (e.g. "boss_dead")
	DefaultClosed bool        // initial state when zone starts
	PushAxis      string      // "x", "y", "z" or "" — axis to push players on gate close
	PushOffset    float32     // how far to push players past the gate center
}

// ToObstacle converts a gate definition to a collision obstacle.
func (g *GateDef) ToObstacle() combat.Obstacle {
	return combat.Obstacle{
		CX:     g.Position.X,
		CZ:     g.Position.Z,
		HX:     g.HalfExtents.X,
		HZ:     g.HalfExtents.Z,
		BaseY:  g.Position.Y - g.HalfExtents.Y,
		Height: g.HalfExtents.Y * 2,
	}
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
	PlayerSpawns []PlayerSpawn
	SpawnYaw     float32 // initial facing direction (radians) for spawned players
	EnemySpawns  []EnemySpawnPoint

	// Instance entry trigger Z threshold (0 = disabled)
	InstanceEntryZ float32

	// Gates (data-driven barriers that open/close on game flow events)
	Gates []GateDef

	// Default enemy collision radius
	EnemyRadius float32

	// ZoneType is "instanced" or "open_world" (from JSON).
	ZoneType string

	// ClearTimeSeconds is the per-instance dungeon clear timer in seconds
	// (0 = unset, the gameflow system falls back to its default limit).
	ClearTimeSeconds float32

	// NPC spawn points
	NPCSpawns []NPCSpawnPoint

	// Portals (zone transition points)
	Portals []PortalDef

	// Navmesh for multi-layer Y resolution (nil if zone has no navmesh data)
	Navmesh *Navmesh
}

// NPCSpawnPoint defines an open-world NPC with patrol waypoints.
type NPCSpawnPoint struct {
	DefName      string  // visual definition name
	Speed        float32 // walk speed (m/s)
	IdleDuration float32 // seconds to idle at each waypoint
	Waypoints    []entity.Vec3
	Metadata     map[string]string // optional key-value pairs (e.g. "tier": "0")
}

// FlowEventName maps a flow type constant to the gate trigger string used in JSON.
var FlowEventName = map[uint8]string{
	1:  "spawn_players",
	2:  "fight_start",
	5:  "return_lobby",
	7:  "boss_dead",
	8:  "all_dead",
	9:  "boss_activated",
	10: "boss_reset",
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
	if l.Navmesh != nil {
		if floorY, ok := l.Navmesh.SampleY(pos.X, pos.Z, pos.Y); ok {
			pos.Y = floorY + 0.1
		}
	} else if pos.Y < 0.1 {
		pos.Y = 0.1
	}
}

// ResolveY finds the floor Y at (x, z) closest to nearY.
// Falls back to 0.1 if no navmesh or no polygon contains the point.
func (l *Level) ResolveY(x, z, nearY float32) float32 {
	if l.Navmesh != nil {
		if floorY, ok := l.Navmesh.SampleY(x, z, nearY); ok {
			return floorY
		}
	}
	return 0.1
}
