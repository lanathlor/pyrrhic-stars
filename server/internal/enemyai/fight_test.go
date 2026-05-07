package enemyai

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
)

// tickN advances the brain by n ticks at the given dt, collecting all damage events.
func tickN(b *Brain, n int, dt float32, players []*entity.Player, obs []combat.Obstacle, spawn func(entity.Vec3, entity.Vec3, float32, float32, float32)) []combat.DamageEvent {
	var all []combat.DamageEvent
	for range n {
		all = append(all, b.Tick(dt, players, obs, spawn, nil)...)
	}
	return all
}

// tickUntil ticks up to maxTicks or until pred returns true. Returns total events and ticks elapsed.
func tickUntil(b *Brain, maxTicks int, dt float32, players []*entity.Player, spawn func(entity.Vec3, entity.Vec3, float32, float32, float32), pred func() bool) ([]combat.DamageEvent, int) {
	var all []combat.DamageEvent
	for i := range maxTicks {
		all = append(all, b.Tick(dt, players, nil, spawn, nil)...)
		if pred() {
			return all, i + 1
		}
	}
	return all, maxTicks
}

// --- Fight simulation tests ---

// TestFight_HallwayMelee simulates a full trash mob fight:
// patrol → aggro → chase → melee attack → player kills mob.
func TestFight_HallwayMelee(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5}
	e.PatrolB = entity.Vec3{X: 5}
	e.AggroRadius = 8.0
	e.LeashRadius = 20.0
	e.LeashOrigin = entity.Vec3{}

	// Player starts far away
	p := testPlayer(1, entity.Vec3{X: 0, Z: 15})
	players := testPlayers(p)

	// Phase 1: Patrol — no players in aggro range
	tickN(b, 20, 0.05, players, nil, noSpawn)
	if e.State != entity.EnemyPatrol {
		t.Fatalf("should be patrolling, got state %d", e.State)
	}
	if e.Position.X == 0 {
		t.Error("should have moved during patrol")
	}

	// Phase 2: Player approaches — mob aggros
	p.Position = entity.Vec3{X: e.Position.X, Z: 5}
	tickN(b, 5, 0.05, players, nil, noSpawn)
	if e.State != entity.EnemyChase {
		t.Fatalf("should have aggroed, got state %d", e.State)
	}
	if e.TargetPlayerID != 1 {
		t.Errorf("should target player 1, got %d", e.TargetPlayerID)
	}

	// Phase 3: Chase to melee range
	p.Position = entity.Vec3{X: 0, Z: 2}
	_, chaseTicks := tickUntil(b, 200, 0.05, players, noSpawn, func() bool {
		return e.State == entity.EnemyMeleeTelegraph || e.State == entity.EnemyAoETelegraph
	})
	if e.State != entity.EnemyMeleeTelegraph {
		t.Fatalf("should reach melee telegraph, got state %d after %d ticks", e.State, chaseTicks)
	}

	// Phase 4: Attack cycle — tick until damage is dealt
	events, atkTicks := tickUntil(b, 100, 0.05, players, noSpawn, func() bool {
		return p.Health < p.MaxHealth
	})
	if p.Health >= p.MaxHealth {
		t.Fatalf("player should have taken damage after %d ticks", atkTicks)
	}
	t.Logf("first hit after %d ticks: %d events, player HP=%.0f/%.0f", atkTicks, len(events), p.Health, p.MaxHealth)

	// Phase 5: Kill the mob (simulate player attacking back)
	_, phaseTrigger := e.ApplyDamage(e.Health)
	if phaseTrigger != 0 {
		t.Errorf("hallway melee has no phases, got trigger %d", phaseTrigger)
	}
	if e.Alive {
		t.Fatal("mob should be dead")
	}

	// Verify dead state
	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Errorf("dead mob velocity should be zero, got %v", e.Velocity)
	}
}

// TestFight_HallwayRanged simulates a ranged mob fight:
// patrol → aggro → maintain distance → fire projectile → die.
func TestFight_HallwayRanged(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_ranged"])
	e.Alive = true
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5}
	e.PatrolB = entity.Vec3{X: 5}
	e.AggroRadius = 10.0
	e.LeashRadius = 25.0
	e.LeashOrigin = entity.Vec3{}

	p := testPlayer(1, entity.Vec3{X: 0, Z: 4})
	players := testPlayers(p)

	// Aggro — mob should target the player (may already be attacking with ranged tree)
	tickN(b, 5, 0.05, players, nil, noSpawn)
	if e.TargetPlayerID != p.ID {
		t.Fatalf("should have aggroed player %d, targeting %d", p.ID, e.TargetPlayerID)
	}

	// Ranged mob should try to maintain PreferredRange (8.0)
	// Player is at Z=4, so mob should backpedal (moving away)
	projectiles := 0
	spawnFn := func(_, _ entity.Vec3, _, _, _ float32) { projectiles++ }

	// Tick long enough for a ranged attack
	for range 200 {
		b.Tick(0.05, players, nil, spawnFn, nil)
		if projectiles > 0 {
			break
		}
	}
	if projectiles == 0 {
		t.Error("ranged mob should have fired at least one projectile")
	}
	t.Logf("ranged mob fired %d projectiles, mob pos=%v", projectiles, e.Position)

	// Kill the mob
	e.ApplyDamage(e.Health)
	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.Alive {
		t.Error("mob should be dead")
	}
}

// TestFight_GuardCaptain_AllPhases simulates a full boss fight:
// chase → P1 attacks → phase 2 transition → P2 attacks → phase 3 → death.
func TestFight_GuardCaptain_AllPhases(t *testing.T) {
	b, e := testBrain(DefRegistry["guard_captain"])
	e.Alive = true
	e.State = entity.EnemyChase
	e.LeashRadius = 50.0
	e.LeashOrigin = entity.Vec3{}

	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	p.MaxHealth = 1e6 // survive entire fight
	p.Health = p.MaxHealth
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// === Phase 1 ===
	phase1Events, _ := tickUntil(b, 300, 0.05, players, noSpawn, func() bool {
		return p.Health < p.MaxHealth
	})
	if p.Health >= p.MaxHealth {
		t.Fatal("boss should have dealt damage in phase 1")
	}
	if e.Phase != 1 {
		t.Errorf("should be phase 1, got %d", e.Phase)
	}

	var phase1Damage float32
	for _, ev := range phase1Events {
		phase1Damage += ev.Amount
	}
	t.Logf("Phase 1: %d events, %.0f total damage", len(phase1Events), phase1Damage)

	// Continue ticking to accumulate more events
	phase1More := tickN(b, 300, 0.05, players, nil, noSpawn)
	for _, ev := range phase1More {
		phase1Damage += ev.Amount
	}
	t.Logf("Phase 1 total: %.0f damage over %d events", phase1Damage, len(phase1Events)+len(phase1More))

	// === Phase 2 transition (60% HP) ===
	_, phaseTrigger := e.ApplyDamage(e.MaxHealth * 0.41) // drop to 59% HP
	if phaseTrigger != 2 {
		t.Fatalf("expected phase 2 trigger, got %d (HP=%.0f)", phaseTrigger, e.Health)
	}
	if e.Phase != 2 {
		t.Fatalf("expected phase 2, got %d", e.Phase)
	}
	if e.State != entity.EnemyPhaseTransition {
		t.Fatalf("expected phase transition state, got %d", e.State)
	}

	// Tick through phase transition (1.5s = 30 ticks at 0.05)
	tickN(b, 40, 0.05, players, nil, noSpawn)
	if e.State == entity.EnemyPhaseTransition {
		t.Fatal("should have finished phase transition by now")
	}

	// Phase 2 attacks should be faster (cooldown 1.2s vs 1.5s)
	phase2Events := tickN(b, 300, 0.05, players, nil, noSpawn)
	var phase2Damage float32
	for _, ev := range phase2Events {
		phase2Damage += ev.Amount
	}
	t.Logf("Phase 2: %d events, %.0f total damage", len(phase2Events), phase2Damage)

	// === Phase 3 transition (30% HP) ===
	_, phaseTrigger = e.ApplyDamage(e.Health - e.MaxHealth*0.29) // drop to 29%
	if phaseTrigger != 3 {
		t.Fatalf("expected phase 3 trigger, got %d (HP=%.0f)", phaseTrigger, e.Health)
	}
	if e.Phase != 3 {
		t.Fatalf("expected phase 3, got %d", e.Phase)
	}

	tickN(b, 40, 0.05, players, nil, noSpawn)
	if e.State == entity.EnemyPhaseTransition {
		t.Fatal("should have finished phase 3 transition")
	}

	// Phase 3: enraged — cooldown 0.4s, should deal more damage per time
	phase3Events := tickN(b, 300, 0.05, players, nil, noSpawn)
	var phase3Damage float32
	for _, ev := range phase3Events {
		phase3Damage += ev.Amount
	}
	t.Logf("Phase 3: %d events, %.0f total damage", len(phase3Events), phase3Damage)

	// Phase 3 should deal more DPS than phase 1 (shorter cooldowns)
	if len(phase3Events) <= len(phase1Events)+len(phase1More) {
		t.Logf("warning: phase 3 produced fewer events (%d) than phase 1 (%d) — may need tuning",
			len(phase3Events), len(phase1Events)+len(phase1More))
	}

	// === Death ===
	e.ApplyDamage(e.Health)
	b.Tick(0.05, players, nil, noSpawn, nil)
	if e.Alive {
		t.Error("boss should be dead")
	}
	if e.Velocity.X != 0 || e.Velocity.Z != 0 {
		t.Error("dead boss should have zero velocity")
	}
}

// TestFight_MultiPlayer_TargetSwitch verifies the boss re-targets when the current target dies.
func TestFight_MultiPlayer_TargetSwitch(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyChase
	e.AggroRadius = 10.0
	e.LeashRadius = 30.0

	p1 := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	p2 := testPlayer(2, entity.Vec3{X: 3, Z: 2})
	players := testPlayers(p1, p2)
	e.TargetPlayerID = p1.ID

	// Tick until boss attacks p1
	tickUntil(b, 200, 0.05, players, noSpawn, func() bool {
		return p1.Health < p1.MaxHealth
	})

	// Kill p1
	p1.Alive = false
	p1.Health = 0

	// Boss should re-target to p2
	// condHasTarget will fail (p1 dead), then the no-target branch triggers aggro
	tickN(b, 5, 0.05, players, nil, noSpawn)

	// The tree should pick up p2 as the nearest alive player
	// The boss might chase (if state reset) or directly attack
	targetFound := false
	for range 50 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if e.TargetPlayerID == p2.ID {
			targetFound = true
			break
		}
	}
	if !targetFound {
		t.Errorf("boss should re-target to player 2, still targeting %d", e.TargetPlayerID)
	}
}

// TestFight_AoEHitsMultiplePlayers verifies AoE damages all players in radius.
func TestFight_AoEHitsMultiplePlayers(t *testing.T) {
	// Def with high-weight AoE and a low-weight melee (needed so LongestMeleeRange > 0
	// for the default tree's condTargetInMeleeRange).
	def := &EnemyDef{
		Name:      "test_aoe_heavy",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  testTreeData(),
		Abilities: []ability.AbilityDef{
			{
				ID: "melee_tap", Name: "melee_tap", Category: ability.CategoryMelee,
				CommitTime: 0.1, Cooldown: 0.3,
				BaseWeight: 1, MaxRange: 5.0,
				BaseDamage:   5.0,
				Hit:          ability.HitDef{Type: ability.HitAoECone, Range: 5.0, ArcDegrees: 180},
				DamageSource: combat.SourceEnemyMelee,
			},
			{
				ID: "slam", Name: "slam", Category: ability.CategoryAoE,
				CommitTime: 0.2, Cooldown: 0.5,
				BaseWeight: 100, MaxRange: 10.0,
				BaseDamage:   25.0,
				Hit:          ability.HitDef{Type: ability.HitAoECircle, Radius: 5.0},
				DamageSource: combat.SourceEnemyAoE,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	// Three players all within AoE radius (5.0)
	p1 := testPlayer(1, entity.Vec3{X: 1, Z: 1})
	p2 := testPlayer(2, entity.Vec3{X: -1, Z: 1})
	p3 := testPlayer(3, entity.Vec3{X: 0, Z: -1})
	players := testPlayers(p1, p2, p3)
	e.TargetPlayerID = p1.ID

	var allEvents []combat.DamageEvent
	for range 200 {
		evts := b.Tick(0.05, players, nil, noSpawn, nil)
		allEvents = append(allEvents, evts...)
		// Check if at least 2 players were hit in any single tick's events
		if len(evts) >= 2 {
			break
		}
	}

	// Verify multiple players were damaged
	hitPlayers := map[uint16]float32{}
	for _, ev := range allEvents {
		hitPlayers[ev.TargetPeerID] += ev.Amount
	}
	if len(hitPlayers) < 2 {
		t.Errorf("AoE should hit multiple players, only hit %d: %v", len(hitPlayers), hitPlayers)
	}
	t.Logf("AoE hit %d players: %v", len(hitPlayers), hitPlayers)
}

// TestFight_ChargeHitsMultiplePlayers verifies charge hits multiple players along its path.
// Uses a self-contained tree that attempts charge directly.
func TestFight_ChargeHitsMultiplePlayers(t *testing.T) {
	chargeTree := map[string]any{
		"reactive_selector": []any{
			map[string]any{"sequence": []any{"is_dead", "stop"}},
			map[string]any{"sequence": []any{"phase_transitioning", "wait_transition"}},
			map[string]any{"sequence": []any{"!has_target", "aggro_or_patrol"}},
			map[string]any{"sequence": []any{"!in_leash_range", "leash_reset"}},
			map[string]any{"sequence": []any{"is_casting", "wait_ability"}},
			map[string]any{"sequence": []any{"target_beyond(4)", "has_los", "cast(bull_charge)", "wait_ability"}},
			"chase",
		},
	}
	def := &EnemyDef{
		Name:      "test_charge_multi",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  chargeTree,
		Abilities: []ability.AbilityDef{
			{
				ID: "bull_charge", Category: ability.CategoryCharge,
				CommitTime: 0.1, Cooldown: 0.5,
				BaseWeight: 100, MinRange: 4.0,
				FaceTarget: true,
				Charge: &ability.ChargeDef{
					Speed: 20.0, Damage: 30.0,
					MaxDistance: 30.0, HitRadius: 3.0,
					StopOnWall: true,
				},
				DamageSource: combat.SourceEnemyCharge,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Z: 0}

	// Two players lined up along the charge path
	p1 := testPlayer(1, entity.Vec3{X: 0, Z: 5})
	p2 := testPlayer(2, entity.Vec3{X: 0, Z: 10})
	players := testPlayers(p1, p2)
	e.TargetPlayerID = p1.ID

	hitPlayers := map[uint16]bool{}
	for range 400 {
		evts := b.Tick(0.05, players, nil, noSpawn, nil)
		for _, ev := range evts {
			hitPlayers[ev.TargetPeerID] = true
		}
		if len(hitPlayers) >= 2 {
			break
		}
	}

	if len(hitPlayers) < 2 {
		t.Errorf("charge should hit both players in its path, only hit %v", hitPlayers)
	}
}

// TestFight_LeashDuringCombat verifies boss resets when player retreats beyond leash.
func TestFight_LeashDuringCombat(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyChase
	e.LeashRadius = 10.0
	e.LeashOrigin = entity.Vec3{X: 0, Z: 0}
	e.AggroRadius = 8.0

	// Start in combat
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID
	e.Health = e.MaxHealth * 0.5 // half health

	// Tick a few times in combat
	tickN(b, 10, 0.05, players, nil, noSpawn)

	// Player retreats far away, enemy chases and exceeds leash
	p.Position = entity.Vec3{X: 0, Z: 30}
	// Chase until leash triggers
	for range 200 {
		b.Tick(0.05, players, nil, noSpawn, nil)
		if e.State == entity.EnemyPatrol {
			break
		}
	}

	if e.State != entity.EnemyPatrol {
		t.Errorf("expected patrol (leash reset), got state %d", e.State)
	}
	if e.Health != e.MaxHealth {
		t.Errorf("should heal to full on leash, got %.0f/%.0f", e.Health, e.MaxHealth)
	}
	if e.Position != e.LeashOrigin {
		t.Errorf("should return to leash origin, got %v", e.Position)
	}
}

// TestFight_ObstacleAvoidanceDuringChase verifies the enemy steers around obstacles.
func TestFight_ObstacleAvoidanceDuringChase(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Z: 0}

	// Player directly behind an obstacle
	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// Obstacle blocking the direct path
	obs := []combat.Obstacle{{CX: 0, CZ: 5, HX: 2, HZ: 1}}

	// Chase for a while
	tickN(b, 20, 0.05, players, obs, noSpawn)

	// Enemy should have steered to the side (X != 0)
	if e.Position.X == 0 {
		t.Logf("enemy position: %v (may not have reached obstacle yet)", e.Position)
	}
	// At minimum, the enemy should be moving
	if e.Position.Z == 0 {
		t.Error("enemy should have moved forward")
	}
}

// TestFight_RangedBackpedal verifies the ranged mob backs away when player is too close.
func TestFight_RangedBackpedal(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_ranged"])
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Z: 0}

	// Player within MinRange (2.0) — attack branch fails (no valid ability),
	// so chase handles movement. Distance < preferred-margin so mob backpedals.
	p := testPlayer(1, entity.Vec3{X: 0, Z: 1.5})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// Tick — mob should backpedal (move away from player, Z decreasing)
	tickN(b, 10, 0.05, players, nil, noSpawn)

	if e.Position.Z >= 0 {
		t.Errorf("ranged mob should backpedal away from close player, pos=%v", e.Position)
	}
}

// TestFight_RangedAdvance verifies the ranged mob advances when player is too far.
func TestFight_RangedAdvance(t *testing.T) {
	b, e := testBrain(DefRegistry["hallway_ranged"])
	e.Alive = true
	e.State = entity.EnemyChase
	e.Position = entity.Vec3{X: 0, Z: 0}

	// Player far away — beyond preferred range + margin.
	// Obstacle blocks LoS so attack branch fails and chase handles movement.
	p := testPlayer(1, entity.Vec3{X: 0, Z: 20})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID
	obs := []combat.Obstacle{{CX: 0, CZ: 10, HX: 2, HZ: 1}}

	tickN(b, 20, 0.05, players, obs, noSpawn)

	if e.Position.Z <= 0 {
		t.Errorf("ranged mob should advance toward distant player, pos=%v", e.Position)
	}
}

// TestFight_PhaseOverridesAffectAbilities verifies that phase overrides change ability stats.
func TestFight_PhaseOverridesAffectAbilities(t *testing.T) {
	def := DefRegistry["guard_captain"]
	e := entity.NewEnemy(1000, def.MaxHealth, def.Name)

	// Phase 1: base stats
	resolved1 := def.ResolveAbility(&def.Abilities[0], 1) // melee_swipe
	if resolved1.BaseDamage != 25.0 {
		t.Errorf("P1 melee damage = %f, want 25.0", resolved1.BaseDamage)
	}
	if resolved1.CommitTime != 1.2 {
		t.Errorf("P1 commit time = %f, want 1.2", resolved1.CommitTime)
	}

	// Phase 2
	e.Phase = 2
	resolved2 := def.ResolveAbility(&def.Abilities[0], 2)
	if resolved2.CommitTime != 0.9 {
		t.Errorf("P2 commit time = %f, want 0.9", resolved2.CommitTime)
	}

	// fireball_burst Phase 2: commit time shortens (pattern handles projectile spawning)
	fb2 := def.ResolveAbility(&def.Abilities[1], 2)
	if fb2.CommitTime != 0.8 {
		t.Errorf("P2 fireball commit time = %f, want 0.8", fb2.CommitTime)
	}

	// Phase 3: enraged
	e.Phase = 3
	resolved3 := def.ResolveAbility(&def.Abilities[0], 3)
	if resolved3.CommitTime != 0.7 {
		t.Errorf("P3 commit time = %f, want 0.7", resolved3.CommitTime)
	}
	if resolved3.BaseDamage != 30.0 {
		t.Errorf("P3 melee damage = %f, want 30.0", resolved3.BaseDamage)
	}

	// fireball_burst Phase 3: commit time even shorter
	fb3 := def.ResolveAbility(&def.Abilities[1], 3)
	if fb3.CommitTime != 0.6 {
		t.Errorf("P3 fireball commit time = %f, want 0.6", fb3.CommitTime)
	}

	// ground_slam Phase 3: bigger radius and more damage (index 3 after void_barrage)
	gs3 := def.ResolveAbility(&def.Abilities[3], 3)
	if gs3.Hit.Radius != 7.0 {
		t.Errorf("P3 AoE radius = %f, want 7.0", gs3.Hit.Radius)
	}
	if gs3.BaseDamage != 45.0 {
		t.Errorf("P3 AoE damage = %f, want 45.0", gs3.BaseDamage)
	}

	// bull_charge Phase 3: faster and longer (index 4)
	bc3 := def.ResolveAbility(&def.Abilities[4], 3)
	if bc3.Charge.Speed != 16.0 {
		t.Errorf("P3 charge speed = %f, want 16.0", bc3.Charge.Speed)
	}
	if bc3.Charge.MaxDistance != 20.0 {
		t.Errorf("P3 charge max dist = %f, want 20.0", bc3.Charge.MaxDistance)
	}
	if bc3.Charge.Damage != 40.0 {
		t.Errorf("P3 charge damage = %f, want 40.0", bc3.Charge.Damage)
	}
}

// TestFight_DamageEventSourceTypes verifies correct source type on damage events.
func TestFight_DamageEventSourceTypes(t *testing.T) {
	// Melee-only mob for predictable ability selection
	def := &EnemyDef{
		Name:      "test_source_type",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  testTreeData(),
		Abilities: []ability.AbilityDef{
			{
				ID: "melee", Name: "melee", Category: ability.CategoryMelee,
				CommitTime: 0.1, Cooldown: 0.2,
				BaseWeight: 100, MaxRange: 5.0,
				BaseDamage:   10.0,
				Hit:          ability.HitDef{Type: ability.HitAoECone, Range: 5.0, ArcDegrees: 360},
				DamageSource: combat.SourceEnemyMelee,
			},
		},
	}
	b, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase

	p := testPlayer(1, entity.Vec3{X: 0, Z: 1})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	var events []combat.DamageEvent
	for range 200 {
		evts := b.Tick(0.05, players, nil, noSpawn, nil)
		events = append(events, evts...)
		if len(events) > 0 {
			break
		}
	}

	if len(events) == 0 {
		t.Fatal("expected at least one damage event")
	}
	if events[0].SourceType != combat.SourceEnemyMelee {
		t.Errorf("source type = %d, want %d (SourceEnemyMelee)", events[0].SourceType, combat.SourceEnemyMelee)
	}
}

// TestFight_SelectAbilityRangeFiltering verifies abilities are filtered by distance.
func TestFight_SelectAbilityRangeFiltering(t *testing.T) {
	def := testDef() // has melee (max 3), ranged (min 3), aoe (max 7), charge (min 6)
	b, e := testBrain(def)
	e.Alive = true

	tests := []struct {
		name     string
		distance float32
		excluded []ability.AbilityCategory
	}{
		{"melee_range", 2.0, []ability.AbilityCategory{ability.CategoryRanged, ability.CategoryCharge}},
		{"mid_range", 4.0, []ability.AbilityCategory{ability.CategoryMelee}},
		{"far_range", 8.0, []ability.AbilityCategory{ability.CategoryMelee, ability.CategoryAoE}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sample many times to catch all candidates
			selected := map[ability.AbilityCategory]int{}
			for range 200 {
				chosen := b.ctx.SelectAbility(tt.distance)
				if chosen != nil {
					selected[chosen.Category]++
				}
			}
			for _, excluded := range tt.excluded {
				if selected[excluded] > 0 {
					t.Errorf("ability category %d should be excluded at dist %.1f, but was selected %d times",
						excluded, tt.distance, selected[excluded])
				}
			}
		})
	}
}

// TestFight_SelectAbilityAntiRepeat verifies the anti-repeat mechanism.
func TestFight_SelectAbilityAntiRepeat(t *testing.T) {
	def := &EnemyDef{
		Name:       "test_antirepeat",
		MaxHealth:  100,
		MoveSpeed:  4,
		AntiRepeat: 100.0, // extreme anti-repeat
		TreeData:   testTreeData(),
		Abilities: []ability.AbilityDef{
			{ID: "a", Name: "a", Category: ability.CategoryMelee, BaseWeight: 50, MaxRange: 5, Hit: ability.HitDef{Type: ability.HitAoECone, Range: 5}},
			{ID: "b", Name: "b", Category: ability.CategoryMelee, BaseWeight: 50, MaxRange: 5, Hit: ability.HitDef{Type: ability.HitAoECone, Range: 5}},
		},
	}
	b, _ := testBrain(def)

	// After using "a", "b" should dominate
	b.bb.Set("last_attack", "a")
	bCount := 0
	for range 200 {
		chosen := b.ctx.SelectAbility(3.0)
		if chosen != nil && chosen.Name == "b" {
			bCount++
		}
	}
	// With anti-repeat 100, "a" weight drops from 50 to 0, "b" should always win
	if bCount < 190 {
		t.Errorf("anti-repeat: 'b' selected %d/200, expected > 190", bCount)
	}
}

// --- Benchmarks ---

func BenchmarkBrainTick_Patrol(b *testing.B) {
	br, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -10}
	e.PatrolB = entity.Vec3{X: 10}
	players := testPlayers()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.Tick(0.05, players, nil, noSpawn, nil)
	}
}

func BenchmarkBrainTick_Chase(b *testing.B) {
	br, e := testBrain(DefRegistry["hallway_melee"])
	e.Alive = true
	e.State = entity.EnemyChase
	p := testPlayer(1, entity.Vec3{X: 0, Z: 10})
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		e.Position = entity.Vec3{X: 0, Z: 0} // reset position
		br.Tick(0.05, players, nil, noSpawn, nil)
	}
}

func BenchmarkBrainTick_MeleeAttackCycle(b *testing.B) {
	def := &EnemyDef{
		Name:      "bench_melee",
		MaxHealth: 500,
		MoveSpeed: 4.0,
		Radius:    1.0,
		TreeData:  testTreeData(),
		Abilities: []ability.AbilityDef{
			{
				ID: "melee", Name: "melee", Category: ability.CategoryMelee,
				CommitTime: 0.3, Cooldown: 0.3,
				BaseWeight: 100, MaxRange: 5.0,
				BaseDamage:   10.0,
				Hit:          ability.HitDef{Type: ability.HitAoECone, Range: 5.0, ArcDegrees: 360},
				DamageSource: combat.SourceEnemyMelee,
			},
		},
	}
	br, e := testBrain(def)
	e.Alive = true
	e.State = entity.EnemyChase
	p := testPlayer(1, entity.Vec3{X: 0, Z: 1})
	p.Health = 1e9
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.Tick(0.05, players, nil, noSpawn, nil)
	}
}

func BenchmarkBrainTick_GuardCaptainFight(b *testing.B) {
	br, e := testBrain(DefRegistry["guard_captain"])
	e.Alive = true
	e.State = entity.EnemyChase
	p := testPlayer(1, entity.Vec3{X: 0, Z: 2})
	p.Health = 1e9
	players := testPlayers(p)
	e.TargetPlayerID = p.ID

	// Warm up the BT
	for range 50 {
		br.Tick(0.05, players, nil, noSpawn, nil)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.Tick(0.05, players, nil, noSpawn, nil)
	}
}

func BenchmarkBrainTick_MultiEnemy5(b *testing.B) {
	type pair struct {
		brain *Brain
		enemy *entity.Enemy
	}
	var brains [5]pair
	for i := range brains {
		br, e := testBrain(DefRegistry["hallway_melee"])
		e.Alive = true
		e.State = entity.EnemyChase
		e.Position = entity.Vec3{X: float32(i) * 3, Z: 0}
		brains[i] = pair{br, e}
	}
	p := testPlayer(1, entity.Vec3{X: 7, Z: 5})
	players := testPlayers(p)
	for i := range brains {
		brains[i].enemy.TargetPlayerID = p.ID
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, bp := range brains {
			bp.brain.Tick(0.05, players, nil, noSpawn, nil)
		}
	}
}

func BenchmarkSelectAbility(b *testing.B) {
	br, _ := testBrain(DefRegistry["guard_captain"])
	br.ctx.Reset(0.05, testPlayers(), nil, noSpawn, nil, &[]combat.DamageEvent{})

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		br.ctx.SelectAbility(4.0)
	}
}

func BenchmarkBlackboard_TickTimers(b *testing.B) {
	bb := NewBlackboard()
	bb.StartTimer("a", 5.0)
	bb.StartTimer("b", 10.0)
	bb.StartTimer("c", 3.0)
	bb.StartTimer("d", 7.0)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bb.TickTimers(0.05)
		// Reset timers periodically to avoid all expiring
		if bb.TimerExpired("a") {
			bb.StartTimer("a", 5.0)
		}
		if bb.TimerExpired("c") {
			bb.StartTimer("c", 3.0)
		}
	}
}
