package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
)

func makeAddTestWorld(t testing.TB) (*World, *entity.Enemy) {
	t.Helper()
	owner := entity.NewEnemy(1000, 1500, "guard_captain")
	owner.IsBoss = true
	owner.GroupID = 7
	owner.State = entity.EnemyChase
	owner.Position = entity.Vec3{Y: 0.1}

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{Y: 0.1, Z: 5}

	w := &World{
		ZoneType:      1,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{owner},
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
	}
	return w, owner
}

func TestSpawnAddEnemy(t *testing.T) {
	w, owner := makeAddTestWorld(t)
	w.EnemyHPMult = 2.0
	// Brains parallel to Enemies: give the owner a placeholder slot.
	w.Brains = append(w.Brains, nil)

	ok := spawnAddEnemy(w, "hallway_melee", entity.Vec3{X: 3, Y: 0.1, Z: 3}, owner.ID)
	if !ok {
		t.Fatal("spawnAddEnemy should succeed for a registered def")
	}
	if len(w.Enemies) != 2 || len(w.Brains) != 2 {
		t.Fatalf("Enemies=%d Brains=%d, want parallel slices of 2", len(w.Enemies), len(w.Brains))
	}
	add := w.Enemies[1]
	if add.ID < 1500 || add.ID >= 2000 {
		t.Errorf("add ID = %d, want dynamic range [1500, 2000)", add.ID)
	}
	if !add.Alive || add.State != entity.EnemyChase {
		t.Errorf("add should spawn alive and chasing, alive=%v state=%d", add.Alive, add.State)
	}
	if add.SpawnedBy != owner.ID {
		t.Errorf("SpawnedBy = %d, want %d", add.SpawnedBy, owner.ID)
	}
	if add.GroupID != owner.GroupID {
		t.Errorf("GroupID = %d, want inherited %d", add.GroupID, owner.GroupID)
	}
	if add.LeashRadius != 0 {
		t.Errorf("LeashRadius = %f, want 0 (no leash)", add.LeashRadius)
	}
	// HP scaled by group multiplier, BaseMaxHealth unscaled (hallway_melee = 300).
	if add.BaseMaxHealth != 300 {
		t.Errorf("BaseMaxHealth = %f, want 300 (unscaled)", add.BaseMaxHealth)
	}
	if add.MaxHealth != 600 || add.Health != 600 {
		t.Errorf("MaxHealth=%f Health=%f, want 600 (2x scaled)", add.MaxHealth, add.Health)
	}

	// Second spawn increments the ID.
	spawnAddEnemy(w, "hallway_melee", entity.Vec3{X: -3, Y: 0.1, Z: 3}, owner.ID)
	if w.Enemies[2].ID != add.ID+1 {
		t.Errorf("second add ID = %d, want %d", w.Enemies[2].ID, add.ID+1)
	}
}

func TestSpawnAddEnemy_UnknownDefFails(t *testing.T) {
	w, owner := makeAddTestWorld(t)
	w.Brains = append(w.Brains, nil)
	if spawnAddEnemy(w, "no_such_def", entity.Vec3{}, owner.ID) {
		t.Fatal("spawnAddEnemy should fail for an unknown def")
	}
	if len(w.Enemies) != 1 {
		t.Fatalf("Enemies grew to %d on failed spawn", len(w.Enemies))
	}
}

func TestOwnerDeathSweep_KillsOrphanedAdds(t *testing.T) {
	w, owner := makeAddTestWorld(t)
	w.Brains = append(w.Brains, nil)
	spawnAddEnemy(w, "hallway_melee", entity.Vec3{X: 3, Y: 0.1, Z: 3}, owner.ID)
	add := w.Enemies[1]

	owner.State = entity.EnemyDead
	owner.Alive = false

	sys := &AISystem{}
	sys.Tick(w, 0.05)

	if add.Alive || add.State != entity.EnemyDead {
		t.Errorf("add should die when its owner dies, alive=%v state=%d", add.Alive, add.State)
	}
}

func TestResetAliveEnemies_DespawnsAdds(t *testing.T) {
	w, owner := makeAddTestWorld(t)
	w.Brains = append(w.Brains, nil)
	spawnAddEnemy(w, "hallway_melee", entity.Vec3{X: 3, Y: 0.1, Z: 3}, owner.ID)
	add := w.Enemies[1]

	ResetAliveEnemies(w)

	if add.Alive {
		t.Error("spawned add should despawn (die) on wipe reset")
	}
	// Owner (index 0, has a level spawn) resets to patrol, not death.
	if !owner.Alive || owner.State != entity.EnemyPatrol {
		t.Errorf("owner should reset to patrol, alive=%v state=%d", owner.Alive, owner.State)
	}
}

func TestBossReset_DespawnsAdds(t *testing.T) {
	w, b1, _ := makeTwoBossWorld(t)
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{Y: 0.1, Z: 20} // outside boss 1 room
	w.Players[1] = p

	b1.State = entity.EnemyChase
	b1.Health = 900
	w.GateStates[defaultBossGateID] = true
	w.RebuildObstacles()

	// Pad brains to match the two bosses, then summon an add owned by b1.
	w.Brains = append(w.Brains, nil, nil)
	w.AbilityEngine = ability.NewEngine(nil)
	spawnAddEnemy(w, "hallway_melee", entity.Vec3{Y: 0.1, Z: 2}, b1.ID)
	add := w.Enemies[len(w.Enemies)-1]

	sys := &GameFlowSystem{}
	sys.Tick(w, 0.05)

	if b1.State != entity.EnemyPatrol {
		t.Fatalf("boss 1 should reset, state=%d", b1.State)
	}
	if add.Alive {
		t.Error("boss reset should despawn its adds")
	}
}
