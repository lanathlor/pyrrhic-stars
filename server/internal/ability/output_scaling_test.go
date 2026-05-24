package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

// These tests verify that the Output gear stat scales ability damage via
// CommitterDamageMult(). Each test equips gear with Output and asserts that
// the dealt damage exceeds the ability's base damage.

func vanguardWithOutput(output float32) *entity.Player {
	p := newVanguard()
	p.GearStats = entity.GearStats{Output: output}
	p.RecalcStats()
	return p
}

func TestBladeSwirl_OutputScalesCastDamage(t *testing.T) {
	eng := NewEngine(nil)
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	// Commit with 0 Output — should deal base 25 damage
	p0 := newVanguard()
	r0 := eng.Commit("vortex", commitCtx(p0, e))
	if !r0.OK || len(r0.Events) == 0 {
		t.Fatal("base commit failed or missed")
	}
	baseDmg := r0.Events[0].Amount

	// Reset enemy and cooldowns
	e.Health = 1e6

	// Commit with 100 Output — should deal 25 * 2.0 = 50 damage
	p100 := vanguardWithOutput(100)
	r100 := eng.Commit("vortex", commitCtx(p100, e))
	if !r100.OK || len(r100.Events) == 0 {
		t.Fatal("output commit failed or missed")
	}
	scaledDmg := r100.Events[0].Amount

	if scaledDmg <= baseDmg {
		t.Errorf("blade_swirl commit damage not scaled by Output: base=%.1f, with 100 Output=%.1f (want ~%.1f)",
			baseDmg, scaledDmg, baseDmg*2.0)
	}
}

func TestVortex_OutputScalesTickDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := vanguardWithOutput(100) // 2.0x multiplier
	e := enemyInFront(100, 1e6)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Commit("vortex", commitCtx(p, e))
	hpAfterCast := e.Health

	// Standard tier: interval = 0.3s. Tick past it.
	events := eng.TickPlayer(p, 0.35, tickCtx(e))
	if len(events) == 0 {
		t.Fatal("expected tick damage")
	}
	tickDmg := hpAfterCast - e.Health

	// With 100 Output, tick damage should be 25 * 2.0 = 50, not base 25
	if tickDmg < 40 {
		t.Errorf("vortex tick damage not scaled by Output: got %.1f, want ~50 (base 25 * 2.0x)", tickDmg)
	}
}

func TestMeleeLight_OutputScalesDamage(t *testing.T) {
	eng := NewEngine(nil)

	// Base damage (combo step 0 = 30)
	p0 := newVanguard()
	e := enemyInFront(100, 1e6)
	r0 := eng.Commit("cleave", commitCtx(p0, e))
	if !r0.OK || len(r0.Events) == 0 {
		t.Fatal("base commit failed or missed")
	}
	baseDmg := r0.Events[0].Amount

	// With 100 Output (2.0x multiplier)
	e.Health = 1e6
	p100 := vanguardWithOutput(100)
	r100 := eng.Commit("cleave", commitCtx(p100, e))
	if !r100.OK || len(r100.Events) == 0 {
		t.Fatal("output commit failed or missed")
	}
	scaledDmg := r100.Events[0].Amount

	if scaledDmg <= baseDmg {
		t.Errorf("melee_light damage not scaled by Output: base=%.1f, with 100 Output=%.1f (want ~%.1f)",
			baseDmg, scaledDmg, baseDmg*2.0)
	}
}

func TestMeleeHeavy_OutputScalesDamage(t *testing.T) {
	eng := NewEngine(nil)

	// Base damage = 45
	p0 := newVanguard()
	e := enemyInFront(100, 1e6)
	r0 := eng.Commit("upheaval", commitCtx(p0, e))
	if !r0.OK || len(r0.Events) == 0 {
		t.Fatal("base commit failed or missed")
	}
	baseDmg := r0.Events[0].Amount

	// With 100 Output (2.0x multiplier)
	e.Health = 1e6
	p100 := vanguardWithOutput(100)
	p100.Cooldowns["upheaval"] = 0
	r100 := eng.Commit("upheaval", commitCtx(p100, e))
	if !r100.OK || len(r100.Events) == 0 {
		t.Fatal("output commit failed or missed")
	}
	scaledDmg := r100.Events[0].Amount

	if scaledDmg <= baseDmg {
		t.Errorf("melee_heavy damage not scaled by Output: base=%.1f, with 100 Output=%.1f (want ~%.1f)",
			baseDmg, scaledDmg, baseDmg*2.0)
	}
}
