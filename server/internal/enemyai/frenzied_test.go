package enemyai

import (
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/overflux"
)

// TestFrenziedMelee_ClosesAndHits is a regression guard for a bug where the
// frenzied hallway_melee variant added a longer-range cleave (3.0) alongside
// melee_slash (2.5). The mob's melee gate (target_in_melee_range) uses the
// LONGEST melee range, so it would commit melee_slash at distances 2.5-3.0 where
// the swing physically cannot reach — whiffing forever and dealing zero damage.
//
// A single stationary player sits at 2.8 (inside cleave range, outside
// melee_slash range). With one player, players_in_aoe(3) is false so cleave
// never fires, isolating melee_slash. A correct mob must close to within
// melee_slash range and damage the player rather than whiff in the dead zone.
func TestFrenziedMelee_ClosesAndHits(t *testing.T) {
	def := DefRegistry["hallway_melee"]
	if def == nil {
		t.Fatal("hallway_melee not loaded")
	}

	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	b.ApplyOverfluxVariants(overflux.NewState([]overflux.ActiveCondition{
		{ID: overflux.CondFrenzied, Rank: 1},
	}))
	e.TargetPlayerID = 1

	// One stationary player in the melee_slash/cleave dead zone (2.8).
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2.8})
	p.Health = p.MaxHealth
	players := testPlayers(p)

	_, ticks := tickUntil(b, 120, 0.05, players, noSpawn, func() bool {
		return p.Health < p.MaxHealth
	})

	if p.Health >= p.MaxHealth {
		t.Fatalf("frenzied melee mob dealt no damage in %d ticks: it parked at the cleave "+
			"range and whiffed melee_slash (player HP %.0f/%.0f)", ticks, p.Health, p.MaxHealth)
	}
	t.Logf("first hit after %d ticks, player HP=%.0f/%.0f", ticks, p.Health, p.MaxHealth)
}
