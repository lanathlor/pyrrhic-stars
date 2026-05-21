package bot

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"

	"codex-online/server/internal/bosstest"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/system"
)

// Manager tracks all active bots in a zone.
type Manager struct {
	bots      map[uint16]*LiveBot
	byOwner   map[uint16][]uint16
	nextID    uint16
	treeReg   *bosstest.PuppetTreeRegistry
	rng       *rand.Rand
	OnRescale RescaleFunc
}

// NewManager creates a bot manager. If treeDir is non-empty, puppet behavior
// trees are loaded from that directory. Otherwise only hardcoded Go trees are
// used (which is fine for Phase 0).
func NewManager(treeDir string) *Manager {
	m := &Manager{
		bots:    make(map[uint16]*LiveBot),
		byOwner: make(map[uint16][]uint16),
		nextID:  entity.BotIDBase,
		rng:     rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
	if treeDir != "" {
		reg, err := bosstest.LoadPuppetTrees(treeDir)
		if err != nil {
			slog.Warn("bot: failed to load puppet trees, using hardcoded fallback", "error", err)
		} else {
			m.treeReg = reg
		}
	}
	return m
}

// SpawnBot creates a bot player with the given class/spec, positions it near the
// owner, and inserts it into the world. Returns the bot's peer ID.
func (m *Manager) SpawnBot(ownerID uint16, className, specID string, w *system.World) (uint16, error) {
	if len(m.byOwner[ownerID]) >= MaxBotsPerPlayer {
		return 0, fmt.Errorf("bot: max %d bots per player", MaxBotsPerPlayer)
	}
	if _, ok := entity.Classes[className]; !ok {
		return 0, fmt.Errorf("bot: unknown class %q", className)
	}

	botID := m.nextID
	m.nextID++

	seed := m.rng.Uint64()
	pp := bosstest.NewPuppet(botID, className, specID, bosstest.ProfileAverage, seed, "", m.treeReg)
	pp.NoBoundsClamp = true // live bots use level geometry, not hardcoded boss room bounds

	// Position near owner
	owner := w.Players[ownerID]
	if owner != nil {
		pp.Player.Position = owner.Position
	}
	pp.Player.SpawnTick = w.TickNum
	pp.Player.Username = fmt.Sprintf("[BOT] %s", className)
	pp.Player.Ready = true

	// Randomize follow offset so multiple bots don't stack.
	angle := m.rng.Float32() * 2 * math.Pi
	dist := 2.0 + m.rng.Float32()*2.0
	offset := entity.Vec3{
		X: float32(math.Cos(float64(angle))) * dist,
		Z: float32(math.Sin(float64(angle))) * dist,
	}

	lb := &LiveBot{
		Puppet:       pp,
		OwnerID:      ownerID,
		BotID:        botID,
		followOffset: offset,
	}

	m.bots[botID] = lb
	m.byOwner[ownerID] = append(m.byOwner[ownerID], botID)
	w.Players[botID] = pp.Player

	slog.Info("bot: spawned", "bot_id", botID, "owner", ownerID, "class", className, "spec", specID)
	m.triggerRescale(w)
	return botID, nil
}

// DismissBot removes a single bot from the world.
func (m *Manager) DismissBot(botID uint16, w *system.World) {
	lb, ok := m.bots[botID]
	if !ok {
		return
	}
	delete(w.Players, botID)
	delete(m.bots, botID)
	m.removeFromOwnerList(lb.OwnerID, botID)
	slog.Info("bot: dismissed", "bot_id", botID, "owner", lb.OwnerID)
	m.triggerRescale(w)
}

// DismissAllForOwner removes all bots owned by a player.
func (m *Manager) DismissAllForOwner(ownerID uint16, w *system.World) {
	ids := m.byOwner[ownerID]
	for _, id := range ids {
		delete(w.Players, id)
		delete(m.bots, id)
	}
	delete(m.byOwner, ownerID)
	m.triggerRescale(w)
}

// BotCount returns the number of bots owned by a player.
func (m *Manager) BotCount(ownerID uint16) int {
	return len(m.byOwner[ownerID])
}

// AllBots returns all active bots. Used for building AllPuppets slice.
func (m *Manager) AllBots() []*LiveBot {
	out := make([]*LiveBot, 0, len(m.bots))
	for _, lb := range m.bots {
		out = append(out, lb)
	}
	return out
}

func (m *Manager) triggerRescale(w *system.World) {
	if m.OnRescale != nil {
		m.OnRescale(len(w.Players))
	}
}

func (m *Manager) removeFromOwnerList(ownerID, botID uint16) {
	ids := m.byOwner[ownerID]
	for i, id := range ids {
		if id == botID {
			m.byOwner[ownerID] = append(ids[:i], ids[i+1:]...)
			return
		}
	}
}
