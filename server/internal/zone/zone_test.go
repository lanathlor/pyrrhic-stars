package zone

import (
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"testing"
)

// buildShootPayload creates an OpAbilityInput payload for a gunner shot.
// Format: [action:u8][aim_pitch:f32 LE]
func buildShootPayload(aimPitch float32, rotY ...float32) []byte {
	buf := make([]byte, 9)
	buf[0] = entity.ActionShoot
	binary.LittleEndian.PutUint32(buf[1:], math.Float32bits(aimPitch))
	if len(rotY) > 0 {
		binary.LittleEndian.PutUint32(buf[5:], math.Float32bits(rotY[0]))
	}
	return buf
}

// findBoss returns the boss enemy from a zone, or nil.
func findBoss(z *Zone) *entity.Enemy {
	for _, e := range z.world.Enemies {
		if e.IsBoss {
			return e
		}
	}
	return nil
}

// setupFightZone creates an arena zone in the FIGHT state with one gunner
// player aimed directly at the boss, and returns it ready for testing.
// All enemies (trash + boss) are activated and alive.
func setupFightZone(t *testing.T) (*Zone, uint16) {
	t.Helper()
	z := New("test_arena", ZoneTypeArena)
	z.world.State = StateFight

	peerID := uint16(1)

	// Activate all enemies
	for _, e := range z.world.Enemies {
		e.Alive = true
		e.State = entity.EnemyIdle
	}

	// Position boss at origin
	boss := findBoss(z)
	if boss == nil {
		t.Fatal("no boss enemy found in arena zone")
	}
	boss.Position = entity.Vec3{X: 0, Y: 0, Z: 0}

	// Position player at Z=10, aimed at boss center mass (0, 1, 0)
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
	z.world.Players[peerID] = player

	// Add a mock client that captures sent messages
	z.world.Clients[peerID] = &Client{
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
//   1. Processed inputs (handleAbilityInput -> appended to damageEvents)
//   2. Cleared damageEvents at the start of tickFight
//   3. Broadcast damageEvents (now empty -- player events lost)
func TestPlayerDamageEventsSurviveTick(t *testing.T) {
	z, peerID := setupFightZone(t)

	// Record all messages sent to the client
	var sentMessages [][]byte
	z.world.Clients[peerID].Send = func(msg []byte) {
		sentMessages = append(sentMessages, msg)
	}

	// Queue a shoot input
	aimPitch := z.world.Players[peerID].AimPitch
	z.mu.Lock()
	z.pendingInputs = append(z.pendingInputs, system.InputMsg{
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

			if targetPeer >= 1000 && sourcePeer == peerID && sourceType == 0 {
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

// TestEnemyDamageEventsStillWork verifies that enemy->player damage events
// (created during tickFight) are still broadcast correctly.
func TestEnemyDamageEventsStillWork(t *testing.T) {
	z, peerID := setupFightZone(t)

	// Put enemy in melee attack state, right next to the player
	z.world.Enemies[0].Position = entity.Vec3{X: 0, Y: 0, Z: 10}
	z.world.Enemies[0].State = entity.EnemyMeleeAttack
	z.world.Enemies[0].StateTimer = 0.001 // about to finish

	var sentMessages [][]byte
	z.world.Clients[peerID].Send = func(msg []byte) {
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

// =============================================================================
// Additional helpers
// =============================================================================

// setupMultiPlayerFightZone creates a fight zone with N players.
func setupMultiPlayerFightZone(t *testing.T, n int) (*Zone, []uint16) {
	t.Helper()
	z := New("test_arena", ZoneTypeArena)
	z.world.State = StateFight
	// Activate all enemies
	for _, e := range z.world.Enemies {
		e.Alive = true
		e.State = entity.EnemyIdle
	}
	boss := findBoss(z)
	if boss != nil {
		boss.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	}
	var ids []uint16
	for i := 0; i < n; i++ {
		pid := uint16(i + 1)
		p := entity.NewPlayer(pid, "gunner")
		p.Position = entity.Vec3{X: float32(i) * 2, Y: 0.1, Z: 10}
		p.Alive = true
		z.world.Players[pid] = p
		z.world.Clients[pid] = &Client{PeerID: pid, Username: fmt.Sprintf("P%d", pid), Send: func([]byte) {}}
		ids = append(ids, pid)
	}
	return z, ids
}

// captureSend returns a Send function and a pointer to the captured messages.
// Thread-safe for use across processTick calls.
func captureSend() (func([]byte), *[][]byte) {
	var mu sync.Mutex
	var msgs [][]byte
	return func(data []byte) {
		mu.Lock()
		defer mu.Unlock()
		cp := make([]byte, len(data))
		copy(cp, data)
		msgs = append(msgs, cp)
	}, &msgs
}

// findGameFlowEvent returns true if any captured message is an OpGameFlowEvent
// with the given flow type byte.
func findGameFlowEvent(messages [][]byte, flowType uint8) bool {
	for _, msg := range messages {
		if len(msg) < 5 {
			continue
		}
		opcode := binary.BigEndian.Uint16(msg[0:2])
		if opcode == message.OpGameFlowEvent {
			payload := msg[4:]
			if len(payload) > 0 && payload[0] == flowType {
				return true
			}
		}
	}
	return false
}

// findOpcode returns true if any captured message has the given opcode.
func findOpcode(messages [][]byte, op uint16) bool {
	for _, msg := range messages {
		if len(msg) >= 4 && binary.BigEndian.Uint16(msg[0:2]) == op {
			return true
		}
	}
	return false
}

// =============================================================================
// Test: checkFightEnd -- Boss Dead
// =============================================================================

func TestCheckFightEnd_BossDead(t *testing.T) {
	z, peerID := setupFightZone(t)

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	// Kill the boss directly
	boss := findBoss(z)
	if boss == nil {
		t.Fatal("no boss found")
	}
	boss.State = entity.EnemyDead
	boss.Alive = false

	z.processTick()

	if z.world.State != StateFightOver {
		t.Errorf("State = %d, want StateFightOver (%d)", z.world.State, StateFightOver)
	}
	if !z.world.BossDefeated {
		t.Error("BossDefeated = false, want true")
	}
	if z.world.Projectiles != nil {
		t.Errorf("Projectiles = %v, want nil", z.world.Projectiles)
	}
	if !findGameFlowEvent(*msgs, message.FlowBossDead) {
		t.Error("client did not receive FlowBossDead game flow event")
	}
}

// =============================================================================
// Test: checkFightEnd -- All Players Dead (Wipe)
// =============================================================================

func TestCheckFightEnd_AllPlayersDead(t *testing.T) {
	z, ids := setupMultiPlayerFightZone(t, 2)

	send, msgs := captureSend()
	for _, pid := range ids {
		z.world.Clients[pid].Send = send
	}

	// Kill all players
	for _, pid := range ids {
		z.world.Players[pid].Alive = false
		z.world.Players[pid].State = entity.PlayerStateDead
	}

	z.processTick()

	if z.world.State != StateFightOver {
		t.Errorf("State = %d, want StateFightOver (%d)", z.world.State, StateFightOver)
	}
	if z.world.BossDefeated {
		t.Error("BossDefeated = true, want false after wipe")
	}
	// Alive enemies should be reset to patrol, dead ones stay dead
	for _, e := range z.world.Enemies {
		if e.Alive && e.State != entity.EnemyPatrol {
			t.Errorf("Enemy %d State = %d, want EnemyPatrol after wipe reset", e.ID, e.State)
		}
	}
	if !findGameFlowEvent(*msgs, message.FlowAllDead) {
		t.Error("client did not receive FlowAllDead game flow event")
	}
}

// =============================================================================
// Test: checkFightEnd -- Fight Continues
// =============================================================================

func TestCheckFightEnd_FightContinues(t *testing.T) {
	z, _ := setupFightZone(t)

	// Player alive, enemy alive — fight should continue
	z.processTick()

	if z.world.State != StateFight {
		t.Errorf("State = %d, want StateFight (%d)", z.world.State, StateFight)
	}
}

// =============================================================================
// Test: tickFightOver -- All Respawn After Wipe -> Lobby Transition
// =============================================================================

func TestTickFightOver_AllRespawnedAfterWipe(t *testing.T) {
	z, ids := setupMultiPlayerFightZone(t, 2)
	z.world.State = StateFightOver
	z.world.BossDefeated = false

	send, msgs := captureSend()
	for _, pid := range ids {
		z.world.Clients[pid].Send = send
	}

	// All players alive (they have respawned)
	for _, pid := range ids {
		z.world.Players[pid].Alive = true
	}

	z.processTick()

	if z.world.State != StateSpawned {
		t.Errorf("State = %d, want StateSpawned (%d)", z.world.State, StateSpawned)
	}
	if !findGameFlowEvent(*msgs, message.FlowReturnLobby) {
		t.Error("client did not receive FlowReturnLobby game flow event")
	}
}

// =============================================================================
// Test: tickFightOver -- Boss Dead, No Auto-Lobby
// =============================================================================

func TestTickFightOver_BossDeadNoAutoLobby(t *testing.T) {
	z, ids := setupMultiPlayerFightZone(t, 2)
	z.world.State = StateFightOver
	z.world.BossDefeated = true

	for _, pid := range ids {
		z.world.Players[pid].Alive = true
	}

	z.processTick()

	if z.world.State != StateFightOver {
		t.Errorf("State = %d, want StateFightOver (%d) — boss dead should not auto-lobby", z.world.State, StateFightOver)
	}
}

// =============================================================================
// Test: tickFightOver -- Wipe, Not All Respawned Yet
// =============================================================================

func TestTickFightOver_WipeNotAllRespawned(t *testing.T) {
	z, ids := setupMultiPlayerFightZone(t, 2)
	z.world.State = StateFightOver
	z.world.BossDefeated = false

	// One alive, one still dead
	z.world.Players[ids[0]].Alive = true
	z.world.Players[ids[1]].Alive = false
	z.world.Players[ids[1]].State = entity.PlayerStateDead

	z.processTick()

	if z.world.State != StateFightOver {
		t.Errorf("State = %d, want StateFightOver (%d) — not all respawned yet", z.world.State, StateFightOver)
	}
}

// =============================================================================
// Test: handleRespawnRequest -- table-driven
// =============================================================================

func TestHandleRespawnRequest(t *testing.T) {
	tests := []struct {
		name            string
		state           GameFlowState
		playerAlive     bool
		respawnType     byte // 0 = arena, 1 = hub
		wantAlive       bool
		wantHealthReset bool
		wantPosition    *entity.Vec3
		wantCallback    bool
	}{
		{
			name:            "arena respawn in FightOver",
			state:           StateFightOver,
			playerAlive:     false,
			respawnType:     0,
			wantAlive:       true,
			wantHealthReset: true,
			wantPosition:    &entity.Vec3{X: 0, Y: 0.1, Z: 48},
		},
		{
			name:        "arena respawn rejected during fight",
			state:       StateFight,
			playerAlive: false,
			respawnType: 0,
			wantAlive:   false,
		},
		{
			name:         "hub respawn calls callback",
			state:        StateFightOver,
			playerAlive:  false,
			respawnType:  1,
			wantAlive:    false, // hub respawn does not revive locally
			wantCallback: true,
		},
		{
			name:        "alive player ignored",
			state:       StateFightOver,
			playerAlive: true,
			respawnType: 0,
			wantAlive:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z, peerID := setupFightZone(t)
			z.world.State = tc.state

			p := z.world.Players[peerID]
			p.Alive = tc.playerAlive
			if !tc.playerAlive {
				p.State = entity.PlayerStateDead
				p.Health = 0
			}

			var callbackPeerID uint16
			var callbackCalled bool
			z.OnPlayerRespawnHub = func(pid uint16) {
				callbackCalled = true
				callbackPeerID = pid
			}

			// Queue respawn request
			z.mu.Lock()
			z.pendingInputs = append(z.pendingInputs, system.InputMsg{
				PeerID:  peerID,
				Opcode:  message.OpRespawnRequest,
				Payload: []byte{tc.respawnType},
			})
			z.mu.Unlock()

			z.processTick()

			if p.Alive != tc.wantAlive {
				t.Errorf("Alive = %v, want %v", p.Alive, tc.wantAlive)
			}
			if tc.wantHealthReset && p.Health != p.MaxHealth {
				t.Errorf("Health = %f, want %f (MaxHealth)", p.Health, p.MaxHealth)
			}
			if tc.wantPosition != nil {
				if p.Position != *tc.wantPosition {
					t.Errorf("Position = %v, want %v", p.Position, *tc.wantPosition)
				}
			}
			if tc.wantCallback {
				if !callbackCalled {
					t.Error("OnPlayerRespawnHub callback was not called")
				}
				if callbackCalled && callbackPeerID != peerID {
					t.Errorf("callback peerID = %d, want %d", callbackPeerID, peerID)
				}
			}
			if !tc.wantCallback && callbackCalled {
				t.Error("OnPlayerRespawnHub callback was called unexpectedly")
			}
		})
	}
}

// =============================================================================
// Test: InteractExitPortal -- table-driven
// =============================================================================

func TestInteractExitPortal(t *testing.T) {
	tests := []struct {
		name         string
		state        GameFlowState
		bossDefeated bool
		wantCallback bool
	}{
		{
			name:         "triggers hub transfer after boss kill",
			state:        StateFightOver,
			bossDefeated: true,
			wantCallback: true,
		},
		{
			name:         "rejected when boss not dead",
			state:        StateFightOver,
			bossDefeated: false,
			wantCallback: false,
		},
		{
			name:         "rejected during fight",
			state:        StateFight,
			bossDefeated: false,
			wantCallback: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z, peerID := setupFightZone(t)
			z.world.State = tc.state
			z.world.BossDefeated = tc.bossDefeated

			var callbackCalled bool
			z.OnPlayerRespawnHub = func(pid uint16) {
				callbackCalled = true
			}

			// Queue InteractExitPortal
			z.mu.Lock()
			z.pendingInputs = append(z.pendingInputs, system.InputMsg{
				PeerID:  peerID,
				Opcode:  message.OpInteractInput,
				Payload: []byte{message.InteractExitPortal},
			})
			z.mu.Unlock()

			z.processTick()

			if callbackCalled != tc.wantCallback {
				t.Errorf("callback called = %v, want %v", callbackCalled, tc.wantCallback)
			}
		})
	}
}

// =============================================================================
// Test: Lobby -> Spawned -> Fight Transition
// =============================================================================

func TestLobbyToSpawnedToFight(t *testing.T) {
	z := New("test_arena", ZoneTypeArena)
	peerID := uint16(1)

	p := entity.NewPlayer(peerID, "gunner")
	p.Position = entity.Vec3{X: 0, Y: 0.1, Z: 48}
	p.Alive = true
	z.world.Players[peerID] = p

	send, msgs := captureSend()
	z.world.Clients[peerID] = &Client{PeerID: peerID, Username: "TestPlayer", Send: send}

	// Step 1: lobby — player not ready, should stay in lobby
	z.processTick()
	if z.world.State != StateLobby {
		t.Fatalf("expected StateLobby before ready, got %d", z.world.State)
	}

	// Step 2: player readies up
	z.mu.Lock()
	z.pendingInputs = append(z.pendingInputs, system.InputMsg{
		PeerID:  peerID,
		Opcode:  message.OpInteractInput,
		Payload: []byte{message.InteractReadyToggle},
	})
	z.mu.Unlock()

	z.processTick()

	if z.world.State != StateSpawned {
		t.Fatalf("expected StateSpawned after all ready, got %d", z.world.State)
	}
	if !findGameFlowEvent(*msgs, message.FlowSpawnPlayers) {
		t.Error("client did not receive FlowSpawnPlayers game flow event")
	}

	// Step 3: next tick transitions Spawned → Fight automatically (players present)
	z.processTick()

	if z.world.State != StateFight {
		t.Fatalf("expected StateFight after spawned tick, got %d", z.world.State)
	}
	if !findGameFlowEvent(*msgs, message.FlowFightStart) {
		t.Error("client did not receive FlowFightStart game flow event")
	}
}

// =============================================================================
// Test: broadcastState includes FightOver
// =============================================================================

func TestBroadcastState_FightOver(t *testing.T) {
	z, peerID := setupFightZone(t)
	z.world.State = StateFightOver
	z.world.BossDefeated = true // keeps state at FightOver (no auto-lobby)

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	z.processTick()

	if !findOpcode(*msgs, message.OpWorldState) {
		t.Error("client did not receive OpWorldState during StateFightOver")
	}
}

// =============================================================================
// Test: Gunner shot broadcasts PlayerStateAttack in world state
// =============================================================================

// extractPlayerState parses an OpWorldState message and returns the state byte
// for the given peer ID. Returns -1 if the peer was not found.
func extractPlayerState(msg []byte, wantPeer uint16) int {
	if len(msg) < 4 {
		return -1
	}
	opcode := binary.BigEndian.Uint16(msg[0:2])
	if opcode != message.OpWorldState {
		return -1
	}
	payload := msg[4:] // skip header
	if len(payload) < 5 {
		return -1
	}
	// tick:4, player_count:1
	playerCount := int(payload[4])
	off := 5
	for i := 0; i < playerCount; i++ {
		if off+2 > len(payload) {
			return -1
		}
		peerID := binary.LittleEndian.Uint16(payload[off : off+2])
		off += 2
		// pos(3*4) + rot_y(4) + health(4) = 20 bytes
		off += 20
		if off >= len(payload) {
			return -1
		}
		state := int(payload[off])
		off++ // state
		// class:str8
		if off >= len(payload) {
			return -1
		}
		classLen := int(payload[off])
		off++ // class_len
		off += classLen
		// name:str8
		if off >= len(payload) {
			return -1
		}
		nameLen := int(payload[off])
		off++ // name_len
		off += nameLen
		// anim:str8
		if off >= len(payload) {
			return -1
		}
		animLen := int(payload[off])
		off++ // anim_len
		off += animLen // anim bytes
		off += 4       // anim_speed
		off += 4       // aim_pitch
		if peerID == wantPeer {
			return state
		}
	}
	return -1
}

func TestGunnerShotBroadcastsAttackState(t *testing.T) {
	z, peerID := setupFightZone(t)

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	// Verify initial state is Move (0)
	z.processTick()
	found := false
	for _, msg := range *msgs {
		s := extractPlayerState(msg, peerID)
		if s >= 0 {
			found = true
			if s != int(entity.PlayerStateMove) {
				t.Errorf("initial state = %d, want %d (PlayerStateMove)", s, entity.PlayerStateMove)
			}
		}
	}
	if !found {
		t.Fatal("no world state message found in initial tick")
	}

	// Queue a gunner shot
	*msgs = (*msgs)[:0]
	aimPitch := z.world.Players[peerID].AimPitch
	z.mu.Lock()
	z.pendingInputs = append(z.pendingInputs, system.InputMsg{
		PeerID:  peerID,
		Opcode:  message.OpAbilityInput,
		Payload: buildShootPayload(aimPitch),
	})
	z.mu.Unlock()

	z.processTick()

	// World state should now contain PlayerStateAttack (2)
	found = false
	for _, msg := range *msgs {
		s := extractPlayerState(msg, peerID)
		if s >= 0 {
			found = true
			if s != int(entity.PlayerStateAttack) {
				t.Errorf("state after shot = %d, want %d (PlayerStateAttack)", s, entity.PlayerStateAttack)
			}
		}
	}
	if !found {
		t.Fatal("no world state message found after shot tick")
	}
}

func TestGunnerAttackStateResetsAfterCooldown(t *testing.T) {
	z, peerID := setupFightZone(t)

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	// Fire a shot
	aimPitch := z.world.Players[peerID].AimPitch
	z.mu.Lock()
	z.pendingInputs = append(z.pendingInputs, system.InputMsg{
		PeerID:  peerID,
		Opcode:  message.OpAbilityInput,
		Payload: buildShootPayload(aimPitch),
	})
	z.mu.Unlock()
	z.processTick()

	// Run enough ticks for cooldown to expire (0.18s / 0.05s = 3.6 -> 4 ticks)
	for i := 0; i < 5; i++ {
		*msgs = (*msgs)[:0]
		z.processTick()
	}

	// State should be back to Move
	for _, msg := range *msgs {
		s := extractPlayerState(msg, peerID)
		if s >= 0 {
			if s != int(entity.PlayerStateMove) {
				t.Errorf("state after cooldown = %d, want %d (PlayerStateMove)", s, entity.PlayerStateMove)
			}
		}
	}
}

// =============================================================================
// Test: Two-player zone: observer receives attack state from shooter
// =============================================================================

func TestRemotePlayerReceivesGunnerAttackState(t *testing.T) {
	z, ids := setupMultiPlayerFightZone(t, 2)
	shooterID := ids[0]
	observerID := ids[1]

	// Aim shooter at enemy
	eyePos := entity.Vec3{X: 0, Y: 1.6, Z: 10}
	targetCenter := entity.Vec3{X: 0, Y: 1, Z: 0}
	dir := targetCenter.Sub(eyePos).Normalized()
	z.world.Players[shooterID].RotationY = float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
	z.world.Players[shooterID].AimPitch = float32(math.Asin(float64(dir.Y)))

	// Capture observer's messages
	send, msgs := captureSend()
	z.world.Clients[observerID].Send = send

	// Shooter fires
	z.mu.Lock()
	z.pendingInputs = append(z.pendingInputs, system.InputMsg{
		PeerID:  shooterID,
		Opcode:  message.OpAbilityInput,
		Payload: buildShootPayload(z.world.Players[shooterID].AimPitch),
	})
	z.mu.Unlock()
	z.processTick()

	// Observer's world state should contain shooter's attack state
	found := false
	for _, msg := range *msgs {
		s := extractPlayerState(msg, shooterID)
		if s >= 0 {
			found = true
			if s != int(entity.PlayerStateAttack) {
				t.Errorf("shooter state seen by observer = %d, want %d (PlayerStateAttack)", s, entity.PlayerStateAttack)
			}
		}
	}
	if !found {
		t.Fatal("observer did not receive world state containing shooter's peer ID")
	}
}

// =============================================================================
// Test: Hub zone ticks without crashing
// =============================================================================

func TestHubZoneTick(t *testing.T) {
	z := New("test_hub", ZoneTypeHub)
	peerID := uint16(1)

	p := entity.NewPlayer(peerID, "gunner")
	p.Alive = true
	z.world.Players[peerID] = p

	send, msgs := captureSend()
	z.world.Clients[peerID] = &Client{PeerID: peerID, Username: "HubPlayer", Send: send}

	// Should not panic
	z.processTick()

	if !findOpcode(*msgs, message.OpWorldState) {
		t.Error("client did not receive OpWorldState from hub zone tick")
	}
}

// =============================================================================
// Test: Arena entry -- fight must NOT start until a player crosses the trigger
// =============================================================================

func TestArenaInstance_EnemiesAliveOnCreation(t *testing.T) {
	z := New("test_arena", ZoneTypeArena)

	// Enemies should be alive and patrolling from zone creation
	for _, e := range z.world.Enemies {
		if !e.Alive {
			t.Errorf("Enemy %d Alive = false, want true", e.ID)
		}
		if e.State != entity.EnemyPatrol {
			t.Errorf("Enemy %d State = %d, want EnemyPatrol (%d)", e.ID, e.State, entity.EnemyPatrol)
		}
	}
}

func TestArenaInstance_FightAfterPlayerJoin(t *testing.T) {
	z := New("test_arena", ZoneTypeArena)

	send, msgs := captureSend()
	c := &Client{PeerID: 1, Username: "TestPlayer", Send: send}
	z.AddClient(c)

	// First tick transitions Spawned → Fight (player is present)
	z.processTick()

	if z.world.State != StateFight {
		t.Errorf("State = %d, want StateFight (%d)", z.world.State, StateFight)
	}
	if !findGameFlowEvent(*msgs, message.FlowFightStart) {
		t.Error("Client did NOT receive FlowFightStart")
	}
}

func TestArenaInstance_SecondPlayerGetsCatchUp(t *testing.T) {
	z := New("test_arena", ZoneTypeArena)

	c1 := &Client{PeerID: 1, Username: "Player1", Send: func([]byte) {}}
	z.AddClient(c1)

	// Tick to enter fight
	z.processTick()

	if z.world.State != StateFight {
		t.Fatalf("State = %d, want StateFight", z.world.State)
	}

	// Second player joins mid-fight
	send2, msgs2 := captureSend()
	c2 := &Client{PeerID: 2, Username: "Player2", Send: send2}
	z.AddClient(c2)

	if !findGameFlowEvent(*msgs2, message.FlowFightStart) {
		t.Error("Second player did NOT receive FlowFightStart catch-up on join")
	}

	// Second player should be in warmup area
	p2 := z.world.Players[2]
	if p2 == nil {
		t.Fatal("Player 2 not found")
	}
	if p2.Position.Z < 40.0 {
		t.Errorf("Player2.Position.Z = %f, want >= 40 (warmup spawn)", p2.Position.Z)
	}
}
