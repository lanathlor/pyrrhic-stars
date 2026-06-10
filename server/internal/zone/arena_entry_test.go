package zone

import (
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

// buildPlayerInputPayload creates an OpPlayerInput payload.
func buildPlayerInputPayload(x, y, z, rotY float32, tick uint32, visualState uint8, aimPitch float32) []byte {
	buf := make([]byte, 0, 25)
	b4 := make([]byte, 4)

	putF32 := func(v float32) {
		binary.LittleEndian.PutUint32(b4, math.Float32bits(v))
		buf = append(buf, b4...)
	}
	putU32 := func(v uint32) {
		binary.LittleEndian.PutUint32(b4, v)
		buf = append(buf, b4...)
	}

	putF32(x)
	putF32(y)
	putF32(z)
	putF32(rotY)
	putU32(tick)
	buf = append(buf, visualState)
	putF32(aimPitch)

	return buf
}

// TestArenaInstance_EnemiesAliveFromCreation verifies that enemies are
// alive and patrolling as soon as the arena zone is created.
func TestArenaInstance_EnemiesAliveFromCreation(t *testing.T) {
	z := New("test_arena", testArenaLevel(t), nil)

	aliveCount := 0
	patrolCount := 0
	for _, e := range z.world.Enemies {
		if e.Alive {
			aliveCount++
		}
		if e.State == entity.EnemyPatrol {
			patrolCount++
		}
	}

	if aliveCount == 0 {
		t.Error("no alive enemies after zone creation — InitInstance should activate them")
	}
	if patrolCount != aliveCount {
		t.Errorf("patrol=%d, alive=%d — all enemies should start in patrol state", patrolCount, aliveCount)
	}
	t.Logf("zone created with %d alive enemies, all patrolling", aliveCount)
}

// TestArenaInstance_TicksWithPlayer verifies that the zone ticks normally
// once a player joins and enemies are alive.
func TestArenaInstance_TicksWithPlayer(t *testing.T) {
	z := New("test_arena", testArenaLevel(t), nil)

	send, msgs := captureSend()
	c := &Client{PeerID: 1, Username: testPlayerName, Send: send, SendUDP: send, HasUDP: func() bool { return true }}
	z.AddClient(c)

	// Run a tick
	z.processTick()

	// Should broadcast world state
	if !findOpcode(*msgs, message.OpWorldState) {
		t.Error("client did not receive OpWorldState after tick")
	}
	// Enemies should still be alive
	for _, e := range z.world.Enemies {
		if !e.Alive {
			t.Errorf("Enemy %d should be alive after first tick", e.ID)
		}
	}
}

// TestArenaInstance_TeleportRejected verifies that a (0,0,0) position
// from client input is rejected as a teleport.
func TestArenaInstance_TeleportRejected(t *testing.T) {
	z := New("test_arena_zero", testArenaLevel(t), nil)

	send, _ := captureSend()
	c := &Client{PeerID: 1, Username: testPlayerName, Send: send, SendUDP: send, HasUDP: func() bool { return true }}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go z.Run(ctx)

	z.AddClient(c)

	// Simulate client sending (0, 0, 0) position — too far from spawn (Z≈48)
	zeroPayload := buildPlayerInputPayload(0, 0, 0, 0, 1, 0, 0)
	z.QueueInput(1, 0x0030, zeroPayload)

	time.Sleep(100 * time.Millisecond)

	z.mu.Lock()
	pos := z.world.Players[1].Position
	z.mu.Unlock()

	if pos.Z < 40.0 {
		t.Errorf("Player Z=%f after (0,0,0) input — server should have rejected the teleport", pos.Z)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

// TestArenaInstance_ConcurrentTickSafe verifies no race between Run and AddClient.
func TestArenaInstance_ConcurrentTickSafe(t *testing.T) {
	z := New("test_arena_concurrent", testArenaLevel(t), nil)

	send, _ := captureSend()
	c := &Client{PeerID: 1, Username: testPlayerName, Send: send, SendUDP: send, HasUDP: func() bool { return true }}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go z.Run(ctx)

	z.AddClient(c)
	time.Sleep(200 * time.Millisecond)

	cancel()
	time.Sleep(50 * time.Millisecond)

	// Should have ticked at least once with a player. Read under lock to
	// avoid racing with the final tick.
	z.mu.Lock()
	pos := z.world.Players[1].Position
	z.mu.Unlock()
	t.Logf("Player pos: %+v", pos)
}
