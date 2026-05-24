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

func commitCtx(p *entity.Player, enemies ...*entity.Enemy) *CommitContext {
	targets := make([]entity.Target, len(enemies))
	for i, e := range enemies {
		targets[i] = e
	}
	return &CommitContext{Committer: p, Targets: targets}
}

func tickCtx(enemies ...*entity.Enemy) *TickContext {
	targets := make([]entity.Target, len(enemies))
	for i, e := range enemies {
		targets[i] = e
	}
	return &TickContext{Targets: targets}
}

// --- Engine setup ---

func TestNewEngine_RegistersAllAbilities(t *testing.T) {
	eng := NewEngine(nil)

	// Spot-check a few from each class
	for _, id := range []string{
		"fire_shot", "overclock", "rechamber", "rechamber_confirm",
		"reload", "load_enhanced", "mag_dump",
		"cleave", "upheaval", "vg_block", "vg_block_stop", "vortex", "execution",
		"bd_guard",
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

// --- Commit validation (table-driven) ---

func TestCast_Validation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Engine) (*entity.Player, []*entity.Enemy)
		ability string
		reason  string
	}{
		{
			name: "unknown ability",
			setup: func(_ *Engine) (*entity.Player, []*entity.Enemy) {
				return newGunner(), nil
			},
			ability: "nonexistent",
			reason:  "unknown ability",
		},
		{
			name: "dead player",
			setup: func(_ *Engine) (*entity.Player, []*entity.Enemy) {
				p := newGunner()
				p.Alive = false
				return p, nil
			},
			ability: "fire_shot",
			reason:  "dead",
		},
		{
			name: "GCD active",
			setup: func(_ *Engine) (*entity.Player, []*entity.Enemy) {
				p := newGunner()
				p.GCDTimer = 0.5
				return p, nil
			},
			ability: "fire_shot",
			reason:  "gcd",
		},
		{
			name: "per-ability cooldown",
			setup: func(_ *Engine) (*entity.Player, []*entity.Enemy) {
				p := newGunner()
				p.Cooldowns["fire_shot"] = 0.1
				return p, nil
			},
			ability: "fire_shot",
			reason:  "cooldown",
		},
		{
			name: "wrong BD config",
			setup: func(_ *Engine) (*entity.Player, []*entity.Enemy) {
				p := newBladeDancer()
				p.Config = entity.ConfigFan // origin is fan, ability needs orbit
				return p, nil
			},
			ability: "shielded_sweep", // origin_config = 0 (orbit)
			reason:  "wrong config",
		},
		{
			name: "insufficient resource",
			setup: func(_ *Engine) (*entity.Player, []*entity.Enemy) {
				p := newVanguard()
				p.Resources["stamina"].Current = 0
				return p, nil
			},
			ability: "execution", // costs 20 stamina
			reason:  ReasonInsufficientStamina,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(nil)
			p, enemies := tt.setup(eng)
			result := eng.Commit(tt.ability, commitCtx(p, enemies...))
			if result.OK {
				t.Error("expected commit to fail")
			}
			if result.Reason != tt.reason {
				t.Errorf("reason = %q, want %q", result.Reason, tt.reason)
			}
		})
	}
}

// --- Commit data-driven abilities ---

func TestCast_FireShot_HitsEnemy(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 200)

	result := eng.Commit("fire_shot", commitCtx(p, e))
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Events))
	}
	if result.Events[0].TargetID != 100 {
		t.Errorf("targetID = %d, want 100", result.Events[0].TargetID)
	}
	// 10 base + ~0.3 pressure bonus
	assertDmgNear(t, result.Events[0].Amount, 10.3, "fire_shot damage")
	if p.Cooldowns["fire_shot"] == 0 {
		t.Error("fire_shot cooldown not set")
	}
}

func TestCast_FireShot_MissesBehind(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyBehind(100, 200)

	result := eng.Commit("fire_shot", commitCtx(p, e))
	if !result.OK {
		t.Fatal("commit should succeed even with no hits")
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

	result := eng.Commit("fire_shot", &CommitContext{
		Committer: p,
		Targets:   []entity.Target{e},
		Obstacles: []combat.Obstacle{obs},
	})
	if !result.OK {
		t.Fatal("commit should succeed")
	}
	if len(result.Events) != 0 {
		t.Error("expected 0 hits through obstacle")
	}
}

func TestCast_Execution_AoECone(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	// Enemy in front, one behind
	e1 := enemyInFront(100, 200)
	e3 := enemyBehind(102, 200)

	result := eng.Commit("execution", commitCtx(p, e1, e3))
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
	}
	hitIDs := map[uint16]bool{}
	for _, ev := range result.Events {
		hitIDs[ev.TargetID] = true
	}
	if !hitIDs[100] {
		t.Error("e1 in front should be hit")
	}
	if hitIDs[102] {
		t.Error("e3 behind should not be hit")
	}
	// Stamina spent: 30 (execution cost)
	if p.GetResource("stamina") != 70 {
		t.Errorf("stamina = %f, want 70", p.GetResource("stamina"))
	}
	// Standard tier lockout = 1.2
	if p.GCDTimer != 1.2 {
		t.Errorf("GCDTimer = %f, want 1.2 (lockout)", p.GCDTimer)
	}
}

func TestCast_BDGuard_SelfBuff(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()

	result := eng.Commit("bd_guard", commitCtx(p))
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
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

	result := eng.Commit("fire_shot", commitCtx(p, e))
	if !result.OK {
		t.Fatal("commit failed")
	}
	if len(result.Events) != 1 {
		t.Fatal("expected 1 event")
	}
	// fire_shot base = 10 * 2.0 buff = 20, + pressure bonus ~0.6
	assertDmgNear(t, result.Events[0].Amount, 20.6, "2x buff damage")
}

func TestCast_SetsAttackState(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()
	e := enemyInFront(100, 200)

	eng.Commit("fire_shot", commitCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Errorf("state = %d, want %d (attack)", p.State, entity.PlayerStateAttack)
	}
}

// --- BD ability mechanics ---

func TestCast_BDSpell_ConfigTransition(t *testing.T) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	p.Config = entity.ConfigOrbit // origin = orbit
	e := enemyInFront(100, 500)

	// shielded_sweep: orbit → fan
	result := eng.Commit("shielded_sweep", commitCtx(p, e))
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
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
	result := eng.Commit("guarded_thrust", commitCtx(p, e))
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
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
	result := eng.Commit("targeted_spread", commitCtx(p, e))
	if !result.OK {
		t.Fatalf("commit failed: %s", result.Reason)
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
	eng.Commit("shielded_sweep", commitCtx(p, e))
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

	eng.Commit("shielded_sweep", commitCtx(p))
	if p.GCDTimer != 0.5 {
		t.Errorf("GCDTimer = %f, want 0.5", p.GCDTimer)
	}
}

// --- Handler tests ---

func TestRechamber_StartAndConfirm(t *testing.T) {
	eng := NewEngine(nil)
	p := newGunner()

	// Start rechamber
	r := eng.Commit("rechamber", commitCtx(p))
	if !r.OK {
		t.Fatalf("rechamber start failed: %s", r.Reason)
	}
	state, ok := p.AbilityState["rechamber"].(*RechamberState)
	if !ok {
		t.Fatal("rechamber state not set")
	}
	if state.Phase != 1 {
		t.Errorf("phase = %d, want 1", state.Phase)
	}

	// Can't start again while in progress
	r = eng.Commit("rechamber", commitCtx(p))
	if r.OK {
		t.Error("should not be able to start rechamber while in progress")
	}

	// Tick through windup (0.6s)
	eng.TickPlayer(p, 0.6, tickCtx())
	if state.Phase != 2 {
		t.Errorf("after windup phase = %d, want 2", state.Phase)
	}

	// Confirm in timing window
	r = eng.Commit("rechamber_confirm", commitCtx(p))
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

	eng.Commit("rechamber", commitCtx(p))
	state, ok := p.AbilityState["rechamber"].(*RechamberState)
	if !ok {
		t.Fatal("rechamber state not set")
	}

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
	r := eng.Commit("rechamber_confirm", commitCtx(p))
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

	r := eng.Commit("overclock", commitCtx(p))
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
	r = eng.Commit("overclock", commitCtx(p))
	if r.OK {
		t.Error("should not be able to overclock while active")
	}
}

func TestVGBlock_ParryAndBlock(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()

	r := eng.Commit("vg_block", commitCtx(p))
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
	r = eng.Commit("vg_block", commitCtx(p))
	if r.OK {
		t.Error("should not be able to block while blocking")
	}
}

func TestCleaveVG_Repeatable(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e6)

	// Commit cleave twice — same damage each time, no combo escalation
	r1 := eng.Commit("cleave", commitCtx(p, e))
	if !r1.OK {
		t.Fatalf("cleave 1 failed: %s", r1.Reason)
	}
	dmg1 := r1.Events[0].Amount

	p.Cooldowns["cleave"] = 0
	p.GCDTimer = 0

	r2 := eng.Commit("cleave", commitCtx(p, e))
	if !r2.OK {
		t.Fatalf("cleave 2 failed: %s", r2.Reason)
	}
	dmg2 := r2.Events[0].Amount

	// Both hits should deal 30 base (standard tier, 0 onslaught)
	if dmg1 != 30 {
		t.Errorf("cleave 1 damage = %f, want 30", dmg1)
	}
	if math.Abs(float64(dmg2-dmg1)) > 1 {
		t.Errorf("cleave 2 damage = %f, should match cleave 1 = %f (no combo)", dmg2, dmg1)
	}
}

func TestUpheavalVG(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 500)

	r := eng.Commit("upheaval", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("upheaval failed: %s", r.Reason)
	}
	if p.GetResource("stamina") != 80 {
		t.Errorf("stamina = %f, want 80 (100 - 20)", p.GetResource("stamina"))
	}
	if len(r.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(r.Events))
	}
	// Standard tier upheaval: 55 base damage
	if r.Events[0].Amount != 55 {
		t.Errorf("damage = %f, want 55", r.Events[0].Amount)
	}
}

func TestVortex_Handler(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // within 4 radius

	r := eng.Commit("vortex", commitCtx(p, e))
	if !r.OK {
		t.Fatalf("vortex failed: %s", r.Reason)
	}
	if !p.HasBuff("vortex") {
		t.Error("vortex DR buff not applied")
	}
	if p.GetResource("stamina") != 75 {
		t.Errorf("stamina = %f, want 75 (100-25)", p.GetResource("stamina"))
	}
	// Standard tier: GCD = duration = 0.6s
	if p.GCDTimer != 0.6 {
		t.Errorf("GCDTimer = %f, want 0.6", p.GCDTimer)
	}
	// Immediate first AoE tick should have hit
	if len(r.Events) == 0 {
		t.Error("expected immediate AoE hit")
	}
}

func TestVortex_TickDamage(t *testing.T) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1000)
	e.Position = entity.Vec3{X: 0, Y: 0, Z: -3}

	eng.Commit("vortex", commitCtx(p, e))
	hpAfterCast := e.Health

	// Standard tier: 2 hits, interval = 0.6/2 = 0.3s. Tick past 0.3s.
	events := eng.TickPlayer(p, 0.35, tickCtx(e))
	if len(events) == 0 {
		t.Error("expected AoE tick at 0.3s")
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
		t.Error("stamina changed during regen delay")
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
		SourcePeer: p.ID,
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
		SourcePeer: p.ID,
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

	eng.Commit("fire_shot", commitCtx(p, e))
	if p.State != entity.PlayerStateAttack {
		t.Fatal("expected attack state after commit")
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

	eng.Commit("execution", commitCtx(p, e))
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
	results := resolveHit(nil, def, p, nil, nil, 0)
	if len(results) != 0 {
		t.Error("HitNone should return empty")
	}
}

func TestResolveHit_AoECircle(t *testing.T) {
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
	targets := []entity.Target{e1, e2, e3, eFar}
	results := resolveHit(nil, def, p, targets, nil, 0)
	if len(results) != 3 {
		t.Errorf("hits = %d, want 3 (within radius)", len(results))
	}
}

func TestResolveHit_NearestN(t *testing.T) {
	p := newBladeDancer()

	// 4 enemies with threat, ask for 2 nearest
	targets := make([]entity.Target, 4)
	for i := range targets {
		e := entity.NewEnemy(uint16(i+1), 200, "mob")
		e.Position = entity.Vec3{Z: float32(-(i + 1) * 3)}
		e.ThreatTable[p.ID] = 10 // in combat
		targets[i] = e
	}

	def := &AbilityDef{
		Hit:        HitDef{Type: HitNearestN, TargetCount: 2},
		BaseDamage: 10,
	}
	results := resolveHit(nil, def, p, targets, nil, 0)
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
	e1.ThreatTable[p.ID] = 10

	e2 := entity.NewEnemy(2, 200, "mob")
	e2.Position = entity.Vec3{Z: -4}
	// e2 has no threat — not in combat

	def := &AbilityDef{
		Hit:        HitDef{Type: HitNearestN, TargetCount: 5},
		BaseDamage: 10,
	}
	targets := []entity.Target{e1, e2}
	results := resolveHit(nil, def, p, targets, nil, 0)
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
	eng.Commit("overclock", commitCtx(p))
	p.Cooldowns = make(map[string]float32) // clear overclock CD for test

	eng.Commit("fire_shot", commitCtx(p, e))
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
		{TargetID: 100, SourceID: 1, Amount: 25, Target: e},
		{TargetID: 100, SourceID: 1, Amount: 15, Target: e},
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
	ctx := commitCtx(p, e)
	as := getGunnerAssaultState(p)

	b.ResetTimer()
	for b.Loop() {
		clear(p.Cooldowns)
		p.GCDTimer = 0
		as.MagCurrent = as.MagMax
		as.Reloading = false
		as.MagDumpActive = false
		eng.Commit("fire_shot", ctx)
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
	ctx := commitCtx(p, enemies...)

	b.ResetTimer()
	for b.Loop() {
		clear(p.Cooldowns)
		p.GCDTimer = 0
		p.Resources["stamina"].Current = 100
		eng.Commit("execution", ctx)
	}
}

func BenchmarkCast_BDSpell(b *testing.B) {
	eng := NewEngine(nil)
	p := newBladeDancer()
	e := enemyInFront(100, 1e9)
	ctx := commitCtx(p, e)

	b.ResetTimer()
	for b.Loop() {
		p.Config = entity.ConfigOrbit
		p.GCDTimer = 0
		clear(p.Cooldowns)
		eng.Commit("shielded_sweep", ctx)
	}
}

func BenchmarkCast_Overclock(b *testing.B) {
	eng := NewEngine(nil)
	p := newGunner()
	ctx := commitCtx(p)

	b.ResetTimer()
	for b.Loop() {
		clear(p.Cooldowns)
		p.RemoveBuff("overclock")
		eng.Commit("overclock", ctx)
	}
}

func BenchmarkTickPlayer_Full(b *testing.B) {
	eng := NewEngine(nil)
	p := newVanguard()
	e := enemyInFront(100, 1e9)
	tctx := tickCtx(e)

	// Set up realistic state: cooldowns, buffs, resources, a DoT
	p.Cooldowns["cleave"] = 0.55
	p.Cooldowns["vortex"] = 10.0
	p.GCDTimer = 0.3
	p.AddBuff(entity.ActiveBuff{ID: "test_buff", Type: entity.BuffDamageMult, Value: 1.5, Duration: 5.0})
	p.Resources["stamina"].Current = 60

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset state each iteration to avoid drift — reuse existing slice backing arrays
		p.Alive = true
		p.Cooldowns["cleave"] = 0.55
		p.Cooldowns["vortex"] = 10.0
		p.GCDTimer = 0.3
		p.Resources["stamina"].Current = 60
		p.Buffs = p.Buffs[:0]
		p.Buffs = append(p.Buffs, entity.ActiveBuff{ID: "test_buff", Type: entity.BuffDamageMult, Value: 1.5, Duration: 5.0})
		p.DoTs = p.DoTs[:0]
		p.DoTs = append(p.DoTs, entity.ActiveDoT{
			EnemyID: 100, SourcePeer: p.ID, Damage: 10,
			Remaining: 3.0, Interval: 1.0, TickTimer: 0.05,
		})
		e.Health = 1e9
		e.Alive = true
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
	targets := make([]entity.Target, 10)
	for i := range targets {
		e := entity.NewEnemy(uint16(i+1), 1e9, "mob")
		e.Position = entity.Vec3{X: float32(i-5) * 3, Z: -float32(i+1) * 5}
		targets[i] = e
	}
	var buf []DamageResult

	b.ResetTimer()
	for b.Loop() {
		buf = resolveHitscan(buf[:0], p, targets, nil, 10, 0)
	}
}

func BenchmarkResolveAoECircle(b *testing.B) {
	targets := make([]entity.Target, 20)
	for i := range targets {
		e := entity.NewEnemy(uint16(i+1), 1e9, "mob")
		e.Position = entity.Vec3{X: float32(i%5) * 2, Z: float32(i/5) * 2}
		targets[i] = e
	}
	center := entity.Vec3{X: 4, Z: 2}
	var buf []DamageResult

	b.ResetTimer()
	for b.Loop() {
		buf = resolveAoECircle(buf[:0], center, 1, targets, nil, 8, 10, 0)
	}
}

// --- Arcanotechnicien / Harmonist ability benchmarks ---

func newHarmonistBench() *entity.Player {
	p := entity.NewPlayer(10, entity.ClassArcanotechnicien)
	p.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	p.RotationY = 0
	return p
}

func commitCtxWithAllies(p *entity.Player, enemies []*entity.Enemy, allies map[uint16]*entity.Player) *CommitContext {
	targets := make([]entity.Target, len(enemies))
	for i, e := range enemies {
		targets[i] = e
	}
	return &CommitContext{
		Committer: p,
		Targets:   targets,
		Allies:    allies,
		SpawnZone: func(*entity.HealingZone) {},
		SpawnLink: func(*entity.DamageLink) {},
	}
}

func BenchmarkTickPlayer_Harmonist(b *testing.B) {
	eng := NewEngine(nil)
	p := newHarmonistBench()
	e := enemyInFront(100, 1e9)
	tctx := tickCtx(e)

	// Set up realistic state
	p.Cooldowns["siphon_pulse"] = 0.3
	p.Cooldowns["restoration_matrix"] = 8.0
	p.GCDTimer = 0.2
	p.AddBuff(entity.ActiveBuff{ID: "frost_ward", Type: entity.BuffDamageReduction, Value: 1.0, Duration: 4.0})
	p.Confluence.OnAbilityComplete()
	p.Confluence.OnAbilityComplete()
	p.Confluence.OnAbilityComplete() // 3 stacks
	p.VitalCharge = 15
	p.VitalChargeTimer = 2.0

	// Partially spend flux pools so regen has work
	for i := range p.FluxCommit.Pools {
		p.FluxCommit.Pools[i].Current = p.FluxCommit.Pools[i].Max * 0.5
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset state each iteration
		p.Alive = true
		p.Cooldowns["siphon_pulse"] = 0.3
		p.Cooldowns["restoration_matrix"] = 8.0
		p.GCDTimer = 0.2
		p.Buffs = p.Buffs[:0]
		p.Buffs = append(p.Buffs, entity.ActiveBuff{ID: "frost_ward", Type: entity.BuffDamageReduction, Value: 1.0, Duration: 4.0})
		p.Confluence.Stacks = 3
		p.Confluence.IdleTimer = 1.0
		p.Confluence.DecayTimer = 0
		p.VitalCharge = 15
		p.VitalChargeTimer = 2.0
		for i := range p.FluxCommit.Pools {
			p.FluxCommit.Pools[i].Current = p.FluxCommit.Pools[i].Max * 0.5
		}
		e.Health = 1e9
		e.Alive = true
		eng.TickPlayer(p, 0.05, tctx)
	}
}

func BenchmarkAbilityCommit_SiphonPulse(b *testing.B) {
	eng := NewEngine(nil)
	p := newHarmonistBench()
	e := enemyInFront(100, 1e9)

	// Build allies map (4 gunners at varying HP)
	allies := make(map[uint16]*entity.Player, 5)
	allies[p.ID] = p
	for i := uint16(1); i <= 4; i++ {
		a := entity.NewPlayer(i, entity.ClassGunner)
		a.Position = entity.Vec3{X: float32(i), Y: 0, Z: 0}
		a.Health = a.MaxHealth * float32(0.5+float64(i)*0.1) // varying HP
		allies[i] = a
	}

	ctx := commitCtxWithAllies(p, []*entity.Enemy{e}, allies)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Reset state
		p.GCDTimer = 0
		p.Confluence.Stacks = 2
		p.Confluence.IdleTimer = 0
		e.Health = 1e9
		e.Alive = true
		for _, a := range allies {
			a.Health = a.MaxHealth * 0.7
		}
		eng.Commit("siphon_pulse", ctx)
	}
}

func BenchmarkAbilityCommit_VitalCircuit(b *testing.B) {
	eng := NewEngine(nil)
	p := newHarmonistBench()

	ally := entity.NewPlayer(2, entity.ClassGunner)
	ally.Position = entity.Vec3{X: 3, Y: 0, Z: 0}
	allies := map[uint16]*entity.Player{p.ID: p, ally.ID: ally}
	ctx := commitCtxWithAllies(p, nil, allies)
	ctx.TargetPeerID = ally.ID

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.GCDTimer = 0
		delete(p.Cooldowns, "vital_circuit")
		p.Confluence.Stacks = 2
		p.Confluence.IdleTimer = 0
		// Reset biometabolic flux pool
		pool := p.FluxCommit.GetPool("biometabolic")
		pool.Current = pool.Max
		ally.Alive = true
		eng.Commit("vital_circuit", ctx)
	}
}

func BenchmarkAbilityCommit_RestorationMatrix(b *testing.B) {
	eng := NewEngine(nil)
	p := newHarmonistBench()
	p.GearStats.Identity = 12

	allies := map[uint16]*entity.Player{p.ID: p}
	ctx := commitCtxWithAllies(p, nil, allies)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.GCDTimer = 0
		delete(p.Cooldowns, "restoration_matrix")
		p.Confluence.Stacks = 2
		p.Confluence.IdleTimer = 0
		// Reset bioarcanotechnic flux pool
		pool := p.FluxCommit.GetPool("bioarcanotechnic")
		pool.Current = pool.Max
		eng.Commit("restoration_matrix", ctx)
	}
}

func BenchmarkAbilityCommit_FrostWard(b *testing.B) {
	eng := NewEngine(nil)
	p := newHarmonistBench()

	ally := entity.NewPlayer(2, entity.ClassGunner)
	ally.Position = entity.Vec3{X: 3, Y: 0, Z: 0}
	allies := map[uint16]*entity.Player{p.ID: p, ally.ID: ally}
	ctx := commitCtxWithAllies(p, nil, allies)
	ctx.TargetPeerID = ally.ID

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.GCDTimer = 0
		delete(p.Cooldowns, "frost_ward")
		p.Confluence.Stacks = 2
		p.Confluence.IdleTimer = 0
		// Reset frost flux pool
		pool := p.FluxCommit.GetPool("frost")
		pool.Current = pool.Max
		// Remove shield and buff so handler re-creates them
		ally.RemoveBuff("frost_ward")
		delete(ally.AbilityState, "frost_ward_active")
		delete(ally.Resources, "shield")
		eng.Commit("frost_ward", ctx)
	}
}

func BenchmarkAbilityValidation_InsufficientFlux(b *testing.B) {
	eng := NewEngine(nil)
	p := newHarmonistBench()
	// Drain all flux so validation fails
	p.SetAllFluxPoolsCurrent(0)

	allies := map[uint16]*entity.Player{p.ID: p}
	ctx := commitCtxWithAllies(p, nil, allies)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.GCDTimer = 0
		delete(p.Cooldowns, "restoration_matrix")
		p.SetAllFluxPoolsCurrent(0)
		eng.Commit("restoration_matrix", ctx)
	}
}
