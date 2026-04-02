package zone

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

// buildPlayerInputPayload creates an OpPlayerInput payload.
func buildPlayerInputPayload(x, y, z, rotY float32, tick uint32, animName string, animSpeed, aimPitch float32) []byte {
	buf := make([]byte, 0, 64)
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
	nameBytes := []byte(animName)
	buf = append(buf, byte(len(nameBytes)))
	buf = append(buf, nameBytes...)
	putF32(animSpeed)
	putF32(aimPitch)

	return buf
}

// TestArenaEntry_ZeroPositionTriggersFight tests if a (0,0,0) position
// from client input triggers the fight prematurely.
func TestArenaEntry_ZeroPositionTriggersFight(t *testing.T) {
	z := New("test_arena_zero", ZoneTypeArena)

	send, _ := captureSend()
	c := &Client{PeerID: 1, Username: "TestPlayer", Send: send}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go z.Run(ctx)

	z.AddClient(c)

	// Simulate client sending (0, 0, 0) position
	zeroPayload := buildPlayerInputPayload(0, 0, 0, 0, 1, "idle", 1.0, 0)
	z.QueueInput(1, 0x0030, zeroPayload)

	time.Sleep(100 * time.Millisecond)

	z.mu.Lock()
	state := z.world.State
	pos := z.world.Players[1].Position
	z.mu.Unlock()

	t.Logf("After (0,0,0) input: state=%d, pos=%+v", state, pos)

	if state == StateFight {
		t.Error("Fight started because client sent (0,0,0) position — server should clamp Z >= 12 during warmup")
	}
	if pos.Z < 12.0 {
		t.Errorf("Player Z=%f after (0,0,0) input — server should have clamped to Z >= 12", pos.Z)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}

// TestArenaEntry_HubPositionDoesNotTriggerFight tests that a position
// right at the portal (Z=12) doesn't trigger the fight.
func TestArenaEntry_HubPositionDoesNotTriggerFight(t *testing.T) {
	z := New("test_arena_hub", ZoneTypeArena)

	send, _ := captureSend()
	c := &Client{PeerID: 1, Username: "TestPlayer", Send: send}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go z.Run(ctx)

	z.AddClient(c)

	// Z=12 is the portal position — should NOT trigger (condition is Z < 12, not Z <= 12)
	portalPayload := buildPlayerInputPayload(0, 0.1, 12.0, 0, 1, "idle", 1.0, 0)
	z.QueueInput(1, 0x0030, portalPayload)

	time.Sleep(100 * time.Millisecond)

	z.mu.Lock()
	state := z.world.State
	z.mu.Unlock()

	if state == StateFight {
		t.Error("Fight started at Z=12.0 — portal position should not trigger fight")
	}

	// Z=11.9 should trigger
	enterPayload := buildPlayerInputPayload(0, 0.1, 11.9, 0, 2, "idle", 1.0, 0)
	z.QueueInput(1, 0x0030, enterPayload)

	time.Sleep(100 * time.Millisecond)

	z.mu.Lock()
	state = z.world.State
	z.mu.Unlock()

	if state != StateFight {
		t.Errorf("State = %d after Z=11.9, want StateFight", state)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}
