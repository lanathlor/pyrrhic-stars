package ability

import (
	"math"
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// --- helpers ---

func newGunner() *entity.Player {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p.RotationY = 0 // facing -Z
	return p
}

func newVanguard() *entity.Player {
	p := entity.NewPlayer(2, entity.ClassVanguard)
	p.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p.RotationY = 0
	return p
}

func newBladeDancer() *entity.Player {
	p := entity.NewPlayer(3, entity.ClassBladeDancer)
	p.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p.RotationY = 0
	p.Config = entity.ConfigOrbit
	return p
}

// enemyInFront places an alive enemy 5 units in front of a player facing -Z.
func enemyInFront(id uint16, hp float32) *entity.Enemy {
	e := entity.NewEnemy(id, hp, "test_mob")
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -5}
	return e
}

// enemyBehind places an alive enemy 5 units behind a player facing -Z.
func enemyBehind(id uint16, hp float32) *entity.Enemy {
	e := entity.NewEnemy(id, hp, "test_mob")
	e.Position = entity.Vec3{X: 0, Y: 0, Z: 5}
	return e
}

func castCtx(p *entity.Player, enemies ...*entity.Enemy) *CastContext {
	return &CastContext{Player: p, Enemies: enemies}
}

func tickCtx(enemies ...*entity.Enemy) *TickContext {
	return &TickContext{Enemies: enemies}
}

// --- Engine setup ---

func TestNewEngine_RegistersAllAbilities(t *testing.T) {
	eng := NewEngine(nil)

	// Spot-check a few from each class
	for _, id := range []string{
		"fire_shot", "overclock", "rechamber", "rechamber_confirm",
		"melee_light", "melee_heavy", "vg_block", "blade_swirl", "ground_slam",
		"bd_melee", "bd_heavy", "bd_guard",
		"shielded_sweep", "cleaving_pierce", "decree_strike", "dodge",
	} {
		if eng.GetAbility(id) == nil {
			t.Errorf("ability %q not registered", id)
		}
	}
}

func TestGetAbility_Unknown(t *testing.T) {
	eng := NewEngine(nil)
	if eng.GetAbility("nonexistent") != nil {
		t.Error("expected nil for unknown ability")
	}
}

// --- Cast validation (table-driven) ---

func TestCast_Validation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Engine) (*entity.Player, []*entity.Enemy)
		ability string
		reason  string
	}{
		{
			name: "unknown ability",
			setup: func(eng *Engine) (*entity.Player, []*entity.Enemy) {
				return newGunner(), nil
			},
			ability: "nonexistent",
			reason:  "unknown ability",
		},
		{
			name: "dead player",
			setup: func(eng *Engine) (*entity.Player, []*entity.Enemy) {
				p := newGunner()
				p.Alive = false
				return p, nil
			},
			ability: "fire_shot",
			reason:  "dead",
		},
		{
			name: "GCD active",
			setup: func(eng *Engine) (*entity.Player, []*entity.Enemy) {
				p := newGunner()
				p.GCDTimer = 0.5
				return p, nil
			},
			ability: "fire_shot",
			reason:  "gcd",
		},
		{
			name: "per-ability cooldown",
			setup: func(eng *Engine) (*entity.Player, []*entity.Enemy) {
				p := newGunner()
				p.Cooldowns["fire_shot"] = 0.1
				return p, nil
			},
			ability: "fire_shot",
			reason:  "cooldown",
		},
		{
			name: "wrong BD config",
			setup: func(eng *Engine) (*entity.Player, []*entity.Enemy) {
				p := newBladeDancer()
				p.Config = entity.ConfigFan // origin is fan, spell needs orbit
				return p, nil
			},
			ability: "shielded_sweep", // origin_config = 0 (orbit)
			reason:  "wrong config",
		},
		{
			name: "insufficient resource",
			setup: func(eng *Engine) (*entity.Player, []*entity.Enemy) {
				p := newVanguard()
				p.Resources["stamina"].Current = 0
				return p, nil
			},
			ability: "ground_slam", // costs 20 stamina
			reason:  "insufficient stamina",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(nil)
			p, enemies := tt.setup(eng)
			result := eng.Cast(tt.ability, &CastContext{Player: p, Enemies: enemies})
			if result.OK {
				t.Error("expected cast to fail")
			}
			if result.Reason != tt.reason {
				t.Errorf("reason = %q, want %q", result.Reason, tt.reason)
			}
		})
	}
}

// --- Cast data-driven abilities ---

func TestCast_FireShot_HitsEnemy(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 200)

	result := eng.Cast("fire_shot", castCtx(p, e))
	if !result.OK {
		t.Fatalf("cast failed: %s", result.Reason)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Events))
	}
	if result.Events[0].TargetID != 100 {
		t.Errorf("targetID = %d, want 100", result.Events[0].TargetID)
	}
	if result.Events[0].Amount != 10 {
		t.Errorf("damage = %f, want 10", result.Events[0].Amount)
	}
	if p.Cooldowns["fire_shot"] == 0 {
		t.Error("fire_shot cooldown not set")
	}
}

func TestCast_FireShot_MissesBehind(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyBehind(100, 200)

	result := eng.Cast("fire_shot", castCtx(p, e))
	if !result.OK {
		t.Fatal("cast should succeed even with no hits")
	}
	if len(result.Events) != 0 {
		t.Errorf("events = %d, want 0 (enemy behind)", len(result.Events))
	}
}

func TestCast_FireShot_ObstacleBlocksLOS(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 200)
	obs := combat.Obstacle{CX: 0, CZ: -2.5, HX: 2, HZ: 0.5, Height: 3}

	result := eng.Cast("fire_shot", &CastContext{
		Player:    p,
		Enemies:   []*entity.Enemy{e},
		Obstacles: []combat.Obstacle{obs},
	})
	if !result.OK {
		t.Fatal("cast should succeed")
	}
	if len(result.Events) != 0 {
		t.Error("expected 0 hits through obstacle")
	}
}

func TestCast_GroundSlam_AoECone(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	// Two enemies in front, one behind
	e1 := enemyInFront(100, 200)
	e2 := enemyInFront(101, 200)
	e2.Position.X = 2
	e3 := enemyBehind(102, 200)

	result := eng.Cast("ground_slam", castCtx(p, e1, e2, e3))
	if !result.OK {
		t.Fatalf("cast failed: %s", result.Reason)
	}
	// e1 and e2 should be hit, e3 behind should not
	hitIDs := map[uint16]bool{}
	for _, ev := range result.Events {
		hitIDs[ev.TargetID] = true
	}
	if !hitIDs[100] {
		t.Error("e1 in front should be hit")
	}
	// e3 behind should not be hit
	if hitIDs[102] {
		t.Error("e3 behind should not be hit")
	}
	// Stamina should be spent (was 100, cost 20)
	if p.GetResource("stamina") != 80 {
		t.Errorf("stamina = %f, want 80", p.GetResource("stamina"))
	}
	// Lockout sets GCDTimer
	if p.GCDTimer != 1.2 {
		t.Errorf("GCDTimer = %f, want 1.2 (lockout)", p.GCDTimer)
	}
}

func TestCast_BDGuard_SelfBuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()

	result := eng.Cast("bd_guard", castCtx(p))
	if !result.OK {
		t.Fatalf("cast failed: %s", result.Reason)
	}
	if !p.HasBuff("guard") {
		t.Error("guard buff not applied")
	}
	// Check DR value
	dr := p.DamageReduction()
	if dr != 0.5 {
		t.Errorf("DR = %f, want 0.5", dr)
	}
}

func TestCast_DamageMult_Applied(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	// Add a 2x damage buff
	p.AddBuff(entity.ActiveBuff{
		ID: "test_dmg", Type: entity.BuffDamageMult, Value: 2.0, Duration: 5.0,
	})

	result := eng.Cast("fire_shot", castCtx(p, e))
	if !result.OK {
		t.Fatal("cast failed")
	}
	if len(result.Events) != 1 {
		t.Fatal("expected 1 event")
	}
	// fire_shot base = 10, * 2.0 buff = 20
	if result.Events[0].Amount != 20 {
		t.Errorf("damage = %f, want 20 (10 base * 2.0 buff)", result.Events[0].Amount)
	}
}

func TestCast_SetsAttackState(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 200)

	eng.Cast("fire_shot", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

// --- BD spell mechanics ---

func TestCast_BDSpell_ConfigTransition(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit // origin = orbit
	e := enemyInFront(100, 500)

	// shielded_sweep: orbit → fan
	result := eng.Cast("shielded_sweep", castCtx(p, e))
	if !result.OK {
		t.Fatalf("cast failed: %s", result.Reason)
	}
	if p.Config != entity.ConfigFan {
		t.Errorf("config = %d, want %d (fan)", p.Config, entity.ConfigFan)
	}
}

func TestCast_BDSpell_ShieldGrant(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 500) // guarded_thrust is hitscan, needs a target

	// guarded_thrust: orbit→lance, shieldHP=8
	result := eng.Cast("guarded_thrust", castCtx(p, e))
	if !result.OK {
		t.Fatalf("cast failed: %s", result.Reason)
	}
	shield := p.GetResource("shield")
	if shield != 8 {
		t.Errorf("shield = %f, want 8", shield)
	}
}

func TestCast_BDSpell_DoTApplied(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigLance // targeted_spread: lance→scatter
	e := enemyInFront(100, 500)

	// targeted_spread is HitHitscan with a DoT
	result := eng.Cast("targeted_spread", castCtx(p, e))
	if !result.OK {
		t.Fatalf("cast failed: %s", result.Reason)
	}
	if len(result.Events) == 0 {
		t.Fatal("hitscan should hit enemy in front")
	}
	if len(p.DoTs) == 0 {
		t.Error("DoT should be applied to hit targets")
	}
	if p.DoTs[0].EnemyID != 100 {
		t.Errorf("DoT enemy = %d, want 100", p.DoTs[0].EnemyID)
	}
}

func TestCast_BDSpell_DRBuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit
	e := enemyInFront(100, 500)

	// shielded_sweep has DR buff (0.85)
	eng.Cast("shielded_sweep", castCtx(p, e))
	if !p.HasBuff("bd_dr") {
		t.Error("DR buff not applied")
	}
	if dr := p.DamageReduction(); math.Abs(float64(dr-0.85)) > 0.01 {
		t.Errorf("DR = %f, want 0.85", dr)
	}
}

func TestCast_BDSpell_GCD(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit

	eng.Cast("shielded_sweep", castCtx(p))
	if p.GCDTimer != 0.5 {
		t.Errorf("GCDTimer = %f, want 0.5", p.GCDTimer)
	}
}

// --- Handler tests ---

func TestRechamber_StartAndConfirm(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Start rechamber
	r := eng.Cast("rechamber", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber start failed: %s", r.Reason)
	}
	state := p.AbilityState["rechamber"].(*RechamberState)
	if state.Phase != 1 {
		t.Errorf("phase = %d, want 1", state.Phase)
	}

	// Can't start again while in progress
	r = eng.Cast("rechamber", castCtx(p))
	if r.OK {
		t.Error("should not be able to start rechamber while in progress")
	}

	// Tick through windup (0.6s)
	eng.TickPlayer(p, 0.6, tickCtx())
	if state.Phase != 2 {
		t.Errorf("after windup phase = %d, want 2", state.Phase)
	}

	// Confirm in timing window
	r = eng.Cast("rechamber_confirm", castCtx(p))
	if !r.OK {
		t.Fatalf("rechamber_confirm failed: %s", r.Reason)
	}
	if !p.HasBuff("rechamber_buff") {
		t.Error("rechamber_buff not applied")
	}
	if state.Phase != 0 {
		t.Errorf("phase after confirm = %d, want 0", state.Phase)
	}
}

func TestRechamber_MissedWindow(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	eng.Cast("rechamber", castCtx(p))
	state := p.AbilityState["rechamber"].(*RechamberState)

	// Tick through windup (0.6s) → phase 2
	eng.TickPlayer(p, 0.6, tickCtx())
	if state.Phase != 2 {
		t.Fatalf("phase = %d, want 2", state.Phase)
	}

	// Tick through timing window (0.35s) → phase 3 (lockout)
	eng.TickPlayer(p, 0.35, tickCtx())
	if state.Phase != 3 {
		t.Errorf("phase = %d, want 3 (lockout)", state.Phase)
	}

	// Confirm should fail during lockout
	r := eng.Cast("rechamber_confirm", castCtx(p))
	if r.OK {
		t.Error("confirm should fail during lockout")
	}

	// Tick through lockout (0.8s) → phase 0
	eng.TickPlayer(p, 0.8, tickCtx())
	if state.Phase != 0 {
		t.Errorf("phase = %d, want 0 (idle)", state.Phase)
	}
}

func TestOverclock_AppliesBuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	r := eng.Cast("overclock", castCtx(p))
	if !r.OK {
		t.Fatalf("overclock failed: %s", r.Reason)
	}
	if !p.HasBuff("overclock") {
		t.Error("overclock buff not applied")
	}
	if p.Cooldowns["overclock"] != 15.0 {
		t.Errorf("cooldown = %f, want 15.0", p.Cooldowns["overclock"])
	}

	// Can't use again while active
	r = eng.Cast("overclock", castCtx(p))
	if r.OK {
		t.Error("should not be able to overclock while active")
	}
}

func TestVGBlock_ParryAndBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	r := eng.Cast("vg_block", castCtx(p))
	if !r.OK {
		t.Fatalf("vg_block failed: %s", r.Reason)
	}
	if !p.HasBuff("vg_parry") {
		t.Error("parry buff not applied")
	}
	if !p.HasBuff("vg_block") {
		t.Error("block buff not applied")
	}
	if p.State != entity.PlayerStateBlock {
		t.Errorf("state = %d, want %d (block)", p.State, entity.PlayerStateBlock)
	}

	// DR should be 0 (parry = full block, overrides the 0.3 multiplicatively → 0)
	dr := p.DamageReduction()
	if dr != 0 {
		t.Errorf("DR = %f, want 0 (parry active)", dr)
	}

	// Can't block again while blocking
	r = eng.Cast("vg_block", castCtx(p))
	if r.OK {
		t.Error("should not be able to block while blocking")
	}
}

func TestMeleeLightVG_Combo(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)

	damages := make([]float32, 3)
	for i := 0; i < 3; i++ {
		hpBefore := e.Health
		p.Cooldowns = make(map[string]float32) // reset cooldown
		r := eng.Cast("melee_light", castCtx(p, e))
		if !r.OK {
			t.Fatalf("combo step %d failed: %s", i, r.Reason)
		}
		if len(r.Events) == 1 {
			damages[i] = r.Events[0].Amount
		} else {
			damages[i] = hpBefore - e.Health
		}
	}

	// Combo: 30, 35, 55
	expected := [3]float32{30, 35, 55}
	for i, want := range expected {
		if damages[i] != want {
			t.Errorf("combo step %d: damage = %f, want %f", i, damages[i], want)
		}
	}
}

func TestMeleeHeavyVG(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	r := eng.Cast("melee_heavy", castCtx(p, e))
	if !r.OK {
		t.Fatalf("melee_heavy failed: %s", r.Reason)
	}
	if p.GetResource("stamina") != 80 {
		t.Errorf("stamina = %f, want 80 (100 - 20)", p.GetResource("stamina"))
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	if r.Events[0].Amount != 45 {
		t.Errorf("damage = %f, want 45", r.Events[0].Amount)
	}
}

func TestBladeSwirl_Handler(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // within 6 radius

	r := eng.Cast("blade_swirl", castCtx(p, e))
	if !r.OK {
		t.Fatalf("blade_swirl failed: %s", r.Reason)
	}
	if !p.HasBuff("blade_swirl") {
		t.Error("blade_swirl DR buff not applied")
	}
	if p.GetResource("stamina") != 75 {
		t.Errorf("stamina = %f, want 75 (100-25)", p.GetResource("stamina"))
	}
	if p.GCDTimer != 1.5 {
		t.Errorf("GCDTimer = %f, want 1.5", p.GCDTimer)
	}
	// Immediate first AoE tick should have hit
	if len(r.Events) == 0 {
		t.Error("expected immediate AoE hit")
	}
}

func TestBladeSwirl_TickDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Cast("blade_swirl", castCtx(p, e))
	hpAfterCast := e.Health

	// Tick 0.5s to trigger second AoE tick
	events := eng.TickPlayer(p, 0.5, tickCtx(e))
	if len(events) == 0 {
		t.Error("expected AoE tick at 0.5s")
	}
	if e.Health >= hpAfterCast {
		t.Error("enemy should take tick damage")
	}
}

// --- TickPlayer tests ---

func TestTickPlayer_CooldownDecrement(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	p.Cooldowns["fire_shot"] = 1.0

	eng.TickPlayer(p, 0.5, tickCtx())
	if cd := p.Cooldowns["fire_shot"]; math.Abs(float64(cd-0.5)) > 0.01 {
		t.Errorf("cooldown = %f, want ~0.5", cd)
	}

	// Tick again to remove
	eng.TickPlayer(p, 0.6, tickCtx())
	if _, ok := p.Cooldowns["fire_shot"]; ok {
		t.Error("cooldown should be removed at 0")
	}
}

func TestTickPlayer_GCDDecrement(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	p.GCDTimer = 0.5

	eng.TickPlayer(p, 0.3, tickCtx())
	if math.Abs(float64(p.GCDTimer-0.2)) > 0.01 {
		t.Errorf("GCDTimer = %f, want ~0.2", p.GCDTimer)
	}

	eng.TickPlayer(p, 0.3, tickCtx())
	if p.GCDTimer != 0 {
		t.Errorf("GCDTimer = %f, want 0", p.GCDTimer)
	}
}

func TestTickPlayer_BuffExpiry(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	p.AddBuff(entity.ActiveBuff{ID: "test", Type: entity.BuffDamageMult, Value: 2.0, Duration: 1.0})

	eng.TickPlayer(p, 0.5, tickCtx())
	if !p.HasBuff("test") {
		t.Error("buff should still be active at 0.5s")
	}

	eng.TickPlayer(p, 0.6, tickCtx())
	if p.HasBuff("test") {
		t.Error("buff should have expired after 1.1s")
	}
}

func TestTickPlayer_ResourceRegen(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 50

	eng.TickPlayer(p, 1.0, tickCtx())
	// Regen = 30/s, so after 1s should be 80
	stam := p.GetResource("stamina")
	if math.Abs(float64(stam-80)) > 0.5 {
		t.Errorf("stamina = %f, want ~80 (50 + 30*1.0)", stam)
	}
}

func TestTickPlayer_ResourceRegenDelay(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.SpendResource("stamina", 50) // sets delay timer
	staminaAfterSpend := p.GetResource("stamina")

	// First tick within delay — no regen
	eng.TickPlayer(p, 0.3, tickCtx())
	if p.GetResource("stamina") != staminaAfterSpend {
		t.Errorf("stamina changed during regen delay")
	}

	// Tick past delay (0.6s total) — regen starts with leftover time
	eng.TickPlayer(p, 0.4, tickCtx())
	if p.GetResource("stamina") <= staminaAfterSpend {
		t.Error("stamina should have started regenerating after delay")
	}
}

func TestTickPlayer_ResourceDecay(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Resources["shield"].Current = 20

	eng.TickPlayer(p, 1.0, tickCtx())
	// Shield regen = -5/s, so after 1s should be 15
	shield := p.GetResource("shield")
	if math.Abs(float64(shield-15)) > 0.5 {
		t.Errorf("shield = %f, want ~15 (20 - 5*1.0)", shield)
	}
}

func TestTickPlayer_ResourceClampsAtZero(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Resources["shield"].Current = 2

	eng.TickPlayer(p, 1.0, tickCtx())
	shield := p.GetResource("shield")
	if shield < 0 {
		t.Errorf("shield = %f, should not go below 0", shield)
	}
}

func TestTickPlayer_ResourceClampsAtMax(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	p.Resources["stamina"].Current = 99

	eng.TickPlayer(p, 1.0, tickCtx())
	stam := p.GetResource("stamina")
	if stam > 100 {
		t.Errorf("stamina = %f, should not exceed max 100", stam)
	}
}

func TestTickPlayer_DoTDealsDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	p.DoTs = append(p.DoTs, entity.ActiveDoT{
		EnemyID:    100,
		SourcePeer: p.PeerID,
		Damage:     10,
		Remaining:  3.0,
		Interval:   1.0,
		TickTimer:  1.0,
	})

	events := eng.TickPlayer(p, 1.0, tickCtx(e))
	if len(events) == 0 {
		t.Error("expected DoT tick damage event")
	}
	if e.Health != 490 {
		t.Errorf("enemy HP = %f, want 490", e.Health)
	}
}

func TestTickPlayer_DoTExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	p.DoTs = append(p.DoTs, entity.ActiveDoT{
		EnemyID:    100,
		SourcePeer: p.PeerID,
		Damage:     10,
		Remaining:  1.0,
		Interval:   0.5,
		TickTimer:  0.5,
	})

	eng.TickPlayer(p, 1.1, tickCtx())
	if len(p.DoTs) != 0 {
		t.Errorf("DoTs = %d, want 0 (expired)", len(p.DoTs))
	}
}

func TestTickPlayer_AttackStateReset(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	eng.Cast("fire_shot", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Fatal("expected attack state after cast")
	}

	// Tick past cooldown (0.18s)
	eng.TickPlayer(p, 0.2, tickCtx())
	if p.State != entity.PlayerStateMove {
		t.Errorf("state = %d, want %d (move) after cooldown expires", p.State, entity.PlayerStateMove)
	}
}

func TestTickPlayer_AttackStatePersistsDuringLockout(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	eng.Cast("ground_slam", castCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Fatal("expected attack state")
	}

	// Tick partway — lockout is 1.2s, cooldown is 8.0s
	eng.TickPlayer(p, 0.5, tickCtx())
	if p.State != entity.PlayerStateAttack {
		t.Error("attack state should persist during lockout")
	}
}

func TestTickPlayer_DeadPlayerSkipped(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	p.Alive = false
	p.Cooldowns["fire_shot"] = 1.0

	eng.TickPlayer(p, 1.0, tickCtx())
	// Cooldowns should not be ticked for dead players
	if p.Cooldowns["fire_shot"] != 1.0 {
		t.Error("dead player should not be ticked")
	}
}

// --- Resolve hit type tests ---

func TestResolveHit_None(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	def := eng.GetAbility("bd_guard") // HitNone
	results := resolveHit(nil, def, p, nil, nil)
	if len(results) != 0 {
		t.Error("HitNone should return empty")
	}
}

func TestResolveHit_AoECircle(t *testing.T) {
	eng := NewEngine(nil)
	_ = eng // engine not needed for resolveHit directly
	p := newVanguard()

	// 3 enemies within radius, 1 outside
	e1 := enemyInFront(1, 200)
	e1.Position = entity.Vec3{X: 2, Y: 0, Z: -2}
	e2 := enemyInFront(2, 200)
	e2.Position = entity.Vec3{X: -2, Y: 0, Z: 2}
	e3 := enemyInFront(3, 200)
	e3.Position = entity.Vec3{X: 0, Y: 0, Z: -3}
	eFar := enemyInFront(4, 200)
	eFar.Position = entity.Vec3{X: 50, Y: 0, Z: 50}

	def := &AbilityDef{
		Hit:        HitDef{Type: HitAoECircle, Radius: 6},
		BaseDamage: 10,
	}
	results := resolveHit(nil, def, p, []*entity.Enemy{e1, e2, e3, eFar}, nil)
	if len(results) != 3 {
		t.Errorf("hits = %d, want 3 (within radius)", len(results))
	}
}

func TestResolveHit_NearestN(t *testing.T) {
	p := newBladeDancer()

	// 4 enemies with threat, ask for 2 nearest
	enemies := make([]*entity.Enemy, 4)
	for i := range enemies {
		enemies[i] = entity.NewEnemy(uint16(i+1), 200, "mob")
		enemies[i].Position = entity.Vec3{Z: float32(-(i + 1) * 3)}
		enemies[i].ThreatTable[p.PeerID] = 10 // in combat
	}

	def := &AbilityDef{
		Hit:        HitDef{Type: HitNearestN, TargetCount: 2},
		BaseDamage: 10,
	}
	results := resolveHit(nil, def, p, enemies, nil)
	if len(results) != 2 {
		t.Errorf("hits = %d, want 2", len(results))
	}
	// Should be the closest two (Z=-3 and Z=-6)
	for _, r := range results {
		if r.TargetID != 1 && r.TargetID != 2 {
			t.Errorf("unexpected target %d — should be the 2 nearest", r.TargetID)
		}
	}
}

func TestResolveHit_NearestN_SkipsNonCombat(t *testing.T) {
	p := newBladeDancer()

	e1 := entity.NewEnemy(1, 200, "mob")
	e1.Position = entity.Vec3{Z: -3}
	e1.ThreatTable[p.PeerID] = 10

	e2 := entity.NewEnemy(2, 200, "mob")
	e2.Position = entity.Vec3{Z: -4}
	// e2 has no threat — not in combat

	def := &AbilityDef{
		Hit:        HitDef{Type: HitNearestN, TargetCount: 5},
		BaseDamage: 10,
	}
	results := resolveHit(nil, def, p, []*entity.Enemy{e1, e2}, nil)
	if len(results) != 1 {
		t.Errorf("hits = %d, want 1 (only in-combat enemies)", len(results))
	}
}

// --- CooldownMult buff ---

func TestCast_CooldownMultBuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 500)

	// Overclock gives cooldown_mult of 0.556
	eng.Cast("overclock", castCtx(p))
	p.Cooldowns = make(map[string]float32) // clear overclock CD for test

	eng.Cast("fire_shot", castCtx(p, e))
	cd := p.Cooldowns["fire_shot"]
	// 0.18 * 0.556 ≈ 0.10
	if math.Abs(float64(cd-0.18*0.556)) > 0.01 {
		t.Errorf("cooldown = %f, want ~%f (0.18 * 0.556)", cd, 0.18*0.556)
	}
}

// --- ApplyThreat helper ---

func TestApplyThreat(t *testing.T) {
	e := enemyInFront(100, 500)
	results := []DamageResult{
		{TargetID: 100, SourceID: 1, Amount: 25, Enemy: e},
		{TargetID: 100, SourceID: 1, Amount: 15, Enemy: e},
	}
	ApplyThreat(results, 1)
	if e.ThreatTable[1] != 40 {
		t.Errorf("threat = %f, want 40", e.ThreatTable[1])
	}
}

// --- Benchmarks ---

func BenchmarkNewEngine(b *testing.B) {
	for b.Loop() {
		NewEngine(nil)
	}
}

func BenchmarkCast_FireShot(b *testing.B) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 1e9) // huge HP so it never dies
	ctx := castCtx(p, e)

	b.ResetTimer()
	for b.Loop() {
		clear(p.Cooldowns)
		p.GCDTimer = 0
		eng.Cast("fire_shot", ctx)
	}
}

func BenchmarkCast_GroundSlam(b *testing.B) {
	eng := NewEngine(nil)
	p := newVanguard()
	enemies := make([]*entity.Enemy, 5)
	for i := range enemies {
		enemies[i] = enemyInFront(uint16(i+1), 1e9)
		enemies[i].Position = entity.Vec3{X: float32(i-2) * 2, Z: -4}
	}
	ctx := castCtx(p, enemies...)

	b.ResetTimer()
	for b.Loop() {
		clear(p.Cooldowns)
		p.GCDTimer = 0
		p.Resources["stamina"].Current = 100
		eng.Cast("ground_slam", ctx)
	}
}

func BenchmarkCast_BDSpell(b *testing.B) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	e := enemyInFront(100, 1e9)
	ctx := castCtx(p, e)

	b.ResetTimer()
	for b.Loop() {
		p.Config = entity.ConfigOrbit
		p.GCDTimer = 0
		clear(p.Cooldowns)
		eng.Cast("shielded_sweep", ctx)
	}
}

func BenchmarkCast_Overclock(b *testing.B) {
	eng := NewEngine(nil)
	p := newGunner()
	ctx := castCtx(p)

	b.ResetTimer()
	for b.Loop() {
		clear(p.Cooldowns)
		p.RemoveBuff("overclock")
		eng.Cast("overclock", ctx)
	}
}

func BenchmarkTickPlayer_Full(b *testing.B) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e9)
	tctx := tickCtx(e)

	// Set up realistic state: cooldowns, buffs, resources, a DoT
	p.Cooldowns["melee_light"] = 0.55
	p.Cooldowns["blade_swirl"] = 10.0
	p.GCDTimer = 0.3
	p.AddBuff(entity.ActiveBuff{ID: "test_buff", Type: entity.BuffDamageMult, Value: 1.5, Duration: 5.0})
	p.Resources["stamina"].Current = 60

	b.ResetTimer()
	for b.Loop() {
		// Reset state each iteration to avoid drift
		p.Alive = true
		p.Cooldowns["melee_light"] = 0.55
		p.Cooldowns["blade_swirl"] = 10.0
		p.GCDTimer = 0.3
		p.Resources["stamina"].Current = 60
		p.Buffs = []entity.ActiveBuff{{ID: "test_buff", Type: entity.BuffDamageMult, Value: 1.5, Duration: 5.0}}
		p.DoTs = []entity.ActiveDoT{{
			EnemyID: 100, SourcePeer: p.PeerID, Damage: 10,
			Remaining: 3.0, Interval: 1.0, TickTimer: 0.05,
		}}
		eng.TickPlayer(p, 0.05, tctx)
	}
}

func BenchmarkTickPlayer_Minimal(b *testing.B) {
	eng := NewEngine(nil)
	p := newGunner()
	tctx := tickCtx()

	b.ResetTimer()
	for b.Loop() {
		p.Alive = true
		eng.TickPlayer(p, 0.05, tctx)
	}
}

func BenchmarkResolveHitscan(b *testing.B) {
	p := newGunner()
	enemies := make([]*entity.Enemy, 10)
	for i := range enemies {
		enemies[i] = entity.NewEnemy(uint16(i+1), 1e9, "mob")
		enemies[i].Position = entity.Vec3{X: float32(i-5) * 3, Z: -float32(i+1) * 5}
	}
	var buf []DamageResult

	b.ResetTimer()
	for b.Loop() {
		buf = resolveHitscan(buf[:0], p, enemies, nil, 10)
	}
}

func BenchmarkResolveAoECircle(b *testing.B) {
	enemies := make([]*entity.Enemy, 20)
	for i := range enemies {
		enemies[i] = entity.NewEnemy(uint16(i+1), 1e9, "mob")
		enemies[i].Position = entity.Vec3{X: float32(i%5) * 2, Z: float32(i/5) * 2}
	}
	center := entity.Vec3{X: 4, Z: 2}
	var buf []DamageResult

	b.ResetTimer()
	for b.Loop() {
		buf = resolveAoECircle(buf[:0], center, 1, enemies, nil, 8, 10)
	}
}
