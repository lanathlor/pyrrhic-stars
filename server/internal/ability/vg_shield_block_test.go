package ability

import (
	"testing"

	"codex-online/server/internal/entity"
)

func TestShieldBlock_StartAppliesBuffsAndState(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	r := eng.Commit(IDVgShieldBlock, commitCtx(p))
	if !r.OK {
		t.Fatalf("commit failed: %s", r.Reason)
	}
	if !p.HasBuff("vg_shield_parry") {
		t.Error("shield parry buff should be applied")
	}
	if !p.HasBuff(IDVgShieldBlock) {
		t.Error("shield block buff should be applied")
	}
	b := p.GetBuff(IDVgShieldBlock)
	if b == nil || b.Value != shieldBlockDRStart {
		t.Errorf("block DR = %v, want %v", b.Value, shieldBlockDRStart)
	}
	if p.State != entity.PlayerStateBlock {
		t.Errorf("state = %d, want %d (block)", p.State, entity.PlayerStateBlock)
	}

	// DevotionMult should start at 1.0
	state := getVgShieldBlockState(p)
	if state.DevotionMult != devotionMultStart {
		t.Errorf("DevotionMult = %f, want %f", state.DevotionMult, devotionMultStart)
	}
}

func TestShieldBlock_ParryExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.TickPlayer(p, 0.15, tickCtx())
	if p.HasBuff("vg_shield_parry") {
		t.Error("shield parry should have expired after 0.15s")
	}
	if !p.HasBuff(IDVgShieldBlock) {
		t.Error("shield block should still be active")
	}
}

func TestShieldBlock_DRDecays(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))

	// At start: DR should be shieldBlockDRStart (0.10)
	b := p.GetBuff(IDVgShieldBlock)
	if b == nil {
		t.Fatal("shield block buff should be active")
	}
	if b.Value != shieldBlockDRStart {
		t.Errorf("initial DR = %f, want %f", b.Value, shieldBlockDRStart)
	}

	// Tick 0.5s — midpoint of decay
	eng.TickPlayer(p, 0.5, tickCtx())
	b = p.GetBuff(IDVgShieldBlock)
	midpoint := shieldBlockDRStart + (shieldBlockDREnd-shieldBlockDRStart)*0.5
	if b.Value < midpoint-0.02 || b.Value > midpoint+0.02 {
		t.Errorf("DR at 0.5s = %f, want ~%f", b.Value, midpoint)
	}

	// Tick past decay time — should clamp to DREnd
	eng.TickPlayer(p, 1.0, tickCtx())
	eng.TickPlayer(p, 1.0, tickCtx())
	b = p.GetBuff(IDVgShieldBlock)
	if b.Value != shieldBlockDREnd {
		t.Errorf("DR after full decay = %f, want %f (clamped)", b.Value, shieldBlockDREnd)
	}
}

func TestShieldBlock_NoPersecondStaminaDrain(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	initialStamina := p.GetResource("stamina")

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	// Tick 1 second — Shield block should NOT drain stamina per-second
	for range 20 {
		eng.TickPlayer(p, 0.05, tickCtx())
	}

	stam := p.GetResource("stamina")
	// Stamina should be unchanged (no per-second drain; regen delay timer is reset on commit)
	if stam < initialStamina-1 {
		t.Errorf("stamina = %f, want ~%f (no per-second drain)", stam, initialStamina)
	}
}

func TestShieldBlock_DamageDrainsStamina(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	// Expire parry
	eng.TickPlayer(p, 0.2, tickCtx())

	initialStamina := p.GetResource("stamina")
	// Take 100 damage while blocking.
	// Stamina drain uses pre-DR amount: 100 * 0.65 (drain fraction) * 1.0 (tenacity) = 65
	p.ApplyDamage(100)

	stam := p.GetResource("stamina")
	expectedDrain := float32(100) * ShieldStaminaDrainFraction // pre-DR based
	expectedStam := initialStamina - expectedDrain
	if stam < expectedStam-2.0 || stam > expectedStam+2.0 {
		t.Errorf("stamina = %f, want ~%f (after blocking 100 damage)", stam, expectedStam)
	}
}

func TestShieldBlock_DamageGeneratesDevotion(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.TickPlayer(p, 0.2, tickCtx()) // expire parry

	p.ApplyDamage(100)

	dev := getDevotionState(p)
	if dev.Charges <= 0 {
		t.Error("Devotion charges should increase from blocked damage")
	}
}

func TestShieldBlock_DevotionMultDecays(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))

	// During parry window: DevotionMult should be 1.0
	state := getVgShieldBlockState(p)
	if state.DevotionMult != devotionMultStart {
		t.Errorf("DevotionMult during parry = %f, want %f", state.DevotionMult, devotionMultStart)
	}

	// Tick past parry + halfway through decay
	eng.TickPlayer(p, shieldGuardParryWindow+0.5, tickCtx())
	state = getVgShieldBlockState(p)
	midpoint := devotionMultStart - (devotionMultStart-devotionMultEnd)*0.5
	if state.DevotionMult < midpoint-0.1 || state.DevotionMult > midpoint+0.1 {
		t.Errorf("DevotionMult at midpoint = %f, want ~%f", state.DevotionMult, midpoint)
	}

	// Tick well past decay — should clamp to floor
	eng.TickPlayer(p, 2.0, tickCtx())
	state = getVgShieldBlockState(p)
	if state.DevotionMult != devotionMultEnd {
		t.Errorf("DevotionMult after full decay = %f, want %f", state.DevotionMult, devotionMultEnd)
	}
}

func TestShieldBlock_SustainedBlockReducesDevotion(t *testing.T) {
	eng := NewEngine(nil)

	// Fresh block: apply damage immediately after parry expires
	p1 := newShieldVanguard()
	eng.Commit(IDVgShieldBlock, commitCtx(p1))
	eng.TickPlayer(p1, 0.15, tickCtx()) // just past parry
	p1.ApplyDamage(100)
	freshCharges := getDevotionState(p1).Charges

	// Stale block: apply damage after full decay
	p2 := newShieldVanguard()
	eng.Commit(IDVgShieldBlock, commitCtx(p2))
	eng.TickPlayer(p2, 2.0, tickCtx()) // well past decay
	p2.ApplyDamage(100)
	staleCharges := getDevotionState(p2).Charges

	if staleCharges >= freshCharges {
		t.Errorf("stale block Devotion (%f) should be less than fresh block (%f)", staleCharges, freshCharges)
	}
	// Stale should be roughly 25% of fresh
	ratio := staleCharges / freshCharges
	if ratio < 0.15 || ratio > 0.35 {
		t.Errorf("stale/fresh ratio = %f, want ~0.25", ratio)
	}
}

func TestShieldBlock_GuardParryReflect(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	enemy := enemyInFront(100, 500)

	eng.Commit(IDVgShieldBlock, commitCtx(p, enemy))

	// Parry is active — take damage to trigger reflect
	p.ApplyDamage(100)

	// Resolve reflect on next tick
	events := eng.TickPlayer(p, 0.01, tickCtx(enemy))
	if len(events) == 0 {
		t.Error("expected parry reflect damage events")
	}

	// Check bonus Devotion from parry (2x rate)
	dev := getDevotionState(p)
	if dev.Charges <= 0 {
		t.Error("Devotion should be generated from Guard Parry")
	}
}

func TestShieldBlock_GuardBreakOnStaminaDepletion(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 1 // very low stamina

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.TickPlayer(p, 0.2, tickCtx()) // expire parry

	// Take heavy damage — should drain remaining stamina below 0
	p.ApplyDamage(200)

	// Next tick should trigger Guard Break
	eng.TickPlayer(p, 0.05, tickCtx())

	if p.HasBuff(IDVgShieldBlock) {
		t.Error("shield block should have ended on Guard Break")
	}
	if p.State != entity.PlayerStateStagger {
		t.Errorf("state = %d, want %d (stagger)", p.State, entity.PlayerStateStagger)
	}
	if !p.HasBuff("guard_break") {
		t.Error("guard_break vulnerability buff should be applied")
	}
	b := p.GetBuff("guard_break")
	if b == nil || b.Value != guardBreakVulnMult {
		t.Errorf("guard_break value = %v, want %v", b.Value, guardBreakVulnMult)
	}
}

func TestShieldBlock_StopAction(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	r := eng.Commit(IDVgShieldBlockStop, commitCtx(p))
	if !r.OK {
		t.Fatalf("stop failed: %s", r.Reason)
	}
	if p.HasBuff(IDVgShieldBlock) {
		t.Error("shield block buff should be removed")
	}
	if p.Cooldowns[IDVgShieldBlock] < 1.4 || p.Cooldowns[IDVgShieldBlock] > 1.6 {
		t.Errorf("cooldown = %f, want 1.5", p.Cooldowns[IDVgShieldBlock])
	}
}

func TestShieldBlock_CooldownPreventsReblock(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.Commit(IDVgShieldBlockStop, commitCtx(p))

	r := eng.Commit(IDVgShieldBlock, commitCtx(p))
	if r.OK {
		t.Error("should not re-block during cooldown")
	}
}

func TestShieldBlock_CooldownExpires(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.Commit(IDVgShieldBlockStop, commitCtx(p))
	eng.TickPlayer(p, 2.0, tickCtx())

	r := eng.Commit(IDVgShieldBlock, commitCtx(p))
	if !r.OK {
		t.Fatalf("re-block after cooldown should succeed: %s", r.Reason)
	}
}

func TestShieldBlock_ZeroStaminaPreventsStart(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()
	p.Resources["stamina"].Current = 0

	r := eng.Commit(IDVgShieldBlock, commitCtx(p))
	if r.OK {
		t.Error("should not block with 0 stamina")
	}
}

func TestShieldBlock_DoesNotResetOnslaught(t *testing.T) {
	p := newShieldVanguard()
	// Shield spec should not have Onslaught, but verify damage doesn't panic
	p.ApplyDamage(50)
	// No onslaught state should exist
	if _, ok := p.AbilityState["onslaught"]; ok {
		t.Error("Shield spec should not have onslaught state")
	}
}

func TestShieldBlock_BraceFreezesDRDecay(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	// Tick past parry window to start decay
	eng.TickPlayer(p, 0.3, tickCtx())

	// Record DR before Brace
	b := p.GetBuff(IDVgShieldBlock)
	drBeforeBrace := b.Value

	// Commit Brace
	eng.Commit(IDBrace, commitCtx(p))

	// Tick 2 seconds — DR should NOT decay further while braced
	eng.TickPlayer(p, 2.0, tickCtx())

	b = p.GetBuff(IDVgShieldBlock)
	if b == nil {
		t.Fatal("shield block should still be active")
	}
	if b.Value != drBeforeBrace {
		t.Errorf("DR during Brace = %f, want %f (frozen)", b.Value, drBeforeBrace)
	}
}

func TestShieldBlock_BraceFreezesDevotionDecay(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.TickPlayer(p, 0.3, tickCtx())

	state := getVgShieldBlockState(p)
	devMultBeforeBrace := state.DevotionMult

	eng.Commit(IDBrace, commitCtx(p))
	eng.TickPlayer(p, 2.0, tickCtx())

	state = getVgShieldBlockState(p)
	if state.DevotionMult != devMultBeforeBrace {
		t.Errorf("DevotionMult during Brace = %f, want %f (frozen)", state.DevotionMult, devMultBeforeBrace)
	}
}

func TestShieldBlock_EndResetsDevotionMult(t *testing.T) {
	eng := NewEngine(nil)
	p := newShieldVanguard()

	eng.Commit(IDVgShieldBlock, commitCtx(p))
	eng.Commit(IDVgShieldBlockStop, commitCtx(p))

	state := getVgShieldBlockState(p)
	if state.DevotionMult != 0 {
		t.Errorf("DevotionMult after end = %f, want 0", state.DevotionMult)
	}
}
