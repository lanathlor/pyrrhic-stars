package bot

import (
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/system"
)

func testWorld() *system.World {
	return &system.World{
		Players:       make(map[uint16]*entity.Player),
		Clients:       make(map[uint16]*system.Client),
		AbilityEngine: ability.NewEngine(nil),
		TickNum:       100,
	}
}

func TestSpawnBot(t *testing.T) {
	m := NewManager("")
	w := testWorld()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	owner.Position = entity.Vec3{X: 5, Y: 0, Z: 10}
	w.Players[1] = owner

	botID, err := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	if err != nil {
		t.Fatalf("SpawnBot: %v", err)
	}
	if !entity.IsBotID(botID) {
		t.Errorf("bot ID %d should be in bot range", botID)
	}
	if w.Players[botID] == nil {
		t.Fatal("bot not in world players")
	}
	p := w.Players[botID]
	if p.ClassID != entity.ClassVanguard {
		t.Errorf("class = %q, want vanguard", p.ClassID)
	}
	if p.SpecID != "blade" {
		t.Errorf("spec = %q, want blade", p.SpecID)
	}
	if !p.Ready {
		t.Error("bot should be ready")
	}
	if m.BotCount(1) != 1 {
		t.Errorf("bot count = %d, want 1", m.BotCount(1))
	}
}

func TestSpawnBotMaxLimit(t *testing.T) {
	m := NewManager("")
	w := testWorld()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	w.Players[1] = owner

	for i := range MaxBotsPerPlayer {
		_, err := m.SpawnBot(1, entity.ClassGunner, "assault", w)
		if err != nil {
			t.Fatalf("SpawnBot %d: %v", i, err)
		}
	}

	_, err := m.SpawnBot(1, entity.ClassGunner, "assault", w)
	if err == nil {
		t.Error("expected error when exceeding max bots")
	}
}

func TestDismissBot(t *testing.T) {
	m := NewManager("")
	w := testWorld()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	w.Players[1] = owner

	botID, _ := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	m.DismissBot(botID, w)

	if w.Players[botID] != nil {
		t.Error("bot still in world players after dismiss")
	}
	if m.BotCount(1) != 0 {
		t.Errorf("bot count = %d, want 0", m.BotCount(1))
	}
}

func TestDismissAllForOwner(t *testing.T) {
	m := NewManager("")
	w := testWorld()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	w.Players[1] = owner

	id1, _ := m.SpawnBot(1, entity.ClassGunner, "assault", w)
	id2, _ := m.SpawnBot(1, entity.ClassVanguard, "blade", w)

	m.DismissAllForOwner(1, w)

	if w.Players[id1] != nil || w.Players[id2] != nil {
		t.Error("bots still in world after dismiss all")
	}
	if m.BotCount(1) != 0 {
		t.Errorf("bot count = %d, want 0", m.BotCount(1))
	}
}

func TestIsBotID(t *testing.T) {
	if entity.IsBotID(1) {
		t.Error("peer ID 1 should not be a bot")
	}
	if entity.IsBotID(100) {
		t.Error("peer ID 100 should not be a bot")
	}
	if !entity.IsBotID(entity.BotIDBase) {
		t.Error("BotIDBase should be a bot")
	}
	if !entity.IsBotID(entity.BotIDBase + 1) {
		t.Error("BotIDBase+1 should be a bot")
	}
}

func TestTickFollow(t *testing.T) {
	m := NewManager("")
	w := testWorld()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	owner.Position = entity.Vec3{X: 20, Y: 0, Z: 20}
	w.Players[1] = owner

	botID, _ := m.SpawnBot(1, entity.ClassVanguard, "blade", w)
	bot := w.Players[botID]
	bot.Position = entity.Vec3{X: 0, Y: 0, Z: 0}

	// Tick several times — bot should move toward owner
	for range 20 {
		w.InputQueue = w.InputQueue[:0]
		m.TickAll(w, 0.05)
	}

	dx := bot.Position.X - 0.0
	dz := bot.Position.Z - 0.0
	if dx*dx+dz*dz < 1.0 {
		t.Error("bot should have moved from origin toward owner")
	}
}

func TestUnknownClassReturnsError(t *testing.T) {
	m := NewManager("")
	w := testWorld()

	owner := entity.NewPlayer(1, entity.ClassGunner)
	w.Players[1] = owner

	_, err := m.SpawnBot(1, "nonexistent_class", "", w)
	if err == nil {
		t.Error("expected error for unknown class")
	}
}
