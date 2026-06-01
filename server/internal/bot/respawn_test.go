package bot

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/level"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
)

// testWorldWithLevel creates a world with a level that has spawn points,
// simulating an instanced dungeon.
func testWorldWithLevel() *system.World {
	w := &system.World{
		Players:       make(map[uint16]*entity.Player),
		Clients:       make(map[uint16]*system.Client),
		AbilityEngine: ability.NewEngine(nil),
		TickNum:       100,
		State:         system.StateFight,
		Level: &level.Level{
			PlayerSpawns: []level.PlayerSpawn{
				{Position: entity.Vec3{X: 0, Y: 0, Z: 50}}, // spawn point
			},
			Gates: []level.GateDef{
				{
					ID:       "boss_gate",
					Position: entity.Vec3{X: 0, Y: 2.5, Z: 12},
					CloseOn:  []string{"boss_activated"},
					OpenOn:   []string{"boss_dead", "all_dead", "boss_reset"},
				},
			},
		},
	}
	w.InitGateStates()
	return w
}

func TestBotRespawnAfterTrashDeath(t *testing.T) {
	// Scenario: bot dies fighting trash mobs during StateFight (no gate closed).
	// Expected: bot respawns after BotRespawnDelay at spawn point.
	m := NewManager("")
	w := testWorldWithLevel()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	owner.Position = entity.Vec3{X: 10, Y: 0, Z: 10}
	w.Players[1] = owner

	botID, err := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	if err != nil {
		t.Fatalf("SpawnBot: %v", err)
	}

	// Kill the bot
	bot := w.Players[botID]
	bot.Alive = false
	bot.Health = 0

	// No gate closed (trash fight, not boss encounter)

	// Tick through the respawn delay.
	dt := float32(0.05) // 20Hz
	ticksNeeded := int(BotRespawnDelay/dt) + 1
	inputSys := &system.InputSystem{}

	for range ticksNeeded {
		w.InputQueue = w.InputQueue[:0]
		m.TickAll(w, dt)
		inputSys.Tick(w, dt)
	}

	if !bot.Alive {
		t.Errorf("bot should have respawned after %.1fs (state=%d, anyGateClosed=%v)",
			BotRespawnDelay, w.State, w.AnyGateClosed())
	}
}

func TestBotNoRespawnDuringBossFight(t *testing.T) {
	// Scenario: bot dies during boss fight (boss gate IS closed).
	// Expected: bot does NOT respawn.
	m := NewManager("")
	w := testWorldWithLevel()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	w.Players[1] = owner

	botID, err := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	if err != nil {
		t.Fatalf("SpawnBot: %v", err)
	}

	bot := w.Players[botID]
	bot.Alive = false
	bot.Health = 0

	// Boss gate closed = boss encounter sealed
	w.GateStates["boss_gate"] = true
	w.RebuildObstacles()

	dt := float32(0.05)
	ticksNeeded := int(BotRespawnDelay/dt) + 20 // well past the delay
	inputSys := &system.InputSystem{}

	for range ticksNeeded {
		w.InputQueue = w.InputQueue[:0]
		m.TickAll(w, dt)
		inputSys.Tick(w, dt)
	}

	if bot.Alive {
		t.Error("bot should NOT respawn during boss fight (gate closed)")
	}
}

func TestBotRespawnAfterWipe(t *testing.T) {
	// Scenario: wipe happened, state is FightOver, bot should respawn.
	m := NewManager("")
	w := testWorldWithLevel()
	w.State = system.StateFightOver

	owner := entity.NewPlayer(1, entity.ClassGunner)
	owner.Alive = true
	w.Players[1] = owner

	botID, err := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	if err != nil {
		t.Fatalf("SpawnBot: %v", err)
	}

	bot := w.Players[botID]
	bot.Alive = false
	bot.Health = 0

	dt := float32(0.05)
	ticksNeeded := int(BotRespawnDelay/dt) + 1
	inputSys := &system.InputSystem{}

	for range ticksNeeded {
		w.InputQueue = w.InputQueue[:0]
		m.TickAll(w, dt)
		inputSys.Tick(w, dt)
	}

	if !bot.Alive {
		t.Error("bot should respawn after wipe (StateFightOver)")
	}
}

func TestBotRespawnUsesSpawnPoint(t *testing.T) {
	// Verify bot respawns at a spawn point, not at owner position.
	m := NewManager("")
	w := testWorldWithLevel()
	w.State = system.StateFightOver

	spawnPos := w.Level.PlayerSpawns[0].Position

	owner := entity.NewPlayer(1, entity.ClassGunner)
	owner.Position = entity.Vec3{X: 99, Y: 0, Z: 99} // far from spawn
	w.Players[1] = owner

	botID, err := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	if err != nil {
		t.Fatalf("SpawnBot: %v", err)
	}

	bot := w.Players[botID]
	bot.Alive = false
	bot.Health = 0

	dt := float32(0.05)
	ticksNeeded := int(BotRespawnDelay/dt) + 1
	inputSys := &system.InputSystem{}

	// Tick until the respawn request is queued, then run InputSystem once to process it.
	for range ticksNeeded {
		w.InputQueue = w.InputQueue[:0]
		m.TickAll(w, dt)
		// Only run InputSystem, do NOT tick bots again (which would follow-teleport to owner).
		inputSys.Tick(w, dt)
		if bot.Alive {
			break
		}
	}

	if !bot.Alive {
		t.Fatal("bot should have respawned")
	}

	// Bot should have been placed at the level spawn point by handleRespawnRequest.
	dx := bot.Position.X - spawnPos.X
	dz := bot.Position.Z - spawnPos.Z
	if dx*dx+dz*dz > 1.0 {
		t.Errorf("bot respawned at (%.1f, %.1f) instead of spawn point (%.1f, %.1f)",
			bot.Position.X, bot.Position.Z, spawnPos.X, spawnPos.Z)
	}
}

func TestBotRespawnRequestQueued(t *testing.T) {
	// Low-level test: verify the respawn request opcode is actually queued.
	m := NewManager("")
	w := testWorldWithLevel()
	w.State = system.StateFightOver

	owner := entity.NewPlayer(1, entity.ClassGunner)
	w.Players[1] = owner

	botID, err := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	if err != nil {
		t.Fatalf("SpawnBot: %v", err)
	}

	bot := w.Players[botID]
	bot.Alive = false
	bot.Health = 0

	// Tick through most of the delay without clearing the queue on the final iteration.
	dt := float32(0.05)
	ticksNeeded := int(BotRespawnDelay/dt) + 1

	for range ticksNeeded {
		w.InputQueue = w.InputQueue[:0]
		m.TickAll(w, dt)
		// Check after each tick if the request was queued
		for _, inp := range w.InputQueue {
			if inp.PeerID == botID && inp.Opcode == message.OpRespawnRequest {
				return // success
			}
		}
	}

	t.Errorf("expected OpRespawnRequest for bot %d in InputQueue after %d ticks", botID, ticksNeeded)
}
