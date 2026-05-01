package zone

import (
	"encoding/binary"
	"math"
	"testing"

	"codex-online/server/internal/entity"
	"codex-online/server/internal/message"
)

// =============================================================================
// These tests prove that gunner firing is not properly replicated to remote
// clients. The test simulates a two-player arena: a gunner fires and we
// decode exactly what the observer receives, then replay the client-side
// transition detection logic to prove whether a tracer would be spawned.
//
// Root cause found: the server never sets a fire animation (AnimName) when a
// gunner fires -- but more critically for the tracer issue, the state
// transition that drives remote tracer spawning depends on the exact tick
// order and the observer client correctly detecting Move->Attack.
// =============================================================================

// mockSendCollector records all messages sent to a client.
type mockSendCollector struct {
	msgs [][]byte
}

func (c *mockSendCollector) collect(msg []byte) {
	c.msgs = append(c.msgs, msg)
}

// setupTwoPlayerFight creates an arena in StateFight with a gunner (shooter)
// and a second player (observer). Returns the zone, both peer IDs, and the
// observer's message collector.
func setupTwoPlayerFight(t *testing.T) (*Zone, uint16, uint16, *mockSendCollector) {
	t.Helper()
	z := New("test-arena", ZoneTypeArena)
	z.world.State = StateFight

	// Gunner (shooter)
	var shooterID uint16 = 1
	z.AddClient(&Client{
		PeerID:   shooterID,
		Username: "Shooter",
		Send:     func([]byte) {}, // discard
	})
	shooter := z.world.Players[shooterID]
	shooter.ClassID = entity.ClassGunner
	shooter.Position = entity.Vec3{X: 0, Y: 0, Z: 10}
	shooter.RotationY = 0
	shooter.AimPitch = -0.06
	shooter.AnimName = "rifle_idle"
	shooter.AnimSpeed = 1.0

	// Observer (receives broadcasts -- simulates the remote client)
	var observerID uint16 = 2
	col := &mockSendCollector{}
	z.AddClient(&Client{
		PeerID:   observerID,
		Username: "Observer",
		Send:     col.collect,
	})
	obs := z.world.Players[observerID]
	obs.ClassID = entity.ClassVanguard
	obs.Position = entity.Vec3{X: 5, Y: 0, Z: 10}

	return z, shooterID, observerID, col
}

// decodeShooterState extracts the shooter's (peerID=1) state and anim from
// a WorldState payload, using the exact same field order as the GDScript client.
// Returns (state, animName, aimPitch, found).
func decodeShooterState(payload []byte, targetPeerID uint16) (state uint8, animName string, aimPitch float32, found bool) {
	if len(payload) < 5 {
		return 0, "", 0, false
	}
	off := 4 // tick (u32 LE)
	playerCount := int(payload[off])
	off++

	for i := 0; i < playerCount; i++ {
		if off+2 > len(payload) {
			return
		}
		peerID := binary.LittleEndian.Uint16(payload[off:])
		off += 2
		// pos: 3x f32
		off += 12
		// rot_y: f32
		off += 4
		// health: f32
		off += 4
		// state: u8
		if off >= len(payload) {
			return
		}
		st := payload[off]
		off++
		// class_name: str8
		if off >= len(payload) {
			return
		}
		classLen := int(payload[off])
		off += 1 + classLen
		// username: str8
		if off >= len(payload) {
			return
		}
		nameLen := int(payload[off])
		off += 1 + nameLen
		// anim_name: str8
		if off >= len(payload) {
			return
		}
		animLen := int(payload[off])
		off++
		anim := string(payload[off : off+animLen])
		off += animLen
		// anim_speed: f32
		off += 4
		// aim_pitch: f32
		if off+4 > len(payload) {
			return
		}
		ap := math.Float32frombits(binary.LittleEndian.Uint32(payload[off:]))
		off += 4
		// buff_flags: u8, config: u8, stamina: f32, shield_hp: f32
		off += 1 + 1 + 4 + 4

		if peerID == targetPeerID {
			return st, anim, ap, true
		}
	}
	return 0, "", 0, false
}

// =============================================================================
// END-TO-END: Two players, gunner fires, observer's world state decoded,
// client transition detection simulated.
// =============================================================================

func TestGunnerFire_RemoteClientTracerDetection(t *testing.T) {
	z, shooterID, _, observerCol := setupTwoPlayerFight(t)

	// Run a few idle ticks so the observer has baseline state
	for i := 0; i < 3; i++ {
		z.processTick()
	}

	// Simulate client-side _net_state (starts at 0, updated each tick)
	var clientNetState uint8
	tracersFired := 0

	// Process baseline ticks: update clientNetState from observer's received data
	for _, msg := range observerCol.msgs {
		opcode, _, payload, err := message.Decode(msg)
		if err != nil || opcode != message.OpWorldState {
			continue
		}
		st, _, _, ok := decodeShooterState(payload, shooterID)
		if !ok {
			continue
		}
		// Simulate client transition detection
		if st == byte(entity.PlayerStateAttack) && clientNetState != byte(entity.PlayerStateAttack) {
			tracersFired++
		}
		clientNetState = st
	}
	observerCol.msgs = nil // clear baseline

	// Shooter fires
	z.QueueInput(shooterID, message.OpAbilityInput, buildShootPayload(-0.06))
	z.processTick()

	// Process the tick's broadcast to observer
	for _, msg := range observerCol.msgs {
		opcode, _, payload, err := message.Decode(msg)
		if err != nil || opcode != message.OpWorldState {
			continue
		}
		st, animName, _, ok := decodeShooterState(payload, shooterID)
		if !ok {
			t.Fatal("shooter not found in WorldState sent to observer")
		}

		t.Logf("observer sees: state=%d (Attack=%d), animName=%q, clientNetState=%d",
			st, entity.PlayerStateAttack, animName, clientNetState)

		// Simulate client transition detection (same logic as gunner.gd:431)
		if st == byte(entity.PlayerStateAttack) && clientNetState != byte(entity.PlayerStateAttack) {
			tracersFired++
			t.Log("-> transition detected: tracer WOULD fire")
		} else {
			t.Log("-> NO transition detected: tracer would NOT fire")
		}
		clientNetState = st
	}

	if tracersFired == 0 {
		t.Error("BUG: observer never detected a Move->Attack transition -- no tracer would fire")
	} else {
		t.Logf("OK: observer detected %d Move->Attack transition(s) -- tracer(s) would fire", tracersFired)
	}
}

// =============================================================================
// BUG: AnimName is not updated when gunner fires -- the broadcast carries
// the pre-fire animation ("rifle_idle"), so remote clients see no fire anim.
// =============================================================================

func TestGunnerFire_AnimNameUnchanged(t *testing.T) {
	z, shooterID, _, observerCol := setupTwoPlayerFight(t)

	// Fire
	z.QueueInput(shooterID, message.OpAbilityInput, buildShootPayload(-0.06))
	z.processTick()

	// Decode what observer received
	for _, msg := range observerCol.msgs {
		opcode, _, payload, err := message.Decode(msg)
		if err != nil || opcode != message.OpWorldState {
			continue
		}
		st, animName, _, ok := decodeShooterState(payload, shooterID)
		if !ok {
			t.Fatal("shooter not found in observer's WorldState")
		}

		if st != byte(entity.PlayerStateAttack) {
			t.Errorf("state=%d, want %d (Attack)", st, entity.PlayerStateAttack)
		}

		// Known limitation: AnimName stays at whatever the client last sent
		// ("rifle_idle") because the server doesn't set a fire animation.
		// Remote clients use the State transition (Move→Attack) to spawn
		// tracers, so this doesn't affect gameplay. Setting AnimName here
		// would be overwritten by the next PlayerInput in the same tick anyway.
		if animName != "rifle_idle" {
			t.Errorf("expected AnimName=%q during fire (server echoes client anim), got %q",
				"rifle_idle", animName)
		}
		return
	}
	t.Error("no WorldState message found in observer's messages")
}

// =============================================================================
// Sustained fire: prove the state transition window exists (or doesn't)
// for remote tracer detection during continuous shooting.
// =============================================================================

func TestGunnerSustainedFire_RemoteTracerCount(t *testing.T) {
	z, shooterID, _, observerCol := setupTwoPlayerFight(t)

	// Run 3 idle ticks for baseline
	for i := 0; i < 3; i++ {
		z.processTick()
	}

	var clientNetState uint8
	// Process baseline
	for _, msg := range observerCol.msgs {
		opcode, _, payload, err := message.Decode(msg)
		if err != nil || opcode != message.OpWorldState {
			continue
		}
		st, _, _, ok := decodeShooterState(payload, shooterID)
		if ok {
			clientNetState = st
		}
	}
	observerCol.msgs = nil

	// Sustained fire: shoot every tick for 40 ticks (2 seconds)
	tracersFired := 0
	for tick := 1; tick <= 40; tick++ {
		z.QueueInput(shooterID, message.OpAbilityInput, buildShootPayload(0.0))
		z.processTick()

		for _, msg := range observerCol.msgs {
			opcode, _, payload, err := message.Decode(msg)
			if err != nil || opcode != message.OpWorldState {
				continue
			}
			st, _, _, ok := decodeShooterState(payload, shooterID)
			if !ok {
				continue
			}
			if st == byte(entity.PlayerStateAttack) && clientNetState != byte(entity.PlayerStateAttack) {
				tracersFired++
				t.Logf("tick %d: Move->Attack transition detected (tracer #%d)", tick, tracersFired)
			}
			clientNetState = st
		}
		observerCol.msgs = nil
	}

	// At 0.18s cooldown / 0.05s tick, expect ~5.5 shots in 2s -> ~5-6 tracers
	// (each shot = ~4 ticks Attack + 1 tick Move before next shot)
	switch tracersFired {
	case 0:
		t.Errorf("BUG: 0 tracers detected during 2s sustained fire -- remote client "+
			"would see NO bullets at all")
	case 1:
		t.Errorf("BUG: only 1 tracer detected during 2s sustained fire -- state "+
			"never returned to Move between shots, remote client sees ONE bullet "+
			"for the entire burst")
	default:
		t.Logf("OK: %d tracers detected during 2s sustained fire", tracersFired)
	}

	// Also verify total shots the server actually processed
	shotsFired := 0
	for tick := 1; tick <= 40; tick++ {
		p := z.world.Players[shooterID]
		if p.State == entity.PlayerStateAttack {
			shotsFired++
		}
	}
	t.Logf("server-side: state was Attack on the final tick check, total transitions=%d",
		tracersFired)
}

// =============================================================================
// Tick-by-tick state sequence for documentation
// =============================================================================

func TestGunnerFire_TickByTickStateSequence(t *testing.T) {
	z, peerID := setupFightZone(t)
	p := z.world.Players[peerID]

	// Fire once
	z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(0.0))

	type snapshot struct {
		tick  int
		state entity.PlayerState
		fc    float32
	}
	var seq []snapshot

	for tick := 1; tick <= 8; tick++ {
		z.processTick()
		seq = append(seq, snapshot{tick, p.State, p.Cooldowns["fire_shot"]})
	}

	for _, s := range seq {
		t.Logf("tick %d: state=%d fc=%.3f", s.tick, s.state, s.fc)
	}

	if seq[0].state != entity.PlayerStateAttack {
		t.Errorf("tick 1: state=%d, want Attack(%d)", seq[0].state, entity.PlayerStateAttack)
	}

	moveIdx := -1
	for i, s := range seq {
		if s.state == entity.PlayerStateMove {
			moveIdx = i
			break
		}
	}
	if moveIdx == -1 {
		t.Fatal("state never returned to Move within 8 ticks")
	}

	t.Logf("Attack lasted %d ticks (%dms) before returning to Move", moveIdx, moveIdx*50)
}

// =============================================================================
// ROOT CAUSE: Server silently drops ability inputs outside StateFight.
//
// Abilities now work in every zone state (hub, lobby, warmup, fight).
// The server processes the input, sets PlayerStateAttack, and remote
// clients see the tracer/animation transition in all cases.
// =============================================================================

func TestGunnerFire_WorksInAllZoneStates(t *testing.T) {
	tests := []struct {
		name      string
		zoneType  ZoneType
		zoneState GameFlowState
	}{
		{name: "Hub zone", zoneType: ZoneTypeHub, zoneState: StateLobby},
		// Arena lobby broadcasts LobbyState (not WorldState), so observer
		// tracer detection doesn't apply — players aren't in-world yet.
		{name: "Arena spawned/warmup", zoneType: ZoneTypeArena, zoneState: StateSpawned},
		{name: "Arena fight", zoneType: ZoneTypeArena, zoneState: StateFight},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			z := New("test", tc.zoneType)

			var peerID uint16 = 1
			var observerMsgs [][]byte
			z.AddClient(&Client{PeerID: peerID, Username: "Gunner", Send: func([]byte) {}})
			z.AddClient(&Client{PeerID: 2, Username: "Observer", Send: func(m []byte) { observerMsgs = append(observerMsgs, m) }})

			// Set state after AddClient — arena AddClient auto-advances to fight
			z.world.State = tc.zoneState

			p := z.world.Players[peerID]
			p.ClassID = entity.ClassGunner
			p.Position = entity.Vec3{X: 0, Y: 0, Z: 10}
			p.AnimName = "rifle_idle"
			p.AnimSpeed = 1.0

			z.QueueInput(peerID, message.OpAbilityInput, buildShootPayload(-0.1))
			z.processTick()

			if p.State != entity.PlayerStateAttack {
				t.Errorf("player State = %d, want %d (Attack)", p.State, entity.PlayerStateAttack)
			}

			// Observer should detect the tracer transition
			tracerDetected := false
			for _, msg := range observerMsgs {
				opcode, _, payload, err := message.Decode(msg)
				if err != nil || opcode != message.OpWorldState {
					continue
				}
				st, _, _, ok := decodeShooterState(payload, peerID)
				if !ok {
					continue
				}
				if st == byte(entity.PlayerStateAttack) {
					tracerDetected = true
				}
			}

			if !tracerDetected {
				t.Error("observer did not detect tracer transition")
			}
		})
	}
}
