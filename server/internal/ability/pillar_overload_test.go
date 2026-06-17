package ability

import (
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// A pillar between the boss and the player. The player hugs the far side to
// break line of sight (the cheese). pillar_overload must punish through it.
func pillarBetween() combat.Obstacle {
	return combat.Obstacle{CX: 0, CZ: -3, HX: 1, HZ: 1}
}

// TestPillarOverload_HitsHiddenPlayerThroughCover proves the new HitAoEObstacles
// hit type deals damage to a player hiding behind a pillar, where a normal
// boss-centered AoE (HitAoECircle) is blocked by that same pillar.
func TestPillarOverload_HitsHiddenPlayerThroughCover(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	obs := []combat.Obstacle{pillarBetween()}

	// Player tucked just behind the pillar: no LoS from the boss at origin.
	hidden := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -4.4})

	// Sanity: the pillar actually blocks line of sight from boss to player.
	if !combat.SegmentHitsObstacle(e.Position, hidden.Position, obs) {
		t.Fatal("test setup: pillar should block LoS to hidden player")
	}

	// A normal boss-centered AoE is blocked by cover -> no hit.
	circleDef := &AbilityDef{ID: "ground_slam", BaseDamage: 35, Hit: HitDef{Type: HitAoECircle, Radius: 6.5}}
	circleRes := eng.CommitDef(circleDef, &CommitContext{Committer: e, Targets: []entity.Target{hidden}, Obstacles: obs})
	if len(circleRes.Events) != 0 {
		t.Fatalf("HitAoECircle should be blocked by cover, got %d events", len(circleRes.Events))
	}

	// pillar_overload erupts around the pillar and ignores cover -> hit.
	hidden2 := playerTarget(1, entity.Vec3{X: 0, Y: 0.1, Z: -4.4})
	overloadDef := &AbilityDef{ID: "pillar_overload", BaseDamage: 70, Hit: HitDef{Type: HitAoEObstacles, Radius: 6}}
	res := eng.CommitDef(overloadDef, &CommitContext{Committer: e, Targets: []entity.Target{hidden2}, Obstacles: obs})
	if len(res.Events) != 1 {
		t.Fatalf("pillar_overload should hit hidden player through cover, got %d events", len(res.Events))
	}
	if res.Events[0].Amount != 70 {
		t.Errorf("damage = %f, want 70", res.Events[0].Amount)
	}
}

// TestPillarOverload_MissesPlayerAwayFromPillars confirms the blast is anchored
// to cover: a player standing in the open, away from every obstacle, is safe.
func TestPillarOverload_MissesPlayerAwayFromPillars(t *testing.T) {
	eng := NewEngine(nil)
	e := newEnemyCommitter(200, 1000)
	obs := []combat.Obstacle{pillarBetween()}

	exposed := playerTarget(2, entity.Vec3{X: 12, Y: 0.1, Z: 12})
	overloadDef := &AbilityDef{ID: "pillar_overload", BaseDamage: 70, Hit: HitDef{Type: HitAoEObstacles, Radius: 6}}
	res := eng.CommitDef(overloadDef, &CommitContext{Committer: e, Targets: []entity.Target{exposed}, Obstacles: obs})
	if len(res.Events) != 0 {
		t.Fatalf("pillar_overload should miss player away from pillars, got %d events", len(res.Events))
	}
}
