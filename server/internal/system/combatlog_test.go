package system

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/combat"
	"codex-online/server/internal/combatlog"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/entity"
)

const testEncounterID = "guard_captain"

// makeLoggedWorld creates a world with an InMemorySink for combat log testing.
func makeLoggedWorld(t testing.TB, players map[uint16]*entity.Player, enemies []*entity.Enemy) (*World, *combatlog.InMemorySink) {
	sink := combatlog.NewInMemorySink()
	session := combatlog.NewSession(
		sink, "test-instance", "test-group", "test-encounter", testArenaZoneID, "run-1",
		0, combatlog.SourceSimulation, 100,
	)
	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1, // arena
		TickNum:       100,
		Players:       players,
		Enemies:       enemies,
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
		CombatLogSink: sink,
		CombatLogs:    map[int]*combatlog.EncounterSession{0: session},
	}
	return w, sink
}

// --- Player ability → damage + commit_start ---

func TestCombatLog_PlayerAbilityDamage(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	e := entity.NewEnemy(1000, 2000.0, testEncounterID)
	e.Alive = true
	// Position enemy directly in front of player for hitscan
	p.Position = entity.Vec3{X: 0, Z: 5, Y: 0.1}
	p.RotationY = 0 // facing -Z
	e.Position = entity.Vec3{X: 0, Z: 3, Y: 0.1}

	w, sink := makeLoggedWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: codec.EncodeAbilityInput(entity.ActionShoot, 0.0)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Should have commit_start
	commits := sink.EventsOfType(combatlog.EventCommitStart)
	if len(commits) == 0 {
		t.Error("expected at least one commit_start event")
	} else {
		if commits[0].SourceEntity != "player_1" {
			t.Errorf("commit_start Source = %s, want player_1", commits[0].SourceEntity)
		}
		if commits[0].AbilityID == "" {
			t.Error("commit_start AbilityID should not be empty")
		}
	}

	// Should have damage events (if hit resolved)
	dmg := sink.EventsOfType(combatlog.EventDamage)
	if len(dmg) > 0 {
		if dmg[0].SourceEntity != "player_1" {
			t.Errorf("damage Source = %s, want player_1", dmg[0].SourceEntity)
		}
		if dmg[0].Target != "enemy_1000" {
			t.Errorf("damage Target = %s, want enemy_1000", dmg[0].Target)
		}
		if dmg[0].Amount <= 0 {
			t.Error("damage Amount should be > 0")
		}
	}
}

// --- Dodge ---

func TestCombatLog_Dodge(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	w, sink := makeLoggedWorld(t, map[uint16]*entity.Player{1: p}, nil)

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: codec.EncodeAbilityInput(entity.ActionDodge, 0.0)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	dodges := sink.EventsOfType(combatlog.EventDodge)
	if len(dodges) != 1 {
		t.Fatalf("dodge events = %d, want 1", len(dodges))
	}
	if dodges[0].SourceEntity != "player_1" {
		t.Errorf("dodge Source = %s, want player_1", dodges[0].SourceEntity)
	}
	if dodges[0].SourceClass != entity.ClassVanguard {
		t.Errorf("dodge SourceClass = %s, want vanguard", dodges[0].SourceClass)
	}
}

// --- DoT tick → buff_tick ---

func TestCombatLog_DoTTick(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassVanguard)
	e := entity.NewEnemy(1000, 2000.0, testEncounterID)
	e.Alive = true
	e.State = entity.EnemyChase
	e.AddThreat(1, 10)

	// Give player an active DoT on the enemy
	p.DoTs = append(p.DoTs, entity.ActiveDoT{
		EnemyID:    1000,
		SourcePeer: 1,
		Damage:     25,
		Remaining:  5.0,
		Interval:   1.0,
		TickTimer:  0.01, // ticks almost immediately
	})

	w, sink := makeLoggedWorld(t, map[uint16]*entity.Player{1: p}, []*entity.Enemy{e})
	cs := &CombatSystem{}
	cs.Tick(w, 0.05)

	ticks := sink.EventsOfType(combatlog.EventBuffTick)
	if len(ticks) == 0 {
		t.Error("expected at least one buff_tick event from DoT")
	} else {
		if ticks[0].Amount <= 0 {
			t.Error("buff_tick Amount should be > 0")
		}
		if ticks[0].Target != "enemy_1000" {
			t.Errorf("buff_tick Target = %s, want enemy_1000", ticks[0].Target)
		}
	}
}

// --- Buff expiry → buff_remove ---

func TestCombatLog_BuffExpiry(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	// Give player a buff about to expire
	p.Buffs = append(p.Buffs, entity.ActiveBuff{
		ID:       ability.IDOverclock,
		Type:     "damage_mult",
		Value:    1.5,
		Duration: 0.01, // expires this tick
	})

	w, sink := makeLoggedWorld(t, map[uint16]*entity.Player{1: p}, nil)
	cs := &CombatSystem{}
	cs.Tick(w, 0.05)

	removes := sink.EventsOfType(combatlog.EventBuffRemove)
	if len(removes) != 1 {
		t.Fatalf("buff_remove events = %d, want 1", len(removes))
	}
	if removes[0].AbilityID != ability.IDOverclock {
		t.Errorf("buff_remove AbilityID = %s, want overclock", removes[0].AbilityID)
	}
}

// --- Fight lifecycle ---

func TestCombatLog_FightLifecycle_BossKill(t *testing.T) {
	sink := combatlog.NewInMemorySink()
	p := entity.NewPlayer(1, entity.ClassGunner)
	e := entity.NewEnemy(1000, 100.0, "test_boss")
	e.Alive = true
	e.IsBoss = true
	e.GroupID = 3
	e.State = entity.EnemyPatrol

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       50,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e},
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
		CombatLogSink: sink,
	}
	p.Position = w.Level.PlayerSpawns[0].Position

	// Aggro the boss — this starts the per-group combat log
	w.AggroEnemy(e, 1)
	if len(w.CombatLogs) == 0 {
		t.Fatal("CombatLogs should be set after boss aggro")
	}

	// Kill the boss
	e.State = entity.EnemyDead
	e.Alive = false

	w.TickNum = 200
	gf := &GameFlowSystem{}
	gf.Tick(w, 0.05)

	if len(w.CombatLogs) != 0 {
		t.Error("CombatLogs should be empty after fight end")
	}

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}
	if instances[0].Outcome != combatlog.OutcomePlayerWin {
		t.Errorf("outcome = %s, want player_win", instances[0].Outcome)
	}
	if len(instances[0].Participants) < 2 {
		t.Errorf("participants = %d, want >= 2 (player + boss)", len(instances[0].Participants))
	}
}

func TestCombatLog_FightLifecycle_Wipe(t *testing.T) {
	sink := combatlog.NewInMemorySink()
	p := entity.NewPlayer(1, entity.ClassGunner)
	e := entity.NewEnemy(1000, 2000.0, "test_boss")
	e.Alive = true
	e.IsBoss = true
	e.GroupID = 3
	e.State = entity.EnemyPatrol

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       50,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e},
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
		CombatLogSink: sink,
	}
	p.Position = w.Level.PlayerSpawns[0].Position

	// Aggro the boss to start the combat log
	w.AggroEnemy(e, 1)
	// Set boss to idle to avoid boss gate logic
	e.State = entity.EnemyIdle

	// Kill the player
	p.Alive = false
	p.State = entity.PlayerStateDead

	w.TickNum = 200
	gf := &GameFlowSystem{}
	gf.Tick(w, 0.05)

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}
	if instances[0].Outcome != combatlog.OutcomeBossWin {
		t.Errorf("outcome = %s, want boss_win", instances[0].Outcome)
	}
}

// --- Solo boss (GroupID=0) uses synthetic negative key ---

func TestCombatLog_SoloBoss_NegativeKey(t *testing.T) {
	sink := combatlog.NewInMemorySink()
	p := entity.NewPlayer(1, entity.ClassGunner)
	e := entity.NewEnemy(1000, 100.0, testEncounterID)
	e.Alive = true
	e.IsBoss = true
	// GroupID=0 (default) — matches real arena boss
	e.State = entity.EnemyPatrol

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       50,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e},
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
		CombatLogSink: sink,
	}
	p.Position = w.Level.PlayerSpawns[0].Position

	// Aggro → should use synthetic key -1000
	w.AggroEnemy(e, 1)
	if w.CombatLogs[-1000] == nil {
		t.Fatal("CombatLogs[-1000] should exist for solo boss")
	}

	// Kill the boss
	e.State = entity.EnemyDead
	e.Alive = false
	checkEnemyGroupDead(w, e)

	if w.CombatLogs[-1000] != nil {
		t.Error("CombatLogs[-1000] should be removed after finalize")
	}

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}
	if instances[0].EncounterID != testEncounterID {
		t.Errorf("encounterID = %s, want guard_captain", instances[0].EncounterID)
	}
	if instances[0].Outcome != combatlog.OutcomePlayerWin {
		t.Errorf("outcome = %s, want player_win", instances[0].Outcome)
	}
}

// --- No logging outside fights ---

func TestCombatLog_NoLogWhenSessionNil(t *testing.T) {
	t.Helper()
	p := entity.NewPlayer(1, entity.ClassGunner)
	w := makeWorld(t, map[uint16]*entity.Player{1: p}, nil)
	// CombatLog is nil (no session active)

	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: codec.EncodeAbilityInput(entity.ActionDodge, 0.0)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Should not panic — logCombatEvent is a no-op when CombatLog is nil
}

// --- Proximity aggro triggers combat log ---

// TestCombatLog_ProximityAggro_Boss verifies the real game flow: boss aggros
// via proximity in the AI brain (not via AggroEnemy), and the combat log
// session starts correctly. This is the scenario that was broken — the brain
// sets e.State = EnemyChase directly, bypassing AggroEnemy.
func TestCombatLog_ProximityAggro_Boss(t *testing.T) {
	sink := combatlog.NewInMemorySink()

	// Create a boss with GroupID=0 (solo boss, like guard_captain in arena)
	def := enemyai.DefRegistry[testEncounterID]
	if def == nil {
		t.Fatal("guard_captain def not found in registry")
	}
	e := entity.NewEnemy(1000, def.MaxHealth, testEncounterID)
	e.Alive = true
	e.IsBoss = true
	e.AggroRadius = 10
	e.LeashRadius = 30
	e.State = entity.EnemyPatrol
	e.PatrolA = entity.Vec3{X: -5, Y: 0.1, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Y: 0.1, Z: 0}
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	e.LeashOrigin = e.Position

	eng := ability.NewEngine(nil)
	brain := enemyai.NewBrain(def, e, eng)

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true

	lvl := testArenaLevel(t)

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       100,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e},
		Brains:        []enemyai.BrainTicker{brain},
		Level:         lvl,
		AbilityEngine: eng,
		CombatLogSink: sink,
		PatternEngine: combat.NewPatternEngine(),
	}

	// Place player close enough to trigger proximity aggro
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5}

	// Before AI tick: no combat log
	if len(w.CombatLogs) != 0 {
		t.Fatal("CombatLogs should be empty before aggro")
	}

	// Tick the AI system — brain should detect player in aggro range
	// and transition boss from Patrol to Chase
	ai := &AISystem{}
	ai.Tick(w, 0.05)

	// Boss should have left patrol
	if e.State == entity.EnemyPatrol {
		t.Fatal("boss should have aggroed (left patrol)")
	}

	// Combat log session should have started
	if w.CombatLogs[-1000] == nil {
		t.Fatal("CombatLogs[-1000] should exist for solo boss after proximity aggro")
	}

	// Verify encounter ID is the boss name, not the zone
	// (We can't read encounterID from the session directly, but we can
	// finalize and check the instance log)
	e.State = entity.EnemyDead
	e.Alive = false
	checkEnemyGroupDead(w, e)

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}
	if instances[0].EncounterID != testEncounterID {
		t.Errorf("encounterID = %s, want guard_captain", instances[0].EncounterID)
	}
	if instances[0].MobGroupID != 1000 {
		t.Errorf("mobGroupID = %d, want 1000 (boss enemy ID)", instances[0].MobGroupID)
	}
	if instances[0].Outcome != combatlog.OutcomePlayerWin {
		t.Errorf("outcome = %s, want player_win", instances[0].Outcome)
	}
}

// TestCombatLog_ProximityAggro_TrashPack verifies that trash packs with a
// GroupID start a combat log session when they aggro via proximity.
func TestCombatLog_ProximityAggro_TrashPack(t *testing.T) {
	sink := combatlog.NewInMemorySink()

	def := enemyai.DefRegistry["hallway_melee"]
	if def == nil {
		t.Fatal("hallway_melee def not found in registry")
	}

	e1 := entity.NewEnemy(1000, def.MaxHealth, "hallway_melee")
	e1.Alive = true
	e1.GroupID = 1
	e1.AggroRadius = 10
	e1.LeashRadius = 40
	e1.State = entity.EnemyPatrol
	e1.Position = entity.Vec3{X: -3, Y: 0.1, Z: 32}
	e1.PatrolA = entity.Vec3{X: -6, Y: 0.1, Z: 32}
	e1.PatrolB = entity.Vec3{X: 6, Y: 0.1, Z: 32}
	e1.LeashOrigin = e1.Position

	e2 := entity.NewEnemy(1001, def.MaxHealth, "hallway_melee")
	e2.Alive = true
	e2.GroupID = 1
	e2.AggroRadius = 10
	e2.LeashRadius = 40
	e2.State = entity.EnemyPatrol
	e2.Position = entity.Vec3{X: 3, Y: 0.1, Z: 32}
	e2.PatrolA = entity.Vec3{X: 6, Y: 0.1, Z: 32}
	e2.PatrolB = entity.Vec3{X: -6, Y: 0.1, Z: 32}
	e2.LeashOrigin = e2.Position

	eng := ability.NewEngine(nil)
	b1 := enemyai.NewBrain(def, e1, eng)
	b2 := enemyai.NewBrain(def, e2, eng)

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 35} // within aggro range

	lvl := testArenaLevel(t)

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       100,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e1, e2},
		Brains:        []enemyai.BrainTicker{b1, b2},
		Level:         lvl,
		AbilityEngine: eng,
		CombatLogSink: sink,
		PatternEngine: combat.NewPatternEngine(),
	}

	ai := &AISystem{}
	ai.Tick(w, 0.05)

	// At least one enemy should have aggroed
	anyAggroed := e1.State != entity.EnemyPatrol || e2.State != entity.EnemyPatrol
	if !anyAggroed {
		t.Fatal("at least one enemy should have aggroed from proximity")
	}

	// Combat log for group 1 should exist
	if w.CombatLogs[1] == nil {
		t.Fatal("CombatLogs[1] should exist after trash pack aggro")
	}

	// Kill both enemies
	e1.State = entity.EnemyDead
	e1.Alive = false
	e2.State = entity.EnemyDead
	e2.Alive = false
	checkEnemyGroupDead(w, e1)

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}
	if instances[0].EncounterID != "hallway_melee" {
		t.Errorf("encounterID = %s, want hallway_melee", instances[0].EncounterID)
	}
	if instances[0].MobGroupID != 1 {
		t.Errorf("mobGroupID = %d, want 1", instances[0].MobGroupID)
	}
}

// TestCombatLog_DamageAggroAlsoWorks verifies that the AggroEnemy path
// (player hits an enemy still in patrol) also starts the combat log.
func TestCombatLog_DamageAggroAlsoWorks(t *testing.T) {
	sink := combatlog.NewInMemorySink()
	e := entity.NewEnemy(1000, 100.0, testEncounterID)
	e.Alive = true
	e.IsBoss = true
	e.State = entity.EnemyPatrol

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		TickNum:       100,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e},
		Level:         testArenaLevel(t),
		AbilityEngine: ability.NewEngine(nil),
		CombatLogSink: sink,
	}
	p.Position = w.Level.PlayerSpawns[0].Position

	// Direct AggroEnemy (simulates player hitting boss while still patrolling)
	w.AggroEnemy(e, 1)

	if w.CombatLogs[-1000] == nil {
		t.Fatal("CombatLogs[-1000] should exist after AggroEnemy")
	}
}

// TestCombatLog_FullFight_EndToEnd exercises the complete flow:
// AI proximity aggro → combat log starts → events logged → boss dies → finalized
func TestCombatLog_FullFight_EndToEnd(t *testing.T) {
	sink := combatlog.NewInMemorySink()

	def := enemyai.DefRegistry[testEncounterID]
	if def == nil {
		t.Fatal("guard_captain def not found")
	}
	e := entity.NewEnemy(1000, def.MaxHealth, testEncounterID)
	e.Alive = true
	e.IsBoss = true
	e.AggroRadius = 10
	e.LeashRadius = 30
	e.State = entity.EnemyPatrol
	e.Position = entity.Vec3{X: 0, Y: 0.1, Z: 0}
	e.PatrolA = entity.Vec3{X: -5, Y: 0.1, Z: 0}
	e.PatrolB = entity.Vec3{X: 5, Y: 0.1, Z: 0}
	e.LeashOrigin = e.Position

	eng := ability.NewEngine(nil)
	brain := enemyai.NewBrain(def, e, eng)

	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Alive = true
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 5} // close to boss

	lvl := testArenaLevel(t)

	w := &World{
		ZoneID:        testArenaZoneID,
		ZoneType:      1,
		RunID:         "run-test-1",
		TickNum:       100,
		Players:       map[uint16]*entity.Player{1: p},
		Enemies:       []*entity.Enemy{e},
		Brains:        []enemyai.BrainTicker{brain},
		Level:         lvl,
		AbilityEngine: eng,
		CombatLogSink: sink,
		PatternEngine: combat.NewPatternEngine(),
	}

	// Phase 1: AI tick triggers proximity aggro → combat log starts
	ai := &AISystem{}
	ai.Tick(w, 0.05)

	if len(w.CombatLogs) == 0 {
		t.Fatal("CombatLogs should be active after AI proximity aggro")
	}

	// Phase 2: Player shoots boss (simulate ability input)
	p.RotationY = 0 // facing -Z (toward boss)
	w.TickNum = 110
	w.InputQueue = []InputMsg{{PeerID: 1, Opcode: 0x0031, Payload: codec.EncodeAbilityInput(entity.ActionShoot, 0.0)}}
	is := &InputSystem{}
	is.Tick(w, 0.05)

	// Should have logged events
	events := sink.Events()
	if len(events) == 0 {
		t.Fatal("expected combat events after shooting boss")
	}

	// Verify events have correct metadata
	for _, ev := range events {
		if ev.InstanceID == "" {
			t.Error("event InstanceID should not be empty")
		}
		if ev.EncounterID != testEncounterID {
			t.Errorf("event EncounterID = %s, want guard_captain", ev.EncounterID)
		}
	}

	// Phase 3: Kill boss → session finalized
	e.State = entity.EnemyDead
	e.Alive = false
	w.TickNum = 200
	gf := &GameFlowSystem{}
	gf.Tick(w, 0.05)

	if len(w.CombatLogs) != 0 {
		t.Error("CombatLogs should be empty after boss killed")
	}

	instances := sink.Instances()
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}

	inst := instances[0]
	if inst.EncounterID != testEncounterID {
		t.Errorf("encounterID = %s, want guard_captain", inst.EncounterID)
	}
	if inst.MobGroupID != 1000 {
		t.Errorf("mobGroupID = %d, want 1000", inst.MobGroupID)
	}
	if inst.Outcome != combatlog.OutcomePlayerWin {
		t.Errorf("outcome = %s, want player_win", inst.Outcome)
	}
	if inst.ZoneID != testArenaZoneID {
		t.Errorf("zoneID = %s, want test-arena", inst.ZoneID)
	}
	if inst.RunID != "run-test-1" {
		t.Errorf("runID = %s, want run-test-1", inst.RunID)
	}
	if len(inst.Participants) < 2 {
		t.Errorf("participants = %d, want >= 2 (player + boss)", len(inst.Participants))
	}

	// Verify events reference the instance
	castEvents := sink.EventsOfType(combatlog.EventCommitStart)
	if len(castEvents) == 0 {
		t.Error("expected commit_start events")
	}
}
