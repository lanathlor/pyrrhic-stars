package level

import (
	"encoding/json"
	"fmt"
	"os"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

const currentVersion = 3

type boundsJSON struct {
	MinX float32 `json:"min_x"`
	MaxX float32 `json:"max_x"`
	MinY float32 `json:"min_y"`
	MaxY float32 `json:"max_y"`
	MinZ float32 `json:"min_z"`
	MaxZ float32 `json:"max_z"`
}

type obstacleJSON struct {
	Name        string     `json:"name"`
	Center      [3]float32 `json:"center"`
	HalfExtents [3]float32 `json:"half_extents"`
}

type elevatorJSON struct {
	Name    string  `json:"name"`
	CenterX float32 `json:"center_x"`
	CenterZ float32 `json:"center_z"`
	HalfX   float32 `json:"half_x"`
	HalfZ   float32 `json:"half_z"`
	BottomY float32 `json:"bottom_y"`
	TopY    float32 `json:"top_y"`
	Speed   float32 `json:"speed"`
}

type vec3JSON struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type playerSpawnJSON struct {
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Z         float32 `json:"z"`
	Condition string  `json:"condition,omitempty"`
}

type enemySpawnJSON struct {
	X               float32    `json:"x"`
	Y               float32    `json:"y"`
	Z               float32    `json:"z"`
	DefName         string     `json:"def_name"`
	IsBoss          bool       `json:"is_boss,omitempty"`
	PatrolA         vec3JSON   `json:"patrol_a"`
	PatrolB         vec3JSON   `json:"patrol_b"`
	PatrolWaypoints []vec3JSON `json:"patrol_waypoints,omitempty"`
	AggroRadius     float32    `json:"aggro_radius"`
	LeashRadius     float32    `json:"leash_radius"`
	GroupID         int        `json:"group_id,omitempty"`
	Condition       string     `json:"condition,omitempty"`
}

type portalJSON struct {
	Name              string  `json:"name"`
	X                 float32 `json:"x"`
	Y                 float32 `json:"y"`
	Z                 float32 `json:"z"`
	TargetZone        string  `json:"target_zone"`
	InteractionRadius float32 `json:"interaction_radius"`
	Condition         string  `json:"condition,omitempty"`
}

type zoneTriggerJSON struct {
	Name      string  `json:"name"`
	TriggerID string  `json:"trigger_id"`
	Axis      string  `json:"axis"`
	Threshold float32 `json:"threshold"`
}

type npcSpawnJSON struct {
	DefName      string     `json:"def_name"`
	Speed        float32    `json:"speed"`
	IdleDuration float32    `json:"idle_duration"`
	Waypoints    []vec3JSON `json:"waypoints"`
}

type levelDataJSON struct {
	Version      int               `json:"version"`
	Zone         string            `json:"zone"`
	SourceScene  string            `json:"source_scene"`
	Bounds       boundsJSON        `json:"bounds"`
	Obstacles    []obstacleJSON    `json:"obstacles"`
	Elevators    []elevatorJSON    `json:"elevators,omitempty"`
	PlayerSpawns []playerSpawnJSON `json:"player_spawns"`
	SpawnYaw     float32           `json:"spawn_yaw,omitempty"`
	EnemySpawns  []enemySpawnJSON  `json:"enemy_spawns,omitempty"`
	NPCSpawns    []npcSpawnJSON    `json:"npc_spawns,omitempty"`
	Portals      []portalJSON      `json:"portals,omitempty"`
	ZoneTriggers []zoneTriggerJSON `json:"zone_triggers,omitempty"`
}

// loadLevelData reads a JSON level file and applies its geometry to l.
// Game-logic fields (ArenaEntryZ, BossRoomEntryZ, EnemyRadius) are left untouched.
func loadLevelData(path string, l *Level) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("level load %q: %w", path, err)
	}
	var ld levelDataJSON
	if err := json.Unmarshal(raw, &ld); err != nil {
		return fmt.Errorf("level parse %q: %w", path, err)
	}
	if ld.Version < 2 || ld.Version > currentVersion {
		return fmt.Errorf("level %q: version %d (want 2-%d)", path, ld.Version, currentVersion)
	}

	// Bounds
	l.PlayerBoundsMinX = ld.Bounds.MinX
	l.PlayerBoundsMaxX = ld.Bounds.MaxX
	l.PlayerBoundsMinY = ld.Bounds.MinY
	l.PlayerBoundsMaxY = ld.Bounds.MaxY
	l.PlayerBoundsMinZ = ld.Bounds.MinZ
	l.PlayerBoundsMaxZ = ld.Bounds.MaxZ

	l.EnemyBoundsMinX = ld.Bounds.MinX
	l.EnemyBoundsMaxX = ld.Bounds.MaxX
	l.EnemyBoundsMinZ = ld.Bounds.MinZ
	l.EnemyBoundsMaxZ = ld.Bounds.MaxZ

	// Obstacles
	l.Obstacles = make([]combat.Obstacle, len(ld.Obstacles))
	for i, o := range ld.Obstacles {
		l.Obstacles[i] = combat.Obstacle{
			CX:     o.Center[0],
			CZ:     o.Center[2],
			HX:     o.HalfExtents[0],
			HZ:     o.HalfExtents[2],
			Height: o.HalfExtents[1] * 2,
		}
	}

	// Elevators
	l.Elevators = make([]ElevatorVolume, len(ld.Elevators))
	for i, e := range ld.Elevators {
		l.Elevators[i] = ElevatorVolume{
			CenterX: e.CenterX,
			CenterZ: e.CenterZ,
			HalfX:   e.HalfX,
			HalfZ:   e.HalfZ,
			BottomY: e.BottomY,
			TopY:    e.TopY,
			Speed:   e.Speed,
		}
	}

	// Player spawns
	l.PlayerSpawns = make([]PlayerSpawn, len(ld.PlayerSpawns))
	for i, s := range ld.PlayerSpawns {
		l.PlayerSpawns[i] = PlayerSpawn{
			Position:  entity.Vec3{X: s.X, Y: s.Y, Z: s.Z},
			Condition: s.Condition,
		}
	}
	l.SpawnYaw = ld.SpawnYaw

	// Enemy spawns
	l.EnemySpawns = make([]EnemySpawnPoint, len(ld.EnemySpawns))
	for i, s := range ld.EnemySpawns {
		esp := EnemySpawnPoint{
			Position:    entity.Vec3{X: s.X, Y: s.Y, Z: s.Z},
			DefName:     s.DefName,
			IsBoss:      s.IsBoss,
			PatrolA:     entity.Vec3{X: s.PatrolA.X, Y: s.PatrolA.Y, Z: s.PatrolA.Z},
			PatrolB:     entity.Vec3{X: s.PatrolB.X, Y: s.PatrolB.Y, Z: s.PatrolB.Z},
			AggroRadius: s.AggroRadius,
			LeashRadius: s.LeashRadius,
			GroupID:     s.GroupID,
			Condition:   s.Condition,
		}
		if len(s.PatrolWaypoints) >= 2 {
			esp.PatrolWaypoints = make([]entity.Vec3, len(s.PatrolWaypoints))
			for j, w := range s.PatrolWaypoints {
				esp.PatrolWaypoints[j] = entity.Vec3{X: w.X, Y: w.Y, Z: w.Z}
			}
			// Override A/B with first/last for backward compat with 2-point patrol
			esp.PatrolA = esp.PatrolWaypoints[0]
			esp.PatrolB = esp.PatrolWaypoints[len(esp.PatrolWaypoints)-1]
		}
		l.EnemySpawns[i] = esp
	}

	// NPC spawns (hub)
	l.NPCSpawns = make([]NPCSpawnPoint, len(ld.NPCSpawns))
	for i, s := range ld.NPCSpawns {
		wps := make([]entity.Vec3, len(s.Waypoints))
		for j, w := range s.Waypoints {
			wps[j] = entity.Vec3{X: w.X, Y: w.Y, Z: w.Z}
		}
		l.NPCSpawns[i] = NPCSpawnPoint{
			DefName:      s.DefName,
			Speed:        s.Speed,
			IdleDuration: s.IdleDuration,
			Waypoints:    wps,
		}
	}

	// Portals
	l.Portals = make([]PortalDef, len(ld.Portals))
	for i, p := range ld.Portals {
		l.Portals[i] = PortalDef{
			Name:              p.Name,
			Position:          entity.Vec3{X: p.X, Y: p.Y, Z: p.Z},
			TargetZone:        p.TargetZone,
			InteractionRadius: p.InteractionRadius,
			Condition:         p.Condition,
		}
	}

	// Zone triggers → map well-known trigger IDs to Level fields
	for _, zt := range ld.ZoneTriggers {
		switch zt.TriggerID {
		case "arena_entry":
			l.ArenaEntryZ = zt.Threshold
		case "boss_room_entry":
			l.BossRoomEntryZ = zt.Threshold
		}
	}

	return nil
}
