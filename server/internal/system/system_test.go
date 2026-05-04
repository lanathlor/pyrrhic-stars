package system

import (
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"

	"codex-online/server/internal/enemyai"
)

// ---------------------------------------------------------------------------
// FirstEnemy
// ---------------------------------------------------------------------------

func TestFirstEnemy(t *testing.T) {
	tests := []struct {
		name    string
		enemies []*entity.Enemy
		wantNil bool
		wantID  uint16
	}{
		{
			name:    "nil slice returns nil",
			enemies: nil,
			wantNil: true,
		},
		{
			name:    "empty slice returns nil",
			enemies: []*entity.Enemy{},
			wantNil: true,
		},
		{
			name:    "returns first enemy",
			enemies: []*entity.Enemy{entity.NewEnemy(42, 100, "test"), entity.NewEnemy(43, 200, "test2")},
			wantNil: false,
			wantID:  42,
		},
		{
			name:    "single enemy",
			enemies: []*entity.Enemy{entity.NewEnemy(7, 500, "boss")},
			wantNil: false,
			wantID:  7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &World{Enemies: tc.enemies}
			got := w.FirstEnemy()
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got enemy ID=%d", got.ID)
				}
			} else {
				if got == nil {
					t.Fatal("expected non-nil enemy")
				}
				if got.ID != tc.wantID {
					t.Errorf("ID = %d, want %d", got.ID, tc.wantID)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AggroEnemy
// ---------------------------------------------------------------------------

func TestAggroEnemy(t *testing.T) {
	tests := []struct {
		name           string
		setupEnemy     func() *entity.Enemy
		targetPeerID   uint16
		otherEnemies   []*entity.Enemy
		wantState      entity.EnemyState
		wantTarget     uint16
		wantGroupAggro bool // whether group members should also chase
	}{
		{
			name: "patrol enemy transitions to chase",
			setupEnemy: func() *entity.Enemy {
				e := entity.NewEnemy(0, 200, "test")
				e.State = entity.EnemyPatrol
				return e
			},
			targetPeerID: 1,
			wantState:    entity.EnemyChase,
			wantTarget:   1,
		},
		{
			name: "non-patrol enemy stays unchanged",
			setupEnemy: func() *entity.Enemy {
				e := entity.NewEnemy(0, 200, "test")
				e.State = entity.EnemyChase
				e.TargetPlayerID = 5
				return e
			},
			targetPeerID: 1,
			wantState:    entity.EnemyChase,
			wantTarget:   5, // unchanged
		},
		{
			name: "group aggro wakes patrol members",
			setupEnemy: func() *entity.Enemy {
				e := entity.NewEnemy(0, 200, "test")
				e.State = entity.EnemyPatrol
				e.GroupID = 1
				return e
			},
			targetPeerID: 3,
			otherEnemies: func() []*entity.Enemy {
				e2 := entity.NewEnemy(1, 200, "test")
				e2.State = entity.EnemyPatrol
				e2.GroupID = 1
				e2.Alive = true
				return []*entity.Enemy{e2}
			}(),
			wantState:      entity.EnemyChase,
			wantTarget:     3,
			wantGroupAggro: true,
		},
		{
			name: "group aggro skips different group",
			setupEnemy: func() *entity.Enemy {
				e := entity.NewEnemy(0, 200, "test")
				e.State = entity.EnemyPatrol
				e.GroupID = 1
				return e
			},
			targetPeerID: 3,
			otherEnemies: func() []*entity.Enemy {
				e2 := entity.NewEnemy(1, 200, "test")
				e2.State = entity.EnemyPatrol
				e2.GroupID = 2 // different group
				e2.Alive = true
				return []*entity.Enemy{e2}
			}(),
			wantState:      entity.EnemyChase,
			wantTarget:     3,
			wantGroupAggro: false,
		},
		{
			name: "group aggro skips dead members",
			setupEnemy: func() *entity.Enemy {
				e := entity.NewEnemy(0, 200, "test")
				e.State = entity.EnemyPatrol
				e.GroupID = 1
				return e
			},
			targetPeerID: 3,
			otherEnemies: func() []*entity.Enemy {
				e2 := entity.NewEnemy(1, 200, "test")
				e2.State = entity.EnemyPatrol
				e2.GroupID = 1
				e2.Alive = false // dead
				return []*entity.Enemy{e2}
			}(),
			wantState:      entity.EnemyChase,
			wantTarget:     3,
			wantGroupAggro: false,
		},
		{
			name: "no group ID means no group aggro",
			setupEnemy: func() *entity.Enemy {
				e := entity.NewEnemy(0, 200, "test")
				e.State = entity.EnemyPatrol
				e.GroupID = 0
				return e
			},
			targetPeerID: 1,
			otherEnemies: func() []*entity.Enemy {
				e2 := entity.NewEnemy(1, 200, "test")
				e2.State = entity.EnemyPatrol
				e2.GroupID = 0
				e2.Alive = true
				return []*entity.Enemy{e2}
			}(),
			wantState:      entity.EnemyChase,
			wantTarget:     1,
			wantGroupAggro: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.setupEnemy()
			e.Alive = true
			allEnemies := []*entity.Enemy{e}
			allEnemies = append(allEnemies, tc.otherEnemies...)

			w := &World{Enemies: allEnemies}
			w.AggroEnemy(e, tc.targetPeerID)

			if e.State != tc.wantState {
				t.Errorf("primary enemy state = %d, want %d", e.State, tc.wantState)
			}
			if e.TargetPlayerID != tc.wantTarget {
				t.Errorf("primary enemy target = %d, want %d", e.TargetPlayerID, tc.wantTarget)
			}

			for _, other := range tc.otherEnemies {
				if tc.wantGroupAggro {
					if other.State != entity.EnemyChase {
						t.Errorf("group member state = %d, want EnemyChase", other.State)
					}
					if other.TargetPlayerID != tc.targetPeerID {
						t.Errorf("group member target = %d, want %d", other.TargetPlayerID, tc.targetPeerID)
					}
				} else if other.GroupID != e.GroupID {
					// Different group should stay in patrol, not chase.
					if other.State == entity.EnemyChase {
						t.Error("different-group enemy state = EnemyChase, want patrol")
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SpawnEnemyProjectile
// ---------------------------------------------------------------------------

func TestSpawnEnemyProjectile(t *testing.T) {
	w := &World{
		NextProjID: 0,
	}

	pos := entity.Vec3{X: 1, Y: 2, Z: 3}
	dir := entity.Vec3{X: 0, Y: 0, Z: -1}
	w.SpawnEnemyProjectile(0, pos, dir, 20.0, 15.0, 5.0)

	if len(w.Projectiles) != 1 {
		t.Fatalf("projectile count = %d, want 1", len(w.Projectiles))
	}
	if w.NextProjID != 1 {
		t.Errorf("NextProjID = %d, want 1", w.NextProjID)
	}

	p := w.Projectiles[0]
	if p.OwnerID != 0 {
		t.Errorf("OwnerID = %d, want 0 (enemy-owned)", p.OwnerID)
	}
	if p.EnemyIdx != 0 {
		t.Errorf("EnemyIdx = %d, want 0", p.EnemyIdx)
	}
	if p.Speed != 20.0 {
		t.Errorf("Speed = %f, want 20.0", p.Speed)
	}
	if p.Damage != 15.0 {
		t.Errorf("Damage = %f, want 15.0", p.Damage)
	}
	if p.Lifetime != 5.0 {
		t.Errorf("Lifetime = %f, want 5.0", p.Lifetime)
	}
	if !p.Alive {
		t.Error("projectile should be alive")
	}
}

func TestSpawnEnemyProjectile_IncrementingIDs(t *testing.T) {
	w := &World{NextProjID: 10}

	w.SpawnEnemyProjectile(0, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5)
	w.SpawnEnemyProjectile(1, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5)
	w.SpawnEnemyProjectile(2, entity.Vec3{}, entity.Vec3{X: 1}, 10, 10, 5)

	if len(w.Projectiles) != 3 {
		t.Fatalf("projectile count = %d, want 3", len(w.Projectiles))
	}
	if w.NextProjID != 13 {
		t.Errorf("NextProjID = %d, want 13", w.NextProjID)
	}
	if w.Projectiles[0].ID != 11 {
		t.Errorf("first proj ID = %d, want 11", w.Projectiles[0].ID)
	}
	if w.Projectiles[1].ID != 12 {
		t.Errorf("second proj ID = %d, want 12", w.Projectiles[1].ID)
	}
	if w.Projectiles[2].ID != 13 {
		t.Errorf("third proj ID = %d, want 13", w.Projectiles[2].ID)
	}
}

// ---------------------------------------------------------------------------
// SpawnPlayerProjectile
// ---------------------------------------------------------------------------

func TestSpawnPlayerProjectile(t *testing.T) {
	w := &World{
		NextProjID: 5,
	}

	pos := entity.Vec3{X: 10, Y: 1.6, Z: 5}
	dir := entity.Vec3{X: 0, Y: 0, Z: -1}
	w.SpawnPlayerProjectile(42, pos, dir, 30.0, 20.0, 3.0)

	if len(w.Projectiles) != 1 {
		t.Fatalf("projectile count = %d, want 1", len(w.Projectiles))
	}
	if w.NextProjID != 6 {
		t.Errorf("NextProjID = %d, want 6", w.NextProjID)
	}

	p := w.Projectiles[0]
	if p.OwnerID != 42 {
		t.Errorf("OwnerID = %d, want 42", p.OwnerID)
	}
	if p.EnemyIdx != -1 {
		t.Errorf("EnemyIdx = %d, want -1 (player-owned)", p.EnemyIdx)
	}
	if p.Speed != 30.0 {
		t.Errorf("Speed = %f, want 30.0", p.Speed)
	}
	if p.Damage != 20.0 {
		t.Errorf("Damage = %f, want 20.0", p.Damage)
	}
}

// ---------------------------------------------------------------------------
// AISystem.Tick — basic coverage
// ---------------------------------------------------------------------------

func TestAISystem_SkipsNonFightState(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	e := entity.NewEnemy(0, 200, "test")
	e.State = entity.EnemyChase

	w := &World{
		ZoneType: 1,
		State:    StateLobby, // not fight
		Players:  map[uint16]*entity.Player{1: p},
		Enemies:  []*entity.Enemy{e},
		Level:    level.NewArenaLevel(),
	}

	sys := &AISystem{}
	sys.Tick(w, 0.05) // should be a no-op

	// No damage events should be produced
	if len(w.DamageEvents) != 0 {
		t.Errorf("expected no damage events in lobby, got %d", len(w.DamageEvents))
	}
}

func TestAISystem_SkipsDeadEnemies(_ *testing.T) {
	e := entity.NewEnemy(0, 200, "test")
	e.Alive = false
	e.State = entity.EnemyDead

	w := &World{
		ZoneType: 1,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: entity.NewPlayer(1, entity.ClassGunner)},
		Enemies:  []*entity.Enemy{e},
		Level:    level.NewArenaLevel(),
	}

	sys := &AISystem{}
	// Should not panic with dead enemies and no brains
	sys.Tick(w, 0.05)
}

func TestAISystem_SkipsNilEnemies(_ *testing.T) {
	w := &World{
		ZoneType: 1,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: entity.NewPlayer(1, entity.ClassGunner)},
		Enemies:  []*entity.Enemy{nil},
		Brains:   []enemyai.BrainTicker{}, // brains shorter than enemies
		Level:    level.NewArenaLevel(),
	}

	sys := &AISystem{}
	// Should not panic
	sys.Tick(w, 0.05)
}
