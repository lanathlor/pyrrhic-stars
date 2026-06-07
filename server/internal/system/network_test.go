package system

import (
	"encoding/binary"
	"sync"
	"testing"

	"codex-online/server/internal/combat"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

// mockClient creates a Client with a Send func that captures sent messages.
type sentMessages struct {
	mu   sync.Mutex
	msgs [][]byte
}

func (s *sentMessages) add(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	s.msgs = append(s.msgs, cp)
}

func (s *sentMessages) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

func (s *sentMessages) opcode(idx int) uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx >= len(s.msgs) || len(s.msgs[idx]) < 2 {
		return 0
	}
	return binary.BigEndian.Uint16(s.msgs[idx][0:2])
}

func newMockClient(peerID uint16) (*Client, *sentMessages) {
	sm := &sentMessages{}
	c := &Client{
		PeerID:   peerID,
		Username: "test",
		Send:     sm.add,
		SendUDP:  sm.add,
	}
	return c, sm
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — Hub
// ---------------------------------------------------------------------------

func TestNetworkSystem_Hub_BroadcastsWorldState(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Position = entity.Vec3{X: 5, Y: 0.1, Z: 10}

	c1, sent1 := newMockClient(1)

	w := &World{
		ZoneType: 0, // Hub
		TickNum:  50,
		State:    StateLobby,
		Players:  map[uint16]*entity.Player{1: p},
		Enemies:  []*entity.Enemy{},
		Level:    testHubLevel(t),
		Clients:  map[uint16]*Client{1: c1},
		TestMode: true,
	}

	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	if sent1.count() == 0 {
		t.Fatal("expected at least one message sent to client in hub")
	}

	// Hub should send WorldState
	foundWorldState := false
	for i := 0; i < sent1.count(); i++ {
		if sent1.opcode(i) == message.OpWorldState {
			foundWorldState = true
		}
	}
	if !foundWorldState {
		t.Error("hub should broadcast OpWorldState")
	}
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — Arena Lobby
// ---------------------------------------------------------------------------

func TestNetworkSystem_ArenaLobby_BroadcastsLobbyState(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	p.Ready = true

	c1, sent1 := newMockClient(1)

	w := &World{
		ZoneType: 1, // Arena
		TickNum:  50,
		State:    StateLobby,
		Players:  map[uint16]*entity.Player{1: p},
		Enemies:  []*entity.Enemy{},
		Level:    testArenaLevel(t),
		Clients:  map[uint16]*Client{1: c1},
		TestMode: true,
	}

	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	if sent1.count() == 0 {
		t.Fatal("expected at least one message")
	}

	foundLobby := false
	for i := 0; i < sent1.count(); i++ {
		if sent1.opcode(i) == message.OpLobbyState {
			foundLobby = true
		}
	}
	if !foundLobby {
		t.Error("arena lobby should broadcast OpLobbyState")
	}
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — Arena Fight
// ---------------------------------------------------------------------------

func TestNetworkSystem_ArenaFight_BroadcastsWorldStateAndDamage(t *testing.T) {
	p := entity.NewPlayer(1, entity.ClassGunner)
	e := entity.NewEnemy(0, 2000, "guard_captain")

	c1, sent1 := newMockClient(1)

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: p},
		Enemies:  []*entity.Enemy{e},
		Level:    testArenaLevel(t),
		Clients:  map[uint16]*Client{1: c1},
		DamageEvents: []combat.DamageEvent{
			{TargetPeerID: 0, SourcePeerID: 1, Amount: 25.0, HitPos: entity.Vec3{X: 1}, SourceType: combat.SourcePlayerAttack},
		},
		TestMode: true,
	}

	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	foundWorld := false
	foundDamage := false
	for i := 0; i < sent1.count(); i++ {
		op := sent1.opcode(i)
		if op == message.OpWorldState {
			foundWorld = true
		}
		if op == message.OpDamageEvent {
			foundDamage = true
		}
	}
	if !foundWorld {
		t.Error("fight should broadcast OpWorldState")
	}
	if !foundDamage {
		t.Error("fight should broadcast OpDamageEvent")
	}

	// Damage events should be cleared after broadcast
	if len(w.DamageEvents) != 0 {
		t.Errorf("damage events = %d, want 0 (should be cleared)", len(w.DamageEvents))
	}
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — GameFlowEvents
// ---------------------------------------------------------------------------

func TestNetworkSystem_BroadcastsGameFlowEvents(t *testing.T) {
	c1, sent1 := newMockClient(1)
	c2, sent2 := newMockClient(2)

	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: entity.NewPlayer(1, entity.ClassGunner), 2: entity.NewPlayer(2, entity.ClassVanguard)},
		Enemies:  []*entity.Enemy{},
		Level:    testArenaLevel(t),
		Clients:  map[uint16]*Client{1: c1, 2: c2},
		GameFlowEvents: []GameFlowEvent{
			{FlowType: message.FlowFightStart},
		},
		TestMode: true,
	}

	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	// Both clients should receive the game flow event
	for _, pair := range []struct {
		name string
		sent *sentMessages
	}{
		{"client1", sent1},
		{"client2", sent2},
	} {
		foundGameFlow := false
		for i := 0; i < pair.sent.count(); i++ {
			if pair.sent.opcode(i) == message.OpGameFlowEvent {
				foundGameFlow = true
			}
		}
		if !foundGameFlow {
			t.Errorf("%s should receive OpGameFlowEvent", pair.name)
		}
	}

	// Game flow events should be cleared after broadcast
	if len(w.GameFlowEvents) != 0 {
		t.Errorf("game flow events = %d, want 0 (should be cleared)", len(w.GameFlowEvents))
	}
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — Multiple clients all receive messages
// ---------------------------------------------------------------------------

func TestNetworkSystem_MultipleClients(t *testing.T) {
	c1, sent1 := newMockClient(1)
	c2, sent2 := newMockClient(2)
	c3, sent3 := newMockClient(3)

	w := &World{
		ZoneType: 0, // Hub
		TickNum:  10,
		State:    StateLobby,
		Players: map[uint16]*entity.Player{
			1: entity.NewPlayer(1, entity.ClassGunner),
			2: entity.NewPlayer(2, entity.ClassVanguard),
			3: entity.NewPlayer(3, entity.ClassBladeDancer),
		},
		Enemies: []*entity.Enemy{},
		Level:   testHubLevel(t),
		Clients: map[uint16]*Client{1: c1, 2: c2, 3: c3},
	}

	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	for _, pair := range []struct {
		name string
		sent *sentMessages
	}{
		{"client1", sent1},
		{"client2", sent2},
		{"client3", sent3},
	} {
		if pair.sent.count() == 0 {
			t.Errorf("%s received no messages", pair.name)
		}
	}
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — StateFightOver broadcasts world state
// ---------------------------------------------------------------------------

func TestNetworkSystem_FightOverBroadcastsWorldState(t *testing.T) {
	c1, sent1 := newMockClient(1)

	w := &World{
		ZoneType: 1,
		TickNum:  200,
		State:    StateFightOver,
		Players:  map[uint16]*entity.Player{1: entity.NewPlayer(1, entity.ClassGunner)},
		Enemies:  []*entity.Enemy{},
		Level:    testArenaLevel(t),
		Clients:  map[uint16]*Client{1: c1},
		TestMode: true,
	}

	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	foundWorld := false
	for i := 0; i < sent1.count(); i++ {
		if sent1.opcode(i) == message.OpWorldState {
			foundWorld = true
		}
	}
	if !foundWorld {
		t.Error("StateFightOver should broadcast OpWorldState")
	}
}

// ---------------------------------------------------------------------------
// NetworkSystem.Tick — No clients, no panic
// ---------------------------------------------------------------------------

func TestNetworkSystem_NoClients(t *testing.T) {
	w := &World{
		ZoneType: 1,
		TickNum:  100,
		State:    StateFight,
		Players:  map[uint16]*entity.Player{1: entity.NewPlayer(1, entity.ClassGunner)},
		Enemies:  []*entity.Enemy{},
		Level:    testArenaLevel(t),
		Clients:  map[uint16]*Client{},
		DamageEvents: []combat.DamageEvent{
			{TargetPeerID: 0, SourcePeerID: 1, Amount: 10},
		},
		GameFlowEvents: []GameFlowEvent{
			{FlowType: message.FlowFightStart},
		},
	}

	// Should not panic
	sys := &NetworkSystem{}
	sys.Tick(w, 0.05)

	if len(w.DamageEvents) != 0 {
		t.Error("damage events should still be cleared")
	}
	if len(w.GameFlowEvents) != 0 {
		t.Error("game flow events should still be cleared")
	}
}
