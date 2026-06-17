package enemyai

import (
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// pillarObstacle is a small free-standing pillar near the origin.
func pillarObstacle() combat.Obstacle {
	return combat.Obstacle{CX: 5, CZ: 0, HX: 0.75, HZ: 0.75, Height: 4}
}

// wallObstacle is a long boundary wall (not pillar-like).
func wallObstacle() combat.Obstacle {
	return combat.Obstacle{CX: 0, CZ: -15, HX: 20, HZ: 0.25, Height: 5}
}

func TestCond_PlayerCampingPillar(t *testing.T) {
	def := simpleMeleeDef()
	e := entity.NewEnemy(1, def.MaxHealth, def.Name)
	e.Position = entity.Vec3{X: 0, Z: 0}

	// Player hugging the pillar at (5,0).
	camper := testPlayer(1, entity.Vec3{X: 6, Z: 0})

	t.Run("camping but boss still landing hits", func(t *testing.T) {
		e.SecsSinceDealtDamage = 2.0 // below threshold
		c := testCtx(def, e, testPlayers(camper))
		c.Obs = []combat.Obstacle{pillarObstacle()}
		if condPlayerCampingPillar(c) {
			t.Error("should not fire: boss landed damage only 2s ago")
		}
	})

	t.Run("camping and boss whiffing long enough", func(t *testing.T) {
		e.SecsSinceDealtDamage = pillarCheeseSeconds
		c := testCtx(def, e, testPlayers(camper))
		c.Obs = []combat.Obstacle{pillarObstacle()}
		if !condPlayerCampingPillar(c) {
			t.Error("should fire: player camping a pillar and boss whiffing >= threshold")
		}
	})

	t.Run("boss whiffing but player in the open", func(t *testing.T) {
		e.SecsSinceDealtDamage = pillarCheeseSeconds * 2
		open := testPlayer(1, entity.Vec3{X: 0, Z: 30}) // far from the pillar
		c := testCtx(def, e, testPlayers(open))
		c.Obs = []combat.Obstacle{pillarObstacle()}
		if condPlayerCampingPillar(c) {
			t.Error("should not fire: nobody is hugging a pillar")
		}
	})

	t.Run("player against a wall does not count as pillar camping", func(t *testing.T) {
		e.SecsSinceDealtDamage = pillarCheeseSeconds * 2
		atWall := testPlayer(1, entity.Vec3{X: 0, Z: -14}) // hugging the wall
		c := testCtx(def, e, testPlayers(atWall))
		c.Obs = []combat.Obstacle{wallObstacle()}
		if condPlayerCampingPillar(c) {
			t.Error("should not fire: a boundary wall is not a pillar")
		}
	})
}
