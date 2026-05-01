package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestBDSpells_AllRegistered(t *testing.T) {
	eng := NewEngine(nil)

	spellIDs := []string{
		"shielded_sweep", "guarded_thrust", "protected_scatter", "fortified_command",
		"reaping_guard", "cleaving_pierce", "slashing_spread", "sweeping_hex",
		"piercing_barrier", "focused_slash", "targeted_spread", "pinning_strike",
		"dispersed_shield", "rain_of_blades", "converging_strike", "chaos_bind",
		"commanding_ward", "royal_cleave", "decree_strike", "sovereign_scatter",
	}
	for _, id := range spellIDs {
		if eng.GetAbility(id) == nil {
			t.Errorf("BD spell %q not registered", id)
		}
	}
}

func TestBDSpells_WrongConfigBlocked(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		spell     string
		wrongCfg  int
	}{
		{"shielded_sweep", entity.ConfigFan},      // needs orbit
		{"reaping_guard", entity.ConfigOrbit},      // needs fan
		{"piercing_barrier", entity.ConfigScatter}, // needs lance
		{"dispersed_shield", entity.ConfigCrown},   // needs scatter
		{"commanding_ward", entity.ConfigLance},    // needs crown
	}
	for _, tt := range tests {
		t.Run(tt.spell, func(t *testing.T) {
			p := newBladeDancer()
			p.Config = tt.wrongCfg

			r := eng.Cast(tt.spell, castCtx(p))
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

	// Test one spell from each origin config
	tests := []struct {
		spell  string
		config int
	}{
		{"shielded_sweep", entity.ConfigOrbit},
		{"reaping_guard", entity.ConfigFan},
		{"piercing_barrier", entity.ConfigLance},
		{"dispersed_shield", entity.ConfigScatter},
		{"commanding_ward", entity.ConfigCrown},
	}
	for _, tt := range tests {
		t.Run(tt.spell, func(t *testing.T) {
			p := newBladeDancer()
			p.Config = tt.config

			eng.Cast(tt.spell, castCtx(p))
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
	eng.Cast("dispersed_shield", castCtx(p, e))
	shield1 := p.GetResource("shield")
	if shield1 != 18 {
		t.Errorf("shield after first cast = %f, want 18", shield1)
	}

	// Second cast should cap at 25
	p.Config = entity.ConfigOrbit // dispersed_shield dest is orbit
	p.GCDTimer = 0
	// commanding_ward is orbit→orbit(0), grants 20 shield
	p.Config = entity.ConfigCrown
	p.GCDTimer = 0
	eng.Cast("commanding_ward", castCtx(p, e))
	shield2 := p.GetResource("shield")
	if shield2 > 25 {
		t.Errorf("shield = %f, should be capped at 25", shield2)
	}
}

func TestBDSpells_ConfigTransitionPerOrigin(t *testing.T) {
	eng := NewEngine(nil)

	tests := []struct {
		spell  string
		origin int
		dest   int
	}{
		{"shielded_sweep", entity.ConfigOrbit, entity.ConfigFan},
		{"cleaving_pierce", entity.ConfigFan, entity.ConfigLance},
		{"targeted_spread", entity.ConfigLance, entity.ConfigScatter},
		{"rain_of_blades", entity.ConfigScatter, entity.ConfigFan},
		{"decree_strike", entity.ConfigCrown, entity.ConfigLance},
	}
	for _, tt := range tests {
		t.Run(tt.spell, func(t *testing.T) {
			p := newBladeDancer()
			p.Config = tt.origin
			e := enemyInFront(100, 1e6)

			r := eng.Cast(tt.spell, castCtx(p, e))
			if !r.OK {
				t.Fatalf("cast failed: %s", r.Reason)
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
	r := eng.Cast("cleaving_pierce", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
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
	r := eng.Cast("shielded_sweep", castCtx(p, e1, e2))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
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
	eng.Cast("protected_scatter", castCtx(p, e))
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
	r := eng.Cast("targeted_spread", castCtx(p, e))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
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

func TestBDSpells_NearestNTargeting(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigScatter

	// chaos_bind: NearestN, targetCount=4, from scatter→crown
	enemies := make([]*entity.Enemy, 6)
	for i := range enemies {
		enemies[i] = entity.NewEnemy(uint16(i+1), 500, "mob")
		enemies[i].Position = entity.Vec3{Z: float32(-(i + 1) * 3)}
		enemies[i].ThreatTable[p.PeerID] = 10 // must be in combat
	}

	r := eng.Cast("chaos_bind", castCtx(p, enemies...))
	if !r.OK {
		t.Fatalf("cast failed: %s", r.Reason)
	}
	if len(r.Events) > 4 {
		t.Errorf("hits = %d, want at most 4 (NearestN)", len(r.Events))
	}
	if len(r.Events) == 0 {
		t.Error("should hit at least some targets")
	}
}
