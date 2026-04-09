package level

import (
	"encoding/json"
	"fmt"
	"os"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

const currentVersion = 2

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

type enemySpawnJSON struct {
	X           float32  `json:"x"`
	Y           float32  `json:"y"`
	Z           float32  `json:"z"`
	DefName     string   `json:"def_name"`
	IsBoss      bool     `json:"is_boss,omitempty"`
	PatrolA     vec3JSON `json:"patrol_a"`
	PatrolB     vec3JSON `json:"patrol_b"`
	AggroRadius float32  `json:"aggro_radius"`
	LeashRadius float32  `json:"leash_radius"`
	GroupID     int      `json:"group_id,omitempty"`
}

type levelDataJSON struct {
	Version      int              `json:"version"`
	Zone         string           `json:"zone"`
	SourceScene  string           `json:"source_scene"`
	Bounds       boundsJSON       `json:"bounds"`
	Obstacles    []obstacleJSON   `json:"obstacles"`
	Elevators    []elevatorJSON   `json:"elevators,omitempty"`
	PlayerSpawns []vec3JSON       `json:"player_spawns"`
	EnemySpawns  []enemySpawnJSON `json:"enemy_spawns,omitempty"`
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
	if ld.Version != currentVersion {
		return fmt.Errorf("level %q: version %d (want %d)", path, ld.Version, currentVersion)
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

	// Spawns
	l.PlayerSpawns = make([]entity.Vec3, len(ld.PlayerSpawns))
	for i, s := range ld.PlayerSpawns {
		l.PlayerSpawns[i] = entity.Vec3{X: s.X, Y: s.Y, Z: s.Z}
	}

	l.EnemySpawns = make([]EnemySpawnPoint, len(ld.EnemySpawns))
	for i, s := range ld.EnemySpawns {
		l.EnemySpawns[i] = EnemySpawnPoint{
			Position:    entity.Vec3{X: s.X, Y: s.Y, Z: s.Z},
			DefName:     s.DefName,
			IsBoss:      s.IsBoss,
			PatrolA:     entity.Vec3{X: s.PatrolA.X, Y: s.PatrolA.Y, Z: s.PatrolA.Z},
			PatrolB:     entity.Vec3{X: s.PatrolB.X, Y: s.PatrolB.Y, Z: s.PatrolB.Z},
			AggroRadius: s.AggroRadius,
			LeashRadius: s.LeashRadius,
			GroupID:     s.GroupID,
		}
	}

	return nil
}
