package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestBDSpells_AllRegistered(t *testing.T) {
	eng := NewEngine(nil)

	abilityIDs := []string{
		IDShieldedSweep, "guarded_thrust", "protected_scatter", "fortified_command",
		IDReapingGuard, IDCleavingPierce, "slashing_spread", "sweeping_hex",
		IDPiercingBarrier, "focused_slash", "targeted_spread", "pinning_strike",
		IDDispersedShield, "rain_of_blades", "converging_strike", "chaos_bind",
		IDCommandingWard, "royal_cleave", IDDecreeStrike, "sovereign_scatter",
	}
	for _, id := range abilityIDs {
		if eng.GetAbility(id) == nil {
			t.Errorf("BD ability %q not registered", id)
		}
	}
}

func TestBDSpells_WrongConfigBlocked(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		ability  string
		wrongCfg int
	}{
		{IDShieldedSweep, entity.ConfigFan},       // needs orbit
		{IDReapingGuard, entity.ConfigOrbit},      // needs fan
		{IDPiercingBarrier, entity.ConfigScatter}, // needs lance
		{IDDispersedShield, entity.ConfigCrown},   // needs scatter
		{IDCommandingWard, entity.ConfigLance},    // needs crown
	}
	for _, tt := range tests {
		t.Run(tt.ability, func(t *testing.T) {
			p := newBladeDancer()
			p.Config = tt.wrongCfg

			r := eng.Commit(tt.ability, commitCtx(p))
			if r.OK {
				t.Error("should fail with wrong config")
			}
			if r.Reason != "wrong config" {
				t.Errorf("reason = %q, want %q", r.Reason, "wrong config")
			}
		})
	}
}

func TestBDSpells_AllSetGCD(t *testing.T) {
	eng := NewEngine(nil)

	// Test one ability from each origin config
	tests := []struct {
		ability string
		config  int
	}{
		{IDShieldedSweep, entity.ConfigOrbit},
		{IDReapingGuard, entity.ConfigFan},
		{IDPiercingBarrier, entity.ConfigLance},
		{IDDispersedShield, entity.ConfigScatter},
		{IDCommandingWard, entity.ConfigCrown},
	}
	for _, tt := range tests {
		t.Run(tt.ability, func(t *testing.T) {
			p := newBladeDancer()
			p.Config = tt.config

			eng.Commit(tt.ability, commitCtx(p))
			if p.GCDTimer != 0.5 {
				t.Errorf("GCDTimer = %f, want 0.5", p.GCDTimer)
			}
		})
	}
}

func TestBDSpells_ShieldCapped(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigScatter
	e := enemyInFront(100, 500)

	// dispersed_shield grants 18 shield
	eng.Commit(IDDispersedShield, commitCtx(p, e))
	shield1 := p.GetResource("shield")
	if shield1 != 18 {
		t.Errorf("shield after first commit = %f, want 18", shield1)
	}

	// Second commit should cap at 25
	p.Config = entity.ConfigOrbit // dispersed_shield dest is orbit
	p.GCDTimer = 0
	// commanding_ward is orbit→orbit(0), grants 20 shield
	p.Config = entity.ConfigCrown
	p.GCDTimer = 0
	eng.Commit(IDCommandingWard, commitCtx(p, e))
	shield2 := p.GetResource("shield")
	if shield2 > 25 {
		t.Errorf("shield = %f, should be capped at 25", shield2)
	}
}

func TestBDSpells_ConfigTransitionPerOrigin(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		ability string
		origin  int
		dest    int
	}{
		{IDShieldedSweep, entity.ConfigOrbit, entity.ConfigFan},
		{IDCleavingPierce, entity.ConfigFan, entity.ConfigLance},
		{"targeted_spread", entity.ConfigLance, entity.ConfigScatter},
		{"rain_of_blades", entity.ConfigScatter, entity.ConfigFan},
		{IDDecreeStrike, entity.ConfigCrown, entity.ConfigLance},
	}
	for _, tt := range tests {
		t.Run(tt.ability, func(t *testing.T) {
			p := newBladeDancer()
			p.Config = tt.origin
			e := enemyInFront(100, 1e6)

			r := eng.Commit(tt.ability, commitCtx(p, e))
			if !r.OK {
				t.Fatalf("commit failed: %s", r.Reason)
			}
			if p.Config != tt.dest {
				t.Errorf("config = %d, want %d", p.Config, tt.dest)
			}
		})
	}
}

func TestBDSpells_HitscanMissesBehind(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigFan
	e := enemyBehind(100, 500)

	// cleaving_pierce is hitscan
	r := eng.Commit(IDCleavingPierce, commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) != 0 {
		t.Error("hitscan should miss enemy behind")
	}
}

func TestBDSpells_AoECircleHitsNearby(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit

	e1 := enemyInFront(100, 500)
	e1.Position = entity.Vec3{X: 2, Y: 0, Z: -2} // within 4 radius
	e2 := enemyInFront(101, 500)
	e2.Position = entity.Vec3{X: 0, Y: 0, Z: 20} // far outside radius

	// shielded_sweep is AoECircle, radius 4
	r := eng.Commit(IDShieldedSweep, commitCtx(p, e1, e2))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	hitIDs := map[uint16]bool{}
	for _, ev := range r.Events {
		hitIDs[ev.TargetID] = true
	}
	if !hitIDs[100] {
		t.Error("nearby enemy should be hit by AoE")
	}
	if hitIDs[101] {
		t.Error("far enemy should not be hit")
	}
}

func TestBDSpells_DRBuffApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 500)

	// protected_scatter has DR 0.9, duration 1.5
	eng.Commit("protected_scatter", commitCtx(p, e))
	if !p.HasBuff("bd_dr") {
		t.Error("DR buff should be applied")
	}
	dr := p.DamageReduction()
	if math.Abs(float64(dr-0.9)) > 0.01 {
		t.Errorf("DR = %f, want 0.9", dr)
	}
}

func TestBDSpells_DoTAppliedAndTicks(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigLance
	e := enemyInFront(100, 500)

	// targeted_spread: hitscan + DoT (15 dmg, 2.0s, 1.0s interval)
	r := eng.Commit("targeted_spread", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(p.DoTs) == 0 {
		t.Fatal("DoT should be applied")
	}

	hpBefore := e.Health
	eng.TickPlayer(p, 1.0, tickCtx(e))
	if e.Health >= hpBefore {
		t.Error("DoT should deal damage on tick")
	}
}

func TestBDSpells_SlowApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 500)

	// fortified_command (Orbit→Crown) applies slow debuff
	r := eng.Commit("fortified_command", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if !e.HasDebuff(entity.DebuffSlow) {
		t.Error("enemy should have slow debuff")
	}
	if got := e.GetDebuffValue(entity.DebuffSlow); got < 0.29 || got > 0.31 {
		t.Errorf("slow value = %f, want ~0.3", got)
	}
}

func TestBDSpells_RootApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigLance
	e := enemyInFront(100, 500)

	// pinning_strike (Lance→Crown) applies root debuff
	r := eng.Commit("pinning_strike", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if !e.HasDebuff(entity.DebuffRoot) {
		t.Error("enemy should have root debuff")
	}
}

func TestBDSpells_VulnerabilityApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigFan
	e := enemyInFront(100, 500)

	// sweeping_hex (Fan→Crown) applies vulnerability debuff
	eng.Commit("sweeping_hex", commitCtx(p, e))
	if !e.HasDebuff(entity.DebuffVulnerability) {
		t.Fatal("enemy should have vulnerability debuff")
	}

	// Subsequent damage should be amplified by 20%
	hpBefore := e.Health
	e.TargetApplyDamage(100)
	dealt := hpBefore - e.Health
	if dealt < 119.9 || dealt > 120.1 {
		t.Errorf("dealt = %f, want ~120 with 20%% vulnerability", dealt)
	}
}

func TestBDSpells_CleavingPierceSplash(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigFan

	primary := enemyInFront(100, 500)
	// Place secondary enemy 2 units from primary (within 3.0 splash radius)
	secondary := entity.NewEnemy(101, 500, "mob2")
	secondary.Position = entity.Vec3{X: 2, Y: 0, Z: -5}

	r := eng.Commit(IDCleavingPierce, commitCtx(p, primary, secondary))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	hitIDs := map[uint16]float32{}
	for _, ev := range r.Events {
		hitIDs[ev.TargetID] += ev.Amount
	}
	if _, ok := hitIDs[100]; !ok {
		t.Error("primary target should be hit")
	}
	if _, ok := hitIDs[101]; !ok {
		t.Error("secondary target should take splash damage")
	}
	if hitIDs[101] >= hitIDs[100] {
		t.Errorf("splash (%f) should be less than primary (%f)", hitIDs[101], hitIDs[100])
	}
}

func TestBDSpells_PiercingBarrierShieldScales(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigLance
	e := enemyInFront(100, 500)

	r := eng.Commit(IDPiercingBarrier, commitCtx(p, e))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}

	shield := p.GetResource("shield")
	if shield < 1.0 {
		t.Error("should have granted shield from damage dealt")
	}
	// BaseDamage 18 * ShieldPerDamage 0.8 = ~14.4
	if shield < 14.0 || shield > 15.0 {
		t.Errorf("shield = %f, want ~14.4 (18 * 0.8)", shield)
	}
}

func TestBDSpells_CCImmunityBuffApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 500)

	eng.Commit("fortified_command", commitCtx(p, e))
	if !p.HasBuff("bd_cc_immune") {
		t.Error("should have CC immunity buff")
	}
}

func TestBDSpells_ThornsBuffApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigScatter

	eng.Commit(IDDispersedShield, commitCtx(p))
	if !p.HasBuff("bd_thorns") {
		t.Error("should have thorns buff")
	}
}

func TestBDSpells_NearestNTargeting(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigScatter

	// chaos_bind: NearestN, targetCount=4, from scatter→crown
	enemies := make([]*entity.Enemy, 6)
	for i := range enemies {
		enemies[i] = entity.NewEnemy(uint16(i+1), 500, "mob")
		enemies[i].Position = entity.Vec3{Z: float32(-(i + 1) * 3)}
		enemies[i].ThreatTable[p.ID] = 10 // must be in combat
	}

	r := eng.Commit("chaos_bind", commitCtx(p, enemies...))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if len(r.Events) > 4 {
		t.Errorf("hits = %d, want at most 4 (NearestN)", len(r.Events))
	}
	if len(r.Events) == 0 {
		t.Error("should hit at least some targets")
	}
}
