package level

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

const currentVersion = 6

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
	DefName      string            `json:"def_name"`
	Speed        float32           `json:"speed"`
	IdleDuration float32           `json:"idle_duration"`
	Waypoints    []vec3JSON        `json:"waypoints"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type gateJSON struct {
	Name          string     `json:"name"`
	GateID        string     `json:"gate_id"`
	Center        [3]float32 `json:"center"`
	HalfExtents   [3]float32 `json:"half_extents"`
	CloseOn       []string   `json:"close_on,omitempty"`
	OpenOn        []string   `json:"open_on,omitempty"`
	DefaultClosed bool       `json:"default_closed,omitempty"`
	PushAxis      string     `json:"push_axis,omitempty"`
	PushOffset    float32    `json:"push_offset,omitempty"`
}

type navmeshJSON struct {
	Vertices [][]float32 `json:"vertices"`
	Polygons [][]int     `json:"polygons"`
}

type levelDataJSON struct {
	Version      int               `json:"version"`
	Zone         string            `json:"zone"`
	ZoneType     string            `json:"zone_type,omitempty"`
	EnemyRadius  float32           `json:"enemy_radius,omitempty"`
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
	Gates        []gateJSON        `json:"gates,omitempty"`
	NavmeshData  *navmeshJSON      `json:"navmesh,omitempty"`
}

// loadLevelData reads a JSON level file and populates l.
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

	loadBoundsAndGeometry(l, &ld)
	loadNavmesh(l, ld.NavmeshData)

	// Player spawns
	l.PlayerSpawns = make([]PlayerSpawn, len(ld.PlayerSpawns))
	for i, s := range ld.PlayerSpawns {
		l.PlayerSpawns[i] = PlayerSpawn{
			Position:  entity.Vec3{X: s.X, Y: s.Y, Z: s.Z},
			Condition: s.Condition,
		}
	}
	l.SpawnYaw = ld.SpawnYaw

	loadEnemySpawns(l, ld.EnemySpawns)
	loadNPCSpawns(l, ld.NPCSpawns)
	loadPortals(l, ld.Portals)
	loadZoneTriggers(l, ld.ZoneTriggers)
	loadGates(l, ld.Gates)

	// v4 fields
	if ld.ZoneType != "" {
		l.ZoneType = ld.ZoneType
	}
	if ld.EnemyRadius > 0 {
		l.EnemyRadius = ld.EnemyRadius
	}

	return nil
}

func loadBoundsAndGeometry(l *Level, ld *levelDataJSON) {
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

	l.Obstacles = make([]combat.Obstacle, len(ld.Obstacles))
	for i, o := range ld.Obstacles {
		l.Obstacles[i] = combat.Obstacle{
			CX:     o.Center[0],
			CZ:     o.Center[2],
			HX:     o.HalfExtents[0],
			HZ:     o.HalfExtents[2],
			BaseY:  o.Center[1] - o.HalfExtents[1],
			Height: o.HalfExtents[1] * 2,
		}
	}

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
}

func loadNPCSpawns(l *Level, spawns []npcSpawnJSON) {
	l.NPCSpawns = make([]NPCSpawnPoint, len(spawns))
	for i, s := range spawns {
		wps := make([]entity.Vec3, len(s.Waypoints))
		for j, w := range s.Waypoints {
			wps[j] = entity.Vec3{X: w.X, Y: w.Y, Z: w.Z}
		}
		l.NPCSpawns[i] = NPCSpawnPoint{
			DefName:      s.DefName,
			Speed:        s.Speed,
			IdleDuration: s.IdleDuration,
			Waypoints:    wps,
			Metadata:     s.Metadata,
		}
	}
}

func loadPortals(l *Level, portals []portalJSON) {
	l.Portals = make([]PortalDef, len(portals))
	for i, p := range portals {
		l.Portals[i] = PortalDef{
			Name:              p.Name,
			Position:          entity.Vec3{X: p.X, Y: p.Y, Z: p.Z},
			TargetZone:        p.TargetZone,
			InteractionRadius: p.InteractionRadius,
			Condition:         p.Condition,
		}
	}
}

func loadZoneTriggers(l *Level, triggers []zoneTriggerJSON) {
	for _, zt := range triggers {
		if zt.TriggerID == "instance_entry" {
			l.InstanceEntryZ = zt.Threshold
		}
	}
}

func loadGates(l *Level, gates []gateJSON) {
	l.Gates = make([]GateDef, len(gates))
	for i, g := range gates {
		l.Gates[i] = GateDef{
			ID: g.GateID,
			Position: entity.Vec3{
				X: g.Center[0],
				Y: g.Center[1],
				Z: g.Center[2],
			},
			HalfExtents: entity.Vec3{
				X: g.HalfExtents[0],
				Y: g.HalfExtents[1],
				Z: g.HalfExtents[2],
			},
			CloseOn:       g.CloseOn,
			OpenOn:        g.OpenOn,
			DefaultClosed: g.DefaultClosed,
			PushAxis:      g.PushAxis,
			PushOffset:    g.PushOffset,
		}
	}
}

// Load reads a zone definition from shared/levels/<zoneName>.json.
// Returns an error if the file is missing — there is no fallback.
func Load(zoneName string) (*Level, error) {
	l := &Level{}
	path := levelDataPath(zoneName)
	if err := loadLevelData(path, l); err != nil {
		return nil, fmt.Errorf("level.Load(%q): %w", zoneName, err)
	}
	if l.ZoneType == "" {
		l.ZoneType = "open_world"
	}
	return l, nil
}

// levelDataPath returns the path to a level JSON file.
// Checks CODEX_LEVELS_DIR env var first, then walks up from CWD looking for shared/levels/.
func levelDataPath(zone string) string {
	dir := os.Getenv("CODEX_LEVELS_DIR")
	if dir != "" {
		return filepath.Join(dir, zone+".json")
	}
	cwd, _ := os.Getwd()
	for d := cwd; d != "/" && d != "."; d = filepath.Dir(d) {
		candidate := filepath.Join(d, "shared", "levels", zone+".json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", "shared", "levels", zone+".json")
}

func loadNavmesh(l *Level, nm *navmeshJSON) {
	if nm == nil || len(nm.Vertices) == 0 || len(nm.Polygons) == 0 {
		return
	}
	verts := make([]entity.Vec3, len(nm.Vertices))
	for i, v := range nm.Vertices {
		if len(v) < 3 {
			continue
		}
		verts[i] = entity.Vec3{X: v[0], Y: v[1], Z: v[2]}
	}
	l.Navmesh = buildNavmesh(verts, nm.Polygons)
}

func loadEnemySpawns(l *Level, spawns []enemySpawnJSON) {
	l.EnemySpawns = make([]EnemySpawnPoint, len(spawns))
	for i, s := range spawns {
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
			esp.PatrolA = esp.PatrolWaypoints[0]
			esp.PatrolB = esp.PatrolWaypoints[len(esp.PatrolWaypoints)-1]
		}
		l.EnemySpawns[i] = esp
	}
}
