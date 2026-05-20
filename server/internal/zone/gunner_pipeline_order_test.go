package zone

import (
	"encoding/binary"
	"testing"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
	"codex-online/server/internal/system"
)

// =============================================================================
// These tests verify that the CombatSystem→InputSystem pipeline ordering
// allows sustained gunner fire without rejected shots. The root cause was
// InputSystem checking cooldowns *before* CombatSystem drained them, so
// a shot arriving on the tick where its cooldown would expire got rejected.
//
// fire_shot cooldown = 0.18s, dt = 0.05s → 4 ticks to drain.
// With correct ordering, CombatSystem drains the remaining 0.03 first,
// then InputSystem sees cooldown=0 and accepts the shot.
// =============================================================================

// getAssaultState retrieves the GunnerAssaultState from a player.
func getAssaultState(t *testing.T, p *entity.Player) *ability.GunnerAssaultState {
	t.Helper()
	s, ok := p.AbilityState["gunner_assault"].(*ability.GunnerAssaultState)
	if !ok {
		t.Fatal("GunnerAssaultState not found on player")
	}
	return s
}

// TestFireShot_SustainedFire_NoRejections fires 10 shots at the exact cadence
// the cooldown allows (every 4 ticks) and verifies all 10 are accepted.
// Acceptance is tracked via magazine consumption (not damage events) because
// stability bloom may cause hitscan misses at range.
//
// Before the pipeline reorder, every 3rd shot was silently rejected because
// CombatSystem hadn't drained the remaining 0.03s cooldown yet.
func TestFireShot_SustainedFire_NoRejections(t *testing.T) {
	z, peerID := setupFightZone(t)
	boss := findBoss(z)
	boss.Health = 1e6

	p := z.world.Players[peerID]
	aimPitch := p.AimPitch

	// Run 1 idle tick so the assault state is initialized by the tick handler
	z.processTick()
	state := getAssaultState(t, p)

	const shotsToFire = 10
	const ticksPerShot = 4 // ceil(0.18 / 0.05) = 4 ticks

	accepted := 0
	for range shotsToFire {
		magBefore := state.MagCurrent

		z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
		z.processTick()

		if state.MagCurrent < magBefore {
			accepted++
		}

		// Wait the remaining ticks for cooldown to drain
		for i := 1; i < ticksPerShot; i++ {
			z.processTick()
		}
	}

	if accepted != shotsToFire {
		t.Errorf("accepted %d/%d shots — pipeline ordering may be wrong (CombatSystem must run before InputSystem)",
			accepted, shotsToFire)
	} else {
		t.Logf("OK: all %d shots accepted at %d-tick cadence", shotsToFire, ticksPerShot)
	}
}

// TestFireShot_MagazineDepletion_ExactCount fires exactly 30 rounds (full magazine)
// and verifies all 30 are accepted, magazine reaches 0, and auto-reload starts.
func TestFireShot_MagazineDepletion_ExactCount(t *testing.T) {
	z, peerID := setupFightZone(t)
	boss := findBoss(z)
	boss.Health = 1e6

	p := z.world.Players[peerID]
	aimPitch := p.AimPitch

	// Initialize assault state
	z.processTick()
	state := getAssaultState(t, p)

	const magSize = 30
	const ticksPerShot = 4

	accepted := 0
	for range magSize {
		magBefore := state.MagCurrent

		z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
		z.processTick()

		if state.MagCurrent < magBefore {
			accepted++
		}

		for i := 1; i < ticksPerShot; i++ {
			z.processTick()
		}
	}

	if accepted != magSize {
		t.Errorf("accepted %d/%d shots, want %d", accepted, magSize, magSize)
	}
	if state.MagCurrent != 0 {
		t.Errorf("MagCurrent = %d, want 0 (empty)", state.MagCurrent)
	}
	if !state.Reloading {
		t.Error("expected auto-reload to start when magazine hits 0")
	}

	// 31st shot should be rejected (empty + reloading)
	magBefore := state.MagCurrent
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
	z.processTick()
	if state.MagCurrent < magBefore {
		t.Error("shot accepted on empty magazine during reload — should be rejected")
	}
}

// TestFireShot_BoundaryTick_CooldownExpiry is a table-driven test that verifies
// the exact boundary behavior: a shot arriving on the tick where the cooldown
// *would* expire (with correct pipeline order) should be accepted.
func TestFireShot_BoundaryTick_CooldownExpiry(t *testing.T) {
	tests := []struct {
		name           string
		preloadCD      float32
		expectAccepted bool
	}{
		{
			name:           "CD=0.03 (expires this tick, dt=0.05)",
			preloadCD:      0.03,
			expectAccepted: true,
		},
		{
			name:           "CD=0.05 (exactly dt, drains to 0)",
			preloadCD:      0.05,
			expectAccepted: true,
		},
		{
			name:           "CD=0.08 (still 0.03 left after drain)",
			preloadCD:      0.08,
			expectAccepted: false,
		},
		{
			name:           "CD=0.00 (already expired)",
			preloadCD:      0.00,
			expectAccepted: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z, peerID := setupFightZone(t)
			boss := findBoss(z)
			boss.Health = 1e6

			p := z.world.Players[peerID]

			// Initialize assault state
			z.processTick()
			state := getAssaultState(t, p)

			// Pre-load the cooldown
			p.Cooldowns["fire_shot"] = tc.preloadCD

			magBefore := state.MagCurrent
			z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(p.AimPitch))
			z.processTick()

			got := state.MagCurrent < magBefore
			if got != tc.expectAccepted {
				t.Errorf("accepted=%v, want %v (preload CD=%.3f, dt=0.05)",
					got, tc.expectAccepted, tc.preloadCD)
			}
		})
	}
}

// TestFireShot_AutoReload_BlocksFiring verifies that after the magazine empties,
// auto-reload blocks all shots for its full duration, then firing resumes.
func TestFireShot_AutoReload_BlocksFiring(t *testing.T) {
	z, peerID := setupFightZone(t)
	boss := findBoss(z)
	boss.Health = 1e6

	p := z.world.Players[peerID]
	aimPitch := p.AimPitch

	// Initialize assault state and set magazine to 1
	z.processTick()
	state := getAssaultState(t, p)
	state.MagCurrent = 1

	// Fire the last round — should succeed and trigger auto-reload
	magBefore := state.MagCurrent
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
	z.processTick()
	if state.MagCurrent >= magBefore {
		t.Fatal("last round was not accepted")
	}
	if !state.Reloading {
		t.Fatal("expected auto-reload to start")
	}

	// Verify mid-reload rejection: try to fire while reload is in progress.
	// Tick a few times first to advance reload partway.
	for range 10 {
		z.processTick()
	}
	if !state.Reloading {
		t.Fatal("reload should still be in progress after 10 ticks (0.5s of 2.2s)")
	}
	magBefore = state.MagCurrent
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
	z.processTick()
	if state.MagCurrent != magBefore {
		t.Error("shot accepted during reload — should be rejected")
	}

	// Tick until reload completes (no shooting — just let it finish)
	const maxReloadTicks = 100
	reloadTicks := 11 // already ran 10 + 1 with shot
	for range maxReloadTicks {
		if !state.Reloading {
			break
		}
		z.processTick()
		reloadTicks++
	}
	if state.Reloading {
		t.Fatal("reload did not complete within safety bound")
	}
	if state.MagCurrent != state.MagMax {
		t.Errorf("MagCurrent = %d after reload, want %d", state.MagCurrent, state.MagMax)
	}
	// Expect ~44 ticks total for empty reload (2.2s / 0.05s)
	if reloadTicks < 40 || reloadTicks > 50 {
		t.Errorf("reload took %d ticks — expected ~44", reloadTicks)
	}
	t.Logf("reload completed in %d ticks", reloadTicks)

	// CombatSystem needs one more tick to drain the fire_shot cooldown
	// that was set equal to reload duration (may have trailing float residue).
	z.processTick()

	// Now fire — should succeed
	magBefore = state.MagCurrent
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
	z.processTick()
	if state.MagCurrent >= magBefore {
		t.Error("first shot after reload was rejected — firing should resume after reload completes")
	}
}

// TestPipelineOrder_CombatSystemBeforeInput verifies the system pipeline
// ordering at the zone level — CombatSystem must come before InputSystem.
func TestPipelineOrder_CombatSystemBeforeInput(t *testing.T) {
	tests := []struct {
		name     string
		zoneType ZoneType
	}{
		{"open-world", ZoneTypeOpenWorld},
		{"instanced", ZoneTypeInstanced},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z := New("test", tc.zoneType)
			combatIdx := -1
			inputIdx := -1
			for i, sys := range z.systems {
				switch sys.(type) {
				case *system.CombatSystem:
					combatIdx = i
				case *system.InputSystem:
					inputIdx = i
				}
			}
			if combatIdx == -1 {
				t.Fatal("CombatSystem not found in pipeline")
			}
			if inputIdx == -1 {
				t.Fatal("InputSystem not found in pipeline")
			}
			if combatIdx >= inputIdx {
				t.Errorf("CombatSystem (index %d) must run before InputSystem (index %d)", combatIdx, inputIdx)
			}
		})
	}
}

// =============================================================================
// World state broadcast reflects shot on the same tick it fires.
//
// This proves the server is NOT behind — the issue is purely client-side.
// The client optimistically decrements magazine before the server has even
// received the packet, so the next world state (from a prior server tick)
// shows a higher magazine count than the client expects.
// =============================================================================

// extractMagazineFromWorldState finds a player's magazine count in captured messages.
// Returns (magazine, found).
func extractMagazineFromWorldState(msgs [][]byte, targetPeerID uint16) (uint8, bool) {
	for _, msg := range msgs {
		if len(msg) < 4 {
			continue
		}
		opcode := binary.BigEndian.Uint16(msg[0:2])
		if opcode != message.OpWorldState {
			continue
		}
		ws, ok := codec.DecodeWorldState(msg[4:])
		if !ok {
			continue
		}
		for _, p := range ws.Players {
			if p.PeerID == targetPeerID {
				return p.Magazine, true
			}
		}
	}
	return 0, false
}

// TestWorldState_MagazineReflectsShotOnSameTick proves that the world state
// broadcast on the tick a shot fires already contains the decremented magazine.
// This confirms the server is never "1 behind" — the lag is purely network
// latency between client send and server receive.
func TestWorldState_MagazineReflectsShotOnSameTick(t *testing.T) {
	z, peerID := setupFightZone(t)
	boss := findBoss(z)
	boss.Health = 1e6

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	// Initialize assault state
	z.processTick()
	*msgs = nil

	state := getAssaultState(t, z.world.Players[peerID])
	if state.MagCurrent != 30 {
		t.Fatalf("initial magazine = %d, want 30", state.MagCurrent)
	}

	// Fire and check world state from the same tick
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(z.world.Players[peerID].AimPitch))
	z.processTick()

	mag, ok := extractMagazineFromWorldState(*msgs, peerID)
	if !ok {
		t.Fatal("no WorldState message found")
	}
	// The world state from THIS tick should show magazine=29 (shot consumed 1)
	if mag != 29 {
		t.Errorf("world state magazine = %d, want 29 (shot should be reflected on same tick)", mag)
	}
}

// TestWorldState_MagazineLagSimulation simulates the client-server timing that
// causes false "misfire" rollbacks. The sequence:
//  1. Tick N: no input queued. Server broadcasts magazine=30.
//  2. Client fires between tick N and tick N+1 (optimistic: _magazine=29).
//  3. Tick N+1: server receives shot, magazine=29, broadcasts magazine=29.
//
// A naive client that slams server_mag on every world state would see:
//   tick N broadcast arrives → server=30, client=29 → ROLLBACK (false misfire!)
//   tick N+1 broadcast arrives → server=29, client=29 → in sync
//
// The correct client behavior: ignore server_mag > _magazine (server hasn't
// processed our shot yet), only correct downward (server_mag < _magazine).
func TestWorldState_MagazineLagSimulation(t *testing.T) {
	z, peerID := setupFightZone(t)
	boss := findBoss(z)
	boss.Health = 1e6

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	// Initialize
	z.processTick()
	*msgs = nil

	aimPitch := z.world.Players[peerID].AimPitch

	// Tick N: no input. Server broadcasts magazine=30.
	z.processTick()
	magN, ok := extractMagazineFromWorldState(*msgs, peerID)
	if !ok {
		t.Fatal("no WorldState from tick N")
	}
	if magN != 30 {
		t.Fatalf("tick N magazine = %d, want 30", magN)
	}
	*msgs = nil

	// --- Between ticks: client fires optimistically (_magazine = 29) ---
	// We simulate this by noting that the client would decrement locally.
	clientMag := 29 // client's optimistic prediction

	// Tick N+1: server receives and processes the shot.
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
	z.processTick()
	magN1, ok := extractMagazineFromWorldState(*msgs, peerID)
	if !ok {
		t.Fatal("no WorldState from tick N+1")
	}
	*msgs = nil

	// Server's tick N+1 broadcast reflects the shot: magazine=29
	if magN1 != 29 {
		t.Errorf("tick N+1 magazine = %d, want 29", magN1)
	}

	// But the client received tick N's broadcast (magazine=30) AFTER it optimistically
	// decremented to 29. A naive slam would rollback to 30.
	// The correct fix: ignore server_mag > client_mag.
	if magN > uint8(clientMag) {
		t.Logf("EXPECTED: tick N broadcast (mag=%d) > client prediction (mag=%d) — "+
			"client must NOT rollback here", magN, clientMag)
	}

	// Verify the fix rule: only correct downward
	// server=30, client=29 → ignore (server behind, not a misfire)
	// server=29, client=29 → in sync
	// server=28, client=29 → correct down (server consumed extra, unlikely but safe)
	if magN1 <= uint8(clientMag) {
		t.Logf("OK: tick N+1 (mag=%d) <= client (mag=%d) — safe to accept", magN1, clientMag)
	}
}

// TestWorldState_SustainedFire_MagazineMonotonic verifies that during sustained
// fire, the magazine in successive world states is strictly non-increasing.
// Each shot decrements by 1 on the tick it fires.
func TestWorldState_SustainedFire_MagazineMonotonic(t *testing.T) {
	z, peerID := setupFightZone(t)
	boss := findBoss(z)
	boss.Health = 1e6

	send, msgs := captureSend()
	z.world.Clients[peerID].Send = send

	// Initialize
	z.processTick()
	*msgs = nil

	aimPitch := z.world.Players[peerID].AimPitch
	const ticksPerShot = 4
	const shots = 10

	var magSequence []uint8
	prevMag := uint8(30)

	for shot := range shots {
		z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(aimPitch))
		z.processTick()

		mag, ok := extractMagazineFromWorldState(*msgs, peerID)
		if !ok {
			t.Fatalf("no WorldState on shot %d", shot+1)
		}
		magSequence = append(magSequence, mag)

		if mag > prevMag {
			t.Errorf("magazine increased: %d → %d on shot %d (should be monotonically non-increasing)",
				prevMag, mag, shot+1)
		}
		prevMag = mag
		*msgs = nil

		// Drain cooldown
		for i := 1; i < ticksPerShot; i++ {
			z.processTick()
			*msgs = nil
		}
	}

	// Should go 29, 28, 27, ... 20
	if magSequence[0] != 29 {
		t.Errorf("first shot magazine = %d, want 29", magSequence[0])
	}
	if magSequence[shots-1] != uint8(30-shots) {
		t.Errorf("last shot magazine = %d, want %d", magSequence[shots-1], 30-shots)
	}
	t.Logf("magazine sequence: %v", magSequence)
}
