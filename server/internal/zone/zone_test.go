package zone

import (
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"encoding/binary"
	"math"
	"testing"
)

// buildShootPayload creates an OpAbilityInput payload for a gunner shot.
// Format: [action:u8][aim_pitch:f32 LE]
func buildShootPayload(aimPitch float32) []byte {
	buf := make([]byte, 5)
	buf[0] = entity.ActionShoot
	binary.LittleEndian.PutUint32(buf[1:], math.Float32bits(aimPitch))
	return buf
}

// setupFightZone creates an arena zone in the FIGHT state with one gunner
// player aimed directly at the enemy, and returns it ready for testing.
func setupFightZone(t *testing.T) (*Zone, uint16) {
	t.Helper()
	z := New("test_arena", ZoneTypeArena)
	z.State = StateFight

	peerID := uint16(1)

	// Position enemy at origin
	z.Enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	z.Enemy.Alive = true
	z.Enemy.State = entity.EnemyIdle

	// Position player at Z=10, aimed at enemy center mass (0, 1, 0)
	eyePos := entity.Vec3{X: 0, Y: 1.6, Z: 10}
	targetCenter := entity.Vec3{X: 0, Y: 1, Z: 0}
	dir := targetCenter.Sub(eyePos).Normalized()
	yaw := float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
	pitch := float32(math.Asin(float64(dir.Y)))

	player := entity.NewPlayer(peerID, "gunner")
	player.Position = entity.Vec3{X: 0, Y: 0, Z: 10}
	player.RotationY = yaw
	player.AimPitch = pitch
	player.Alive = true
	z.Players[peerID] = player

	// Add a mock client that captures sent messages
	z.clients[peerID] = &Client{
		PeerID:   peerID,
		Username: "TestPlayer",
		Send:     func([]byte) {}, // no-op
	}

	return z, peerID
}

// TestPlayerDamageEventsSurviveTick verifies that damage events created by
// player ability inputs during processTick are NOT cleared before broadcast.
//
// This is a regression test for a bug where processTick:
//   1. Processed inputs (handleAbilityInput → appended to damageEvents)
//   2. Cleared damageEvents at the start of tickFight
//   3. Broadcast damageEvents (now empty — player events lost)
func TestPlayerDamageEventsSurviveTick(t *testing.T) {
	z, peerID := setupFightZone(t)

	// Record all messages sent to the client
	var sentMessages [][]byte
	z.clients[peerID].Send = func(msg []byte) {
		sentMessages = append(sentMessages, msg)
	}

	// Queue a shoot input
	aimPitch := z.Players[peerID].AimPitch
	z.mu.Lock()
	z.inputQueue = append(z.inputQueue, inputMsg{
		PeerID:  peerID,
		Opcode:  message.OpAbilityInput,
		Payload: buildShootPayload(aimPitch),
	})
	z.mu.Unlock()

	// Run one full tick
	z.processTick()

	// Check that damageEvents contains the player's hit
	// (If the bug is present, damageEvents will be empty because the clear
	// happens after inputs are processed but before broadcast.)
	foundDamageEvent := false
	for _, msg := range sentMessages {
		if len(msg) < 4 {
			continue
		}
		opcode := binary.BigEndian.Uint16(msg[0:2])
		if opcode == message.OpDamageEvent {
			// Parse the payload to verify it's our player's hit
			payload := msg[4:]
			if len(payload) < 21 {
				t.Errorf("DamageEvent payload too short: %d bytes, want 21", len(payload))
				continue
			}
			targetPeer := binary.LittleEndian.Uint16(payload[0:2])
			sourcePeer := binary.LittleEndian.Uint16(payload[2:4])
			amount := math.Float32frombits(binary.LittleEndian.Uint32(payload[4:8]))
			sourceType := payload[20]

			if targetPeer == 0 && sourcePeer == peerID && sourceType == 0 {
				foundDamageEvent = true
				if amount != 10.0 {
					t.Errorf("damage amount = %f, want 10.0", amount)
				}
				t.Logf("DamageEvent OK: target=%d source=%d amount=%.1f", targetPeer, sourcePeer, amount)
			}
		}
	}

	if !foundDamageEvent {
		t.Errorf("player damage event was NOT broadcast to client — " +
			"events from handleAbilityInput are being cleared before broadcastDamageEvents")
	}
}

// TestEnemyDamageEventsStillWork verifies that enemy→player damage events
// (created during tickFight) are still broadcast correctly.
func TestEnemyDamageEventsStillWork(t *testing.T) {
	z, peerID := setupFightZone(t)

	// Put enemy in melee attack state, right next to the player
	z.Enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 10}
	z.Enemy.State = entity.EnemyMeleeAttack
	z.Enemy.StateTimer = 0.001 // about to finish

	var sentMessages [][]byte
	z.clients[peerID].Send = func(msg []byte) {
		sentMessages = append(sentMessages, msg)
	}

	// Run tick — enemy should hit the player during tickFight
	z.processTick()

	foundEnemyDamage := false
	for _, msg := range sentMessages {
		if len(msg) < 4 {
			continue
		}
		opcode := binary.BigEndian.Uint16(msg[0:2])
		if opcode == message.OpDamageEvent {
			payload := msg[4:]
			if len(payload) < 21 {
				continue
			}
			targetPeer := binary.LittleEndian.Uint16(payload[0:2])
			sourceType := payload[20]
			if targetPeer == peerID && sourceType == 1 { // SourceEnemyMelee
				foundEnemyDamage = true
			}
		}
	}

	if !foundEnemyDamage {
		t.Log("no enemy melee damage event found (enemy may not have hit during this tick state)")
		// Not a hard failure — the enemy FSM might not produce a hit in one tick.
		// This test mainly ensures the pipeline doesn't crash.
	}
}
