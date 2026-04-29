package system

import (
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
)

func makePhysicsWorld() *World {
	return &World{
		ZoneType: 1,
		State:    StateFight,
		Players:  make(map[uint16]*entity.Player),
		Level:    level.NewArenaLevel(),
	}
}

// --- PhysicsSystem ---

func TestPhysicsProjectileMovement(t *testing.T) {
	w := makePhysicsWorld()
	proj := entity.NewProjectile(1, 0, -1,
		entity.Vec3{X: 0, Y: 1.5, Z: 25}, // in hallway, clear of obstacles
		entity.Vec3{X: 0, Z: -1},          // moving -Z
		20, 10, 5.0)
	w.Projectiles = []*entity.Projectile{proj}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	// Should move: 20 * 0.05 = 1.0 units in -Z
	if proj.Position.Z > 24.1 || proj.Position.Z < 23.9 {
		t.Errorf("position.Z = %f, want ~24.0", proj.Position.Z)
	}
	if len(w.Projectiles) != 1 {
		t.Errorf("projectile count = %d, want 1", len(w.Projectiles))
	}
}

func TestPhysicsProjectileExpires(t *testing.T) {
	w := makePhysicsWorld()
	proj := entity.NewProjectile(1, 0, -1,
		entity.Vec3{X: 0, Y: 1.5, Z: 25},
		entity.Vec3{X: 0, Z: -1},
		20, 10, 0.04) // very short lifetime
	w.Projectiles = []*entity.Projectile{proj}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	if len(w.Projectiles) != 0 {
		t.Errorf("projectile count = %d, want 0 (expired)", len(w.Projectiles))
	}
}

func TestPhysicsProjectileObstacleCollision(t *testing.T) {
	w := makePhysicsWorld()
	// Place projectile right at an obstacle
	// Boss room pillar at (-8, -6) with half-extents 0.75
	proj := entity.NewProjectile(1, 0, -1,
		entity.Vec3{X: -8, Y: 0.5, Z: -6},
		entity.Vec3{X: 1, Z: 0},
		20, 10, 5.0)
	w.Projectiles = []*entity.Projectile{proj}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	if len(w.Projectiles) != 0 {
		t.Errorf("projectile count = %d, want 0 (hit obstacle)", len(w.Projectiles))
	}
}

func TestPhysicsEnemyProjectileHitsPlayer(t *testing.T) {
	w := makePhysicsWorld()
	p := entity.NewPlayer(1, "gunner")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 25}
	w.Players[1] = p

	e := entity.NewEnemy(0, 1000, "test")
	e.Alive = true
	w.Enemies = []*entity.Enemy{e}

	// Enemy projectile heading toward player
	proj := entity.NewProjectile(1, 0, 0, // ownerID=0 → enemy projectile, enemyIdx=0
		entity.Vec3{X: 0, Y: 1.1, Z: 25}, // right at player
		entity.Vec3{X: 0, Z: -1},
		20, 25, 5.0)
	w.Projectiles = []*entity.Projectile{proj}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	// Should hit player and create damage event
	if p.Health == p.MaxHealth {
		t.Error("player should have taken damage")
	}
	if len(w.DamageEvents) == 0 {
		t.Error("expected at least one damage event")
	}
	if len(w.Projectiles) != 0 {
		t.Errorf("projectile should be consumed, count = %d", len(w.Projectiles))
	}
	// Threat should be added to enemy
	if !e.HasThreat(1) {
		t.Error("enemy should have threat from hit player")
	}
}

func TestPhysicsEnemyProjectileSkipsDeadPlayer(t *testing.T) {
	w := makePhysicsWorld()
	p := entity.NewPlayer(1, "gunner")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 25}
	p.Alive = false
	w.Players[1] = p

	proj := entity.NewProjectile(1, 0, 0,
		entity.Vec3{X: 0, Y: 1.1, Z: 25},
		entity.Vec3{X: 0, Z: -1},
		20, 25, 5.0)
	w.Projectiles = []*entity.Projectile{proj}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	if len(w.DamageEvents) != 0 {
		t.Error("dead player should not be hit")
	}
}

func TestPhysicsNotInFightState(t *testing.T) {
	w := makePhysicsWorld()
	w.State = StateLobby
	proj := entity.NewProjectile(1, 0, -1,
		entity.Vec3{X: 0, Y: 1.5, Z: 25},
		entity.Vec3{X: 0, Z: -1},
		20, 10, 5.0)
	w.Projectiles = []*entity.Projectile{proj}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	// Projectile should not be processed
	if proj.Position.Z < 24.9 {
		t.Error("physics should not run outside fight state")
	}
}

func TestPhysicsMultipleProjectiles(t *testing.T) {
	w := makePhysicsWorld()
	proj1 := entity.NewProjectile(1, 0, -1,
		entity.Vec3{X: 0, Y: 1.5, Z: 25},
		entity.Vec3{X: 0, Z: -1},
		20, 10, 5.0)
	proj2 := entity.NewProjectile(2, 0, -1,
		entity.Vec3{X: 5, Y: 1.5, Z: 25},
		entity.Vec3{X: 0, Z: -1},
		20, 10, 0.01) // expires immediately
	proj3 := entity.NewProjectile(3, 0, -1,
		entity.Vec3{X: -5, Y: 1.5, Z: 25},
		entity.Vec3{X: 0, Z: -1},
		20, 10, 5.0)
	w.Projectiles = []*entity.Projectile{proj1, proj2, proj3}

	sys := PhysicsSystem{}
	sys.Tick(w, 0.05)

	if len(w.Projectiles) != 2 {
		t.Errorf("projectile count = %d, want 2 (one expired)", len(w.Projectiles))
	}
}

// --- AISystem helpers ---

func TestPlayersOnSameSide(t *testing.T) {
	p1 := entity.NewPlayer(1, "gunner")
	p1.Position = entity.Vec3{Z: 5} // Z < 12 → boss room
	p2 := entity.NewPlayer(2, "vanguard")
	p2.Position = entity.Vec3{Z: 20} // Z > 12 → hallway

	players := map[uint16]*entity.Player{1: p1, 2: p2}

	// Enemy in boss room (Z=0 < 12)
	bossRoom := playersOnSameSide(players, 0, 12)
	if len(bossRoom) != 1 {
		t.Errorf("boss room players = %d, want 1", len(bossRoom))
	}
	if _, ok := bossRoom[1]; !ok {
		t.Error("player 1 should be in boss room")
	}

	// Enemy in hallway (Z=20 > 12)
	hallway := playersOnSameSide(players, 20, 12)
	if len(hallway) != 1 {
		t.Errorf("hallway players = %d, want 1", len(hallway))
	}
	if _, ok := hallway[2]; !ok {
		t.Error("player 2 should be in hallway")
	}
}

func TestPropagateGroupAggro(t *testing.T) {
	w := makePhysicsWorld()

	e1 := entity.NewEnemy(1, 200, "hallway_melee")
	e1.GroupID = 1
	e1.State = entity.EnemyChase // aggroed
	e1.TargetPlayerID = 42

	e2 := entity.NewEnemy(2, 200, "hallway_melee")
	e2.GroupID = 1
	e2.State = entity.EnemyPatrol // still patrolling

	e3 := entity.NewEnemy(3, 200, "hallway_melee")
	e3.GroupID = 2 // different group
	e3.State = entity.EnemyPatrol

	w.Enemies = []*entity.Enemy{e1, e2, e3}

	propagateGroupAggro(w)

	if e2.State != entity.EnemyChase {
		t.Errorf("e2 state = %d, want EnemyChase (group aggro)", e2.State)
	}
	if e2.TargetPlayerID != 42 {
		t.Errorf("e2 target = %d, want 42", e2.TargetPlayerID)
	}
	if e3.State != entity.EnemyPatrol {
		t.Errorf("e3 state = %d, want EnemyPatrol (different group)", e3.State)
	}
}

func TestPropagateGroupAggroNoGroup(t *testing.T) {
	w := makePhysicsWorld()

	e1 := entity.NewEnemy(1, 200, "test")
	e1.GroupID = 0 // no group
	e1.State = entity.EnemyChase

	e2 := entity.NewEnemy(2, 200, "test")
	e2.GroupID = 0
	e2.State = entity.EnemyPatrol

	w.Enemies = []*entity.Enemy{e1, e2}

	propagateGroupAggro(w)

	if e2.State != entity.EnemyPatrol {
		t.Error("e2 should remain patrol (GroupID=0 means no group)")
	}
}

// --- Spatial helpers ---

func TestPushOutOfObstacles(t *testing.T) {
	pos := entity.Vec3{X: 5.5, Z: 0}
	obs := []combat.Obstacle{{CX: 5, CZ: 0, HX: 1, HZ: 1}}
	combat.PushOutOfObstacles(&pos, obs, 0.5)
	if pos.X < 6.4 {
		t.Errorf("pos.X = %f, should be pushed out to >= 6.5", pos.X)
	}
}

func TestIsAtWall(t *testing.T) {
	// Within 0.5 margin of minX wall (-20)
	if !combat.IsAtWall(entity.Vec3{X: -19.6, Z: 25}, -20, 20, -15, 50) {
		t.Error("should be at wall near minX")
	}
	// Within 0.5 margin of maxZ wall (50)
	if !combat.IsAtWall(entity.Vec3{X: 0, Z: 49.6}, -20, 20, -15, 50) {
		t.Error("should be at wall near maxZ")
	}
	if combat.IsAtWall(entity.Vec3{X: 0, Z: 25}, -20, 20, -15, 50) {
		t.Error("should not be at wall in center")
	}
}

func TestRotateVecY(t *testing.T) {
	v := entity.Vec3{X: 0, Z: -1} // forward
	rotated := combat.RotateVecY(v, 3.14159/2)
	// 90° rotation should give roughly (-1, 0, 0) or (1, 0, 0) depending on direction
	if rotated.Length() < 0.99 || rotated.Length() > 1.01 {
		t.Errorf("rotated length = %f, want ~1.0", rotated.Length())
	}
}
