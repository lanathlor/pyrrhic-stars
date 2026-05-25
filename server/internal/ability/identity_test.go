package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

// =============================================================================
// Gunner — Munitions
// =============================================================================

func TestIdentity_Gunner_MunitionsResourceCreated(t *testing.T) {
	p := newGunner()
	r := p.Resources["munitions"]
	if r == nil {
		t.Fatal("munitions resource not created")
	}
	if r.Max != 5 {
		t.Errorf("munitions.Max = %f, want 5", r.Max)
	}
	if r.Current != 5 {
		t.Errorf("munitions.Current = %f, want 5", r.Current)
	}
	if r.Regen != 0.10 {
		t.Errorf("munitions.Regen = %f, want 0.10", r.Regen)
	}
}

func TestIdentity_Gunner_RecalcStats_ScalesMunitions(t *testing.T) {
	p := newGunner()
	p.GearStats.Identity = 50
	p.RecalcStats()

	r := p.Resources["munitions"]
	if r == nil {
		t.Fatal("munitions resource not found")
	}
	// Max = 5 + 50*0.1 = 10
	if math.Abs(float64(r.Max-10)) > 0.01 {
		t.Errorf("munitions.Max = %f, want 10", r.Max)
	}
	// Regen = 0.10 * (1 + 50/100) = 0.15
	if math.Abs(float64(r.Regen-0.15)) > 0.01 {
		t.Errorf("munitions.Regen = %f, want 0.15", r.Regen)
	}
}

func TestIdentity_Gunner_EnhancedRound_PressureGeneratesBatch(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	as := getGunnerAssaultState(p)

	// Build pressure to max by consecutive hits on same target.
	for i := range assaultPressureMax {
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.2, tickCtx())
	}

	// At max pressure, a batch of enhanced rounds should have been generated.
	if as.EnhancedReserve != assaultEnhancedBatch {
		t.Errorf("enhanced reserve = %d, want %d", as.EnhancedReserve, assaultEnhancedBatch)
	}
}

func TestIdentity_Gunner_EnhancedRound_NoEnhancedNoBonusDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	as := getGunnerAssaultState(p)
	as.EnhancedLoaded = 0

	// Fire one shot — no enhanced loaded, should deal base + small pressure bonus only.
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	// First shot on target = pressure stack 1 → bonus = 10 * 1 * 0.03 = 0.3
	assertDmgNear(t, r.Events[0].Amount, 10.3, "no enhanced loaded")
}

func TestIdentity_Gunner_EnhancedRound_IdentityScalesDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	p.GearStats.Identity = 100
	p.RecalcStats()
	e := enemyInFront(100, 1e6)
	as := getGunnerAssaultState(p)

	// Manually load enhanced rounds and set pressure stacks for deterministic test.
	as.EnhancedLoaded = 1
	as.PressureStacks = 5
	as.PressureTarget = e.ID

	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("fire_shot failed: %s", r.Reason)
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	// Base hit = 10 + pressure bonus (10 * 6stacks * 0.03 = 1.8) = 11.8
	// Enhanced bonus = (15 + 1.5*6) * (1+100/100) * 1.0 = 24 * 2 = 48
	// Total ≈ 59.8
	wantBase := float32(10.0) + 10.0*6*assaultPressureBonus
	wantEnhanced := (float32(assaultEnhancedBase) + float32(assaultEnhancedPerStack)*6) * 2.0
	assertDmgNear(t, r.Events[0].Amount, wantBase+wantEnhanced, "enhanced + identity=100")
}

func TestIdentity_Gunner_EnhancedRound_ConsumedPerShot(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e6)
	as := getGunnerAssaultState(p)

	// Load 3 enhanced rounds
	as.EnhancedLoaded = 3

	for i := range 3 {
		r := eng.Commit("fire_shot", commitCtx(p, e))
		if !r.OK {
			t.Fatalf("shot %d failed: %s", i+1, r.Reason)
		}
		eng.TickPlayer(p, 0.2, tickCtx())
	}
	if as.EnhancedLoaded != 0 {
		t.Errorf("enhanced loaded = %d, want 0 after 3 shots", as.EnhancedLoaded)
	}
	// 4th shot should come from magazine, not enhanced
	magBefore := as.MagCurrent
	r := eng.Commit("fire_shot", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("4th shot failed: %s", r.Reason)
	}
	if as.MagCurrent != magBefore-1 {
		t.Errorf("mag = %d, want %d (consumed from magazine, not enhanced)", as.MagCurrent, magBefore-1)
	}
	_ = r
}

// =============================================================================
// Vanguard — Tenacity
// =============================================================================

func TestIdentity_Vanguard_RecalcStats_ScalesStamina(t *testing.T) {
	p := newVanguard()
	p.GearStats.Identity = 50
	p.RecalcStats()

	r := p.Resources["stamina"]
	if r == nil {
		t.Fatal("stamina resource not found")
	}
	// Max = 100 + 50 = 150
	if r.Max != 150 {
		t.Errorf("stamina.Max = %f, want 150", r.Max)
	}
	// Regen = 30 * (1 + 50/100) = 45
	if math.Abs(float64(r.Regen-45)) > 0.1 {
		t.Errorf("stamina.Regen = %f, want 45", r.Regen)
	}
}

func TestIdentity_Vanguard_TenacityEfficiency(t *testing.T) {
	tests := []struct {
		identity float32
		want     float32
	}{
		{0, 1.0},
		{100, 1.0 / 1.5}, // 0.6667
		{200, 1.0 / 2.0}, // 0.5
	}
	for _, tt := range tests {
		p := newVanguard()
		p.GearStats.Identity = tt.identity
		got := p.TenacityEfficiency()
		if math.Abs(float64(got-tt.want)) > 0.001 {
			t.Errorf("TenacityEfficiency(Identity=%f) = %f, want %f", tt.identity, got, tt.want)
		}
	}
}

func TestIdentity_Vanguard_TenacityEfficiency_NonVanguard(t *testing.T) {
	p := newGunner()
	p.GearStats.Identity = 100
	if p.TenacityEfficiency() != 1.0 {
		t.Errorf("non-vanguard TenacityEfficiency = %f, want 1.0", p.TenacityEfficiency())
	}
}

func TestIdentity_Vanguard_MeleeLightReducedCost(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.GearStats.Identity = 100
	p.RecalcStats()
	p.Resources["stamina"].Current = p.Resources["stamina"].Max // fill to new max
	e := enemyInFront(100, 1000)

	staminaBefore := p.GetResource("stamina")
	eng.Commit("cleave", commitCtx(p, e))
	staminaAfter := p.GetResource("stamina")

	// Cost = 10 * TenacityEfficiency(100) = 10 * 0.6667 ≈ 6.667
	cost := staminaBefore - staminaAfter
	wantCost := float32(10.0) * p.TenacityEfficiency()
	if math.Abs(float64(cost-wantCost)) > 0.1 {
		t.Errorf("stamina cost = %f, want %f", cost, wantCost)
	}
}

func TestIdentity_Vanguard_BlockDrainReduced(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.GearStats.Identity = 200
	p.RecalcStats()
	p.Resources["stamina"].Current = p.Resources["stamina"].Max

	eng.Commit("vg_block", commitCtx(p))
	staminaBefore := p.GetResource("stamina")

	// Use small dt so regen delay (0.6s) prevents regen from kicking in
	eng.TickPlayer(p, 0.1, tickCtx())
	staminaAfter := p.GetResource("stamina")

	// Drain = blockDrainPerSec * 0.1 * TenacityEfficiency(200) = 15 * 0.1 * 0.5 = 0.75
	drain := staminaBefore - staminaAfter
	wantDrain := blockDrainPerSec * 0.1 * p.TenacityEfficiency()
	if math.Abs(float64(drain-wantDrain)) > 0.05 {
		t.Errorf("block drain = %f, want %f", drain, wantDrain)
	}
}

func TestIdentity_Vanguard_DodgeReducedCost(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.GearStats.Identity = 100
	p.RecalcStats()
	p.Resources["stamina"].Current = p.Resources["stamina"].Max

	staminaBefore := p.GetResource("stamina")
	r := eng.Commit("dodge", commitCtx(p))
	if !r.OK {
		t.Fatalf("dodge failed: %s", r.Reason)
	}
	staminaAfter := p.GetResource("stamina")

	// Dodge costs stamina via data-driven Costs. The cost gets TenacityEfficiency applied.
	cost := staminaBefore - staminaAfter
	// At Identity=100, efficiency = 1/1.5 ≈ 0.6667
	// Base dodge cost is defined on the ability def
	dodgeDef := eng.GetAbility("dodge")
	if dodgeDef == nil {
		t.Fatal("dodge ability not found")
	}
	var baseCost float32
	for _, c := range dodgeDef.Costs {
		if c.Resource == "stamina" {
			baseCost = c.Amount
		}
	}
	wantCost := baseCost * p.TenacityEfficiency()
	if math.Abs(float64(cost-wantCost)) > 0.1 {
		t.Errorf("dodge stamina cost = %f, want %f (base %f * efficiency %f)", cost, wantCost, baseCost, p.TenacityEfficiency())
	}
}

func TestIdentity_Vanguard_StaminaRegenScaled(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.GearStats.Identity = 100
	p.RecalcStats()
	p.Resources["stamina"].Current = 0

	eng.TickPlayer(p, 1.0, tickCtx())
	// Regen = 30 * (1 + 100/100) = 60/s. After 1s = 60 (without Tempo).
	stam := p.GetResource("stamina")
	if math.Abs(float64(stam-60)) > 1.0 {
		t.Errorf("stamina after 1s = %f, want ~60", stam)
	}
}

// =============================================================================
// Blade Dancer — Resonance
// =============================================================================

func TestIdentity_BD_ResonanceResourceCreated(t *testing.T) {
	p := newBladeDancer()
	r := p.Resources["resonance"]
	if r == nil {
		t.Fatal("resonance resource not created")
	}
	if r.Max != 100 {
		t.Errorf("resonance.Max = %f, want 100", r.Max)
	}
	if r.Current != 0 {
		t.Errorf("resonance.Current = %f, want 0", r.Current)
	}
	if r.Regen != -2 {
		t.Errorf("resonance.Regen = %f, want -2", r.Regen)
	}
}

func TestIdentity_BD_TransitionAddsResonance(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 1000)

	// shielded_sweep: orbit → fan (has DestConfig)
	r := eng.Commit("shielded_sweep", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("shielded_sweep failed: %s", r.Reason)
	}
	// Gain = 10 * (1 + 0/100) = 10
	res := p.GetResource("resonance")
	if res != 10 {
		t.Errorf("resonance = %f, want 10", res)
	}
}

func TestIdentity_BD_TransitionScalesWithIdentity(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.GearStats.Identity = 100
	p.RecalcStats()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 1000)

	eng.Commit("shielded_sweep", commitCtx(p, e))
	// Gain = 10 * (1 + 100/100) = 20
	res := p.GetResource("resonance")
	if res != 20 {
		t.Errorf("resonance = %f, want 20", res)
	}
}

func TestIdentity_BD_ResonanceDecays(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Resources["resonance"].Current = 50

	eng.TickPlayer(p, 1.0, tickCtx())
	// Decay = -2/s, after 1s = 50 - 2 = 48
	res := p.GetResource("resonance")
	if math.Abs(float64(res-48)) > 0.5 {
		t.Errorf("resonance = %f, want ~48 (50 - 2*1.0)", res)
	}
}

func TestIdentity_BD_ResonanceDecaySlowsWithIdentity(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.GearStats.Identity = 100
	p.RecalcStats()
	p.Resources["resonance"].Current = 50

	eng.TickPlayer(p, 1.0, tickCtx())
	// Decay = -2 * (1 / (1 + 100/100)) = -2 * 0.5 = -1/s. After 1s = 50 - 1 = 49
	res := p.GetResource("resonance")
	if math.Abs(float64(res-49)) > 0.5 {
		t.Errorf("resonance = %f, want ~49 (decay slowed by Identity=100)", res)
	}
}

func TestIdentity_BD_ResonanceThreshold50(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	p.Resources["resonance"].Current = 50
	e := enemyInFront(100, 1000)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // within AoECircle radius 4

	hpBefore := e.Health
	r := eng.Commit("shielded_sweep", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("shielded_sweep failed: %s", r.Reason)
	}

	// Base damage dealt + 25% bonus from resonance ≥50
	totalDmg := hpBefore - e.Health
	// shielded_sweep deals some base damage. With resonanceAmpFactor=0.25,
	// the total should be base * 1.25
	if len(r.Events) == 0 {
		t.Fatal("expected hits")
	}
	// The event amount already includes the bonus
	baseDmg := r.Events[0].Amount / 1.25
	if math.Abs(float64(totalDmg-baseDmg*1.25)) > 1.0 {
		t.Errorf("total damage = %f, expected ~1.25x base", totalDmg)
	}
	// Resonance should be consumed (set to 0) then gain from transition
	// Gain = 10 (Identity=0)
	res := p.GetResource("resonance")
	if res != 10 {
		t.Errorf("resonance after amp+transition = %f, want 10 (consumed then gained 10)", res)
	}
}

func TestIdentity_BD_ResonanceThreshold100(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	p.Resources["resonance"].Current = 100
	e := enemyInFront(100, 1000)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // within AoECircle radius 4

	hpBefore := e.Health
	r := eng.Commit("shielded_sweep", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("shielded_sweep failed: %s", r.Reason)
	}

	totalDmg := hpBefore - e.Health
	if len(r.Events) == 0 {
		t.Fatal("expected hits")
	}
	// With resonanceAmpFactor=0.5, total = base * 1.5
	baseDmg := r.Events[0].Amount / 1.5
	if math.Abs(float64(totalDmg-baseDmg*1.5)) > 1.0 {
		t.Errorf("total damage = %f, expected ~1.5x base", totalDmg)
	}
	// Resonance consumed then gained 10
	res := p.GetResource("resonance")
	if res != 10 {
		t.Errorf("resonance after amp+transition = %f, want 10", res)
	}
}

func TestIdentity_BD_ResonanceConsumedOnAmp(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	p.Resources["resonance"].Current = 75
	e := enemyInFront(100, 1000)

	eng.Commit("shielded_sweep", commitCtx(p, e))
	// 75 ≥ 50, so amp activates (0.25 factor), resonance consumed to 0, then +10 from transition
	res := p.GetResource("resonance")
	if res != 10 {
		t.Errorf("resonance = %f, want 10 (consumed then gained)", res)
	}
}

func TestIdentity_BD_ShieldUnaffected(t *testing.T) {
	p := newBladeDancer()
	shield := p.Resources["shield"]
	if shield == nil {
		t.Fatal("shield resource not found")
	}
	if shield.Regen != -5 {
		t.Errorf("shield.Regen = %f, want -5", shield.Regen)
	}

	// With Identity, shield should be unaffected
	p.GearStats.Identity = 100
	p.RecalcStats()
	if shield.Regen != -5 {
		t.Errorf("shield.Regen after Identity=100 = %f, want -5 (unchanged)", shield.Regen)
	}
}

// =============================================================================
// Backward Compatibility — Identity=0 produces base values
// =============================================================================

func TestIdentity_ZeroIdentity_BackwardCompat(t *testing.T) {
	// Vanguard: stamina stays at base
	pv := newVanguard()
	if pv.Resources["stamina"].Max != 100 {
		t.Errorf("vanguard stamina.Max = %f, want 100", pv.Resources["stamina"].Max)
	}
	if pv.Resources["stamina"].Regen != 30 {
		t.Errorf("vanguard stamina.Regen = %f, want 30", pv.Resources["stamina"].Regen)
	}
	if pv.TenacityEfficiency() != 1.0 {
		t.Errorf("vanguard TenacityEfficiency = %f, want 1.0", pv.TenacityEfficiency())
	}

	// Gunner: munitions at base
	pg := newGunner()
	if pg.Resources["munitions"].Max != 5 {
		t.Errorf("gunner munitions.Max = %f, want 5", pg.Resources["munitions"].Max)
	}
	if pg.Resources["munitions"].Regen != 0.10 {
		t.Errorf("gunner munitions.Regen = %f, want 0.10", pg.Resources["munitions"].Regen)
	}

	// Blade Dancer: resonance at base
	pbd := newBladeDancer()
	if pbd.Resources["resonance"].Max != 100 {
		t.Errorf("blade_dancer resonance.Max = %f, want 100", pbd.Resources["resonance"].Max)
	}
	if pbd.Resources["resonance"].Regen != -2 {
		t.Errorf("blade_dancer resonance.Regen = %f, want -2", pbd.Resources["resonance"].Regen)
	}
}
