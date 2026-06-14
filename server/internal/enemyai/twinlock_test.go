package enemyai

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
)

// TestTwinLock_FiresAtTwoPlayers verifies a multi_target_count: 2 ranged ability
// spawns its volley toward two distinct players at once, while a single-target
// ability fires only along the committed aim.
func TestTwinLock_FiresAtTwoPlayers(t *testing.T) {
	def := DefRegistry["hallway_ranged"]
	b, e := testBrain(def)
	e.Position = entity.Vec3{}
	e.RangedTargetPos = entity.Vec3{Z: 10} // single-target aim: straight ahead

	left := testPlayer(1, entity.Vec3{X: -6, Z: 6})
	right := testPlayer(2, entity.Vec3{X: 6, Z: 6})
	far := testPlayer(3, entity.Vec3{X: 0, Z: 30})
	for _, p := range []*entity.Player{left, right, far} {
		p.Alive = true
		p.Health = p.MaxHealth
	}
	b.ctx.Players = []*entity.Player{left, right, far}

	var dirs []entity.Vec3
	b.ctx.SpawnFn = func(_, dir entity.Vec3, _, _, _ float32) {
		dirs = append(dirs, dir)
	}
	proj := &ability.ProjectileDef{Count: 1, Speed: 12, Damage: 8, Lifetime: 3}

	// Twin-lock: two volleys toward the two nearest players (left & right, not far).
	dirs = nil
	b.ctx.SpawnProjectiles(ability.AbilityDef{Name: "twin", MultiTargetCount: 2, Projectile: proj})
	if len(dirs) != 2 {
		t.Fatalf("twin-lock should spawn 2 projectiles, got %d", len(dirs))
	}
	if !((dirs[0].X < 0 && dirs[1].X > 0) || (dirs[0].X > 0 && dirs[1].X < 0)) {
		t.Fatalf("twin-lock should fire at two opposite-side players, got dirs %v", dirs)
	}

	// Single-target: one volley along the committed aim (+Z, X≈0).
	dirs = nil
	b.ctx.SpawnProjectiles(ability.AbilityDef{Name: "single", Projectile: proj})
	if len(dirs) != 1 {
		t.Fatalf("single-target should spawn 1 projectile, got %d", len(dirs))
	}
	if dirs[0].Z <= 0.9 {
		t.Fatalf("single-target should fire straight ahead (+Z), got %v", dirs[0])
	}
}
