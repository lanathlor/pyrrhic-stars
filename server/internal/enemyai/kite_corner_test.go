package enemyai

import (
	"testing"

	"codex-online/server/internal/entity"
)

// TestKiter_EscapesCorner reproduces the live-play bug where the fleeing
// aceras_general backed itself into a corner and stayed pinned: the backpedal
// branch of chase drove straight away from the target with no wall awareness,
// and the zone's position clamp held it on the wall forever. A kiter cornered
// by a player must slide along the wall and recover its kite distance.
func TestKiter_EscapesCorner(t *testing.T) {
	def := DefRegistry["aceras_general"]
	if def == nil {
		t.Fatal("aceras_general not in registry")
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.LeashRadius = 200.0

	// Boss pinned in the min-X/min-Z corner, player closing from the diagonal.
	e.Position = entity.Vec3{X: -19, Y: 0.1, Z: -14}
	p := testPlayer(1, entity.Vec3{X: -15, Y: 0.1, Z: -10})
	p.MaxHealth = 1e6
	p.Health = p.MaxHealth
	e.TargetPlayerID = p.ID
	players := testPlayers(p)

	// Simulate the zone's post-tick clamp that pins the boss on the walls.
	clamp := func() {
		e.Position.X = entity.Clamp(e.Position.X, b.BoundsMinX+def.Radius, b.BoundsMaxX-def.Radius)
		e.Position.Z = entity.Clamp(e.Position.Z, b.BoundsMinZ+def.Radius, b.BoundsMaxZ-def.Radius)
	}

	best := e.Position.Sub(p.Position).Flat().Length()
	for range 400 { // 20s at 20Hz
		b.Tick(0.05, players, nil, noSpawn, nil)
		clamp()
		if d := e.Position.Sub(p.Position).Flat().Length(); d > best {
			best = d
		}
	}

	// preferred_range is 12 with a 20% margin; a healthy kiter recovers well
	// past the cornered distance (~5.7). Pinned-in-corner stays at ~5.7.
	if best < 8.5 {
		t.Errorf("cornered kiter never escaped: max distance from player %.2f, want >= 8.5 (pos %v)", best, e.Position)
	}
	t.Logf("max kite distance recovered: %.2f, final pos %v", best, e.Position)
}
