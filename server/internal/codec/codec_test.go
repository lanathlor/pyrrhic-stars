package codec

import (
	"encoding/binary"
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

const testName = "Alice"
const testNameBob = "Bob"
const testFireball = "fireball"
const testShield = "shield"

// =============================================================================
// Roundtrip: EncodePlayerInput → DecodePlayerInput
// =============================================================================

func TestPlayerInputRoundtrip(t *testing.T) {
	buf := EncodePlayerInput(nil, 1.5, 2.0, -3.5, 0.7, 42, 5, -0.3)
	msg, ok := DecodePlayerInput(buf)
	if !ok {
		t.Fatal("DecodePlayerInput returned !ok")
	}
	if msg.PosX != 1.5 {
		t.Errorf("PosX = %f, want 1.5", msg.PosX)
	}
	if msg.PosY != 2.0 {
		t.Errorf("PosY = %f, want 2.0", msg.PosY)
	}
	if msg.PosZ != -3.5 {
		t.Errorf("PosZ = %f, want -3.5", msg.PosZ)
	}
	if msg.RotY != 0.7 {
		t.Errorf("RotY = %f, want 0.7", msg.RotY)
	}
	if msg.Tick != 42 {
		t.Errorf("Tick = %d, want 42", msg.Tick)
	}
	if msg.VisualState != 5 {
		t.Errorf("VisualState = %d, want 5", msg.VisualState)
	}
	if msg.AimPitch != -0.3 {
		t.Errorf("AimPitch = %f, want -0.3", msg.AimPitch)
	}
}

func TestPlayerInputMinimal(t *testing.T) {
	// 16 bytes: position + rotation only
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint32(buf[0:], math.Float32bits(1.0))
	binary.LittleEndian.PutUint32(buf[4:], math.Float32bits(2.0))
	binary.LittleEndian.PutUint32(buf[8:], math.Float32bits(3.0))
	binary.LittleEndian.PutUint32(buf[12:], math.Float32bits(0.5))

	msg, ok := DecodePlayerInput(buf)
	if !ok {
		t.Fatal("DecodePlayerInput returned !ok for 16-byte payload")
	}
	if msg.PosX != 1.0 || msg.PosY != 2.0 || msg.PosZ != 3.0 {
		t.Errorf("position = (%f,%f,%f), want (1,2,3)", msg.PosX, msg.PosY, msg.PosZ)
	}
	if msg.Tick != 0 {
		t.Errorf("Tick = %d, want 0", msg.Tick)
	}
	if msg.VisualState != 0 {
		t.Errorf("VisualState = %d, want 0", msg.VisualState)
	}
}

func TestPlayerInputTooShort(t *testing.T) {
	if _, ok := DecodePlayerInput(nil); ok {
		t.Error("nil payload should return !ok")
	}
	if _, ok := DecodePlayerInput([]byte{1, 2, 3}); ok {
		t.Error("3-byte payload should return !ok")
	}
}

// =============================================================================
// Roundtrip: EncodeAbilityInput → DecodeAbilityInput
// =============================================================================

func TestAbilityInputRoundtrip(t *testing.T) {
	buf := EncodeAbilityInput(1, -0.15)
	msg := DecodeAbilityInput(buf)
	if msg == nil {
		t.Fatal("DecodeAbilityInput returned nil")
	}
	if msg.Action != 1 {
		t.Errorf("Action = %d, want 1", msg.Action)
	}
	if msg.AimPitch != -0.15 {
		t.Errorf("AimPitch = %f, want -0.15", msg.AimPitch)
	}
}

func TestAbilityInputActionOnly(t *testing.T) {
	msg := DecodeAbilityInput([]byte{3})
	if msg == nil {
		t.Fatal("DecodeAbilityInput returned nil for 1-byte payload")
	}
	if msg.Action != 3 {
		t.Errorf("Action = %d, want 3", msg.Action)
	}
	if msg.AimPitch != 0 {
		t.Errorf("AimPitch = %f, want 0", msg.AimPitch)
	}
}

func TestAbilityInputTooShort(t *testing.T) {
	if DecodeAbilityInput(nil) != nil {
		t.Error("nil payload should return nil")
	}
}

// =============================================================================
// DecodeInteractInput
// =============================================================================

func TestInteractInputClassSelect(t *testing.T) {
	// [action:0][nameLen:6][gunner]
	payload := []byte{0, 6}
	payload = append(payload, []byte(entity.ClassGunner)...)
	msg, ok := DecodeInteractInput(payload)
	if !ok {
		t.Fatal("DecodeInteractInput returned !ok")
	}
	if msg.Action != 0 {
		t.Errorf("Action = %d, want 0", msg.Action)
	}
	if msg.ClassName != entity.ClassGunner {
		t.Errorf("ClassName = %q, want %q", msg.ClassName, entity.ClassGunner)
	}
}

func TestInteractInputReadyToggle(t *testing.T) {
	msg, ok := DecodeInteractInput([]byte{1})
	if !ok {
		t.Fatal("DecodeInteractInput returned !ok")
	}
	if msg.Action != 1 {
		t.Errorf("Action = %d, want 1", msg.Action)
	}
	if msg.ClassName != "" {
		t.Errorf("ClassName = %q, want empty", msg.ClassName)
	}
}

func TestInteractInputTooShort(t *testing.T) {
	if _, ok := DecodeInteractInput(nil); ok {
		t.Error("nil payload should return !ok")
	}
}

// =============================================================================
// DecodeRespawnRequest
// =============================================================================

func TestDecodeRespawnRequest(t *testing.T) {
	tests := []struct {
		payload  []byte
		wantType uint8
		wantOK   bool
	}{
		{[]byte{0}, 0, true},
		{[]byte{1}, 1, true},
		{nil, 0, false},
		{[]byte{}, 0, false},
	}
	for _, tc := range tests {
		typ, ok := DecodeRespawnRequest(tc.payload)
		if ok != tc.wantOK || typ != tc.wantType {
			t.Errorf("DecodeRespawnRequest(%v) = (%d, %v), want (%d, %v)",
				tc.payload, typ, ok, tc.wantType, tc.wantOK)
		}
	}
}

// =============================================================================
// EncodeWorldState — wire format verification
// =============================================================================

func TestWorldStateProjectileRoundtrip(t *testing.T) {
	p := &entity.Player{
		Combatant: entity.Combatant{
			ID:        1,
			Position:  entity.Vec3{X: 1.0},
			Health:    100.0,
			MaxHealth: 150.0,
		},
		ClassID:      entity.ClassGunner,
		Resources:    make(map[string]*entity.Resource),
		AbilityState: make(map[string]any),
	}
	e := entity.NewEnemy(0, 2000.0, "guard_captain")

	projs := []*entity.Projectile{
		{ID: 10, Position: entity.Vec3{X: 3, Y: 1, Z: -2}, Direction: entity.Vec3{X: 0, Z: 1}, Speed: 22.0, AngularVelocity: 0.5, VisualTag: testFireball},
		{ID: 11, Position: entity.Vec3{X: 5, Z: 4}, Direction: entity.Vec3{X: 1}, Speed: 15.0},
	}

	buf := EncodeWorldState(42, map[uint16]*entity.Player{1: p}, []*entity.Enemy{e}, projs)
	ws, ok := DecodeWorldState(buf)
	if !ok {
		t.Fatal("DecodeWorldState failed")
	}
	if ws.Tick != 42 {
		t.Errorf("tick = %d, want 42", ws.Tick)
	}
	if len(ws.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(ws.Players))
	}
	if len(ws.Enemies) != 1 {
		t.Fatalf("enemies = %d, want 1", len(ws.Enemies))
	}
	if len(ws.Projectiles) != 2 {
		t.Fatalf("projectiles = %d, want 2", len(ws.Projectiles))
	}

	proj0 := ws.Projectiles[0]
	if proj0.ID != 10 {
		t.Errorf("proj[0].ID = %d, want 10", proj0.ID)
	}
	if proj0.PosX != 3 || proj0.PosY != 1 || proj0.PosZ != -2 {
		t.Errorf("proj[0].Pos = (%f,%f,%f), want (3,1,-2)", proj0.PosX, proj0.PosY, proj0.PosZ)
	}
	if proj0.Speed != 22.0 {
		t.Errorf("proj[0].Speed = %f, want 22", proj0.Speed)
	}
	if proj0.AngularVelocity != 0.5 {
		t.Errorf("proj[0].AngularVelocity = %f, want 0.5", proj0.AngularVelocity)
	}
	if proj0.VisualTag != testFireball {
		t.Errorf("proj[0].VisualTag = %q, want %q", proj0.VisualTag, testFireball)
	}

	proj1 := ws.Projectiles[1]
	if proj1.ID != 11 {
		t.Errorf("proj[1].ID = %d, want 11", proj1.ID)
	}
	if proj1.Speed != 15.0 {
		t.Errorf("proj[1].Speed = %f, want 15", proj1.Speed)
	}
	if proj1.VisualTag != "" {
		t.Errorf("proj[1].VisualTag = %q, want empty", proj1.VisualTag)
	}
}

func TestWorldStateFluxRoundtrip(t *testing.T) {
	p := &entity.Player{
		Combatant: entity.Combatant{
			ID:        1,
			Position:  entity.Vec3{X: 1.0},
			Health:    100.0,
			MaxHealth: 150.0,
		},
		ClassID: entity.ClassGunner,
		Resources: map[string]*entity.Resource{
			"flux":                {Current: 75.5, Max: 160.0},
			"stamina":             {Current: 50.0},
			entity.ResourceShield: {Current: 25.0},
			"munitions":           {Current: 10.0},
			"resonance":           {Current: 30.0},
		},
		AbilityState: make(map[string]any),
	}

	buf := EncodeWorldState(1, map[uint16]*entity.Player{1: p}, nil, nil)
	ws, ok := DecodeWorldState(buf)
	if !ok {
		t.Fatal("DecodeWorldState failed")
	}
	if len(ws.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(ws.Players))
	}
	dp := ws.Players[0]
	if dp.Stamina != 50.0 {
		t.Errorf("Stamina = %f, want 50.0", dp.Stamina)
	}
	if dp.ShieldHP != 25.0 {
		t.Errorf("ShieldHP = %f, want 25.0", dp.ShieldHP)
	}
	if dp.Munitions != 10.0 {
		t.Errorf("Munitions = %f, want 10.0", dp.Munitions)
	}
	if dp.Resonance != 30.0 {
		t.Errorf("Resonance = %f, want 30.0", dp.Resonance)
	}
	if dp.Flux != 75.5 {
		t.Errorf("Flux = %f, want 75.5", dp.Flux)
	}
	if dp.MaxFlux != 160.0 {
		t.Errorf("MaxFlux = %f, want 160.0", dp.MaxFlux)
	}
}

// TestWorldStateArcanotechFluxPoolsNoDesync guards against a wire desync where
// appendFluxCommitPools wrote count=len(pools) but always emitted all 4 schools.
// With the default Harmonist loadout (2 committed schools) the count said 2 while
// 4 schools' worth of bytes were written, leaving 16 stray bytes that corrupted
// every section after the player block. In game this made enemies and NPCs
// (the hub merchants) vanish for Arcanotechnicien players only.
func TestWorldStateArcanotechFluxPoolsNoDesync(t *testing.T) {
	p := &entity.Player{
		Combatant: entity.Combatant{ID: 1, Health: 100.0, MaxHealth: 150.0},
		ClassID:   entity.ClassArcanotechnicien,
		Resources: map[string]*entity.Resource{
			"flux": {Current: 40.0, Max: 100.0},
		},
		AbilityState: make(map[string]any),
		FluxCommit:   &entity.FluxCommitment{TotalMax: 100.0, TotalRegen: 10.0},
	}
	// Default Harmonist: only 2 of the 4 schools are committed.
	p.FluxCommit.SetCommitment(map[string]float32{
		entity.SchoolBioarcanotechnic: 0.5,
		entity.SchoolBiometabolic:     0.5,
	})

	// A trailing entity that must survive decode: if the player block is
	// mis-sized, this gets eaten by the desync (mirrors the missing merchants).
	enemy := &entity.Enemy{Combatant: entity.Combatant{ID: 1001, Health: 50.0, MaxHealth: 50.0, Alive: true}}

	buf := EncodeWorldState(7, map[uint16]*entity.Player{1: p}, []*entity.Enemy{enemy}, nil)
	ws, ok := DecodeWorldState(buf)
	if !ok {
		t.Fatal("DecodeWorldState failed")
	}
	if len(ws.Enemies) != 1 {
		t.Fatalf("enemies = %d, want 1 (flux pool desync ate the trailing entity)", len(ws.Enemies))
	}
	if ws.Enemies[0].EnemyID != 1001 {
		t.Errorf("enemy id = %d, want 1001 (flux pool desync corrupted trailing data)", ws.Enemies[0].EnemyID)
	}
}

func TestDecodeWorldStateNilProjectiles(t *testing.T) {
	buf := EncodeWorldState(1, nil, nil, nil)
	ws, ok := DecodeWorldState(buf)
	if !ok {
		t.Fatal("DecodeWorldState failed")
	}
	if len(ws.Projectiles) != 0 {
		t.Errorf("projectiles = %d, want 0", len(ws.Projectiles))
	}
}

func TestEncodeWorldStateWireFormat(t *testing.T) {
	p := &entity.Player{
		Combatant: entity.Combatant{
			ID:        7,
			Position:  entity.Vec3{X: 1.0, Y: 2.0, Z: 3.0},
			RotationY: 0.5,
			Health:    100.0,
			MaxHealth: 150.0,
		},
		State:        entity.PlayerStateAttack,
		ClassID:      entity.ClassGunner,
		Username:     testName,
		AimPitch:     -0.1,
		Resources:    make(map[string]*entity.Resource),
		AbilityState: make(map[string]any),
	}
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Position = entity.Vec3{X: 5.0, Y: 0.0, Z: -3.0}

	buf := EncodeWorldState(10, map[uint16]*entity.Player{1: p}, []*entity.Enemy{e}, nil)

	off := 0
	// tick
	tick := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if tick != 10 {
		t.Errorf("tick = %d, want 10", tick)
	}
	// player count
	if buf[off] != 1 {
		t.Fatalf("player_count = %d, want 1", buf[off])
	}
	off++
	// peer_id
	peerID := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if peerID != 7 {
		t.Errorf("peer_id = %d, want 7", peerID)
	}
	// pos
	posX := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if posX != 1.0 {
		t.Errorf("pos_x = %f, want 1.0", posX)
	}
	off += 4 // pos_y
	off += 4 // pos_z
	off += 4 // rot_y
	// health
	health := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if health != 100.0 {
		t.Errorf("health = %f, want 100.0", health)
	}
	// max_health
	maxHealth := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if maxHealth != 150.0 {
		t.Errorf("max_health = %f, want 150.0", maxHealth)
	}
	// state
	if buf[off] != byte(entity.PlayerStateAttack) {
		t.Errorf("state = %d, want %d", buf[off], entity.PlayerStateAttack)
	}
	off++
	// class:str8
	classLen := int(buf[off])
	off++
	className := string(buf[off : off+classLen])
	off += classLen
	if className != entity.ClassGunner {
		t.Errorf("class = %q, want %q", className, entity.ClassGunner)
	}
	// spec:str8
	specLen := int(buf[off])
	off++
	specName := string(buf[off : off+specLen])
	off += specLen
	if specName != "" {
		t.Errorf("spec = %q, want %q", specName, "")
	}
	// name:str8
	nameLen := int(buf[off])
	off++
	name := string(buf[off : off+nameLen])
	off += nameLen
	if name != testName {
		t.Errorf("name = %q, want %q", name, testName)
	}
	// visual_state
	visualState := buf[off]
	off++
	if visualState != 0 {
		t.Errorf("visual_state = %d, want 0", visualState)
	}
	// aim_pitch
	aimPitch := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if aimPitch != -0.1 {
		t.Errorf("aim_pitch = %f, want -0.1", aimPitch)
	}
	// buff_flags (1 byte) + config (1 byte) + stamina (4 bytes)
	buffFlags := buf[off]
	off++
	if buffFlags != 0 {
		t.Errorf("buff_flags = %d, want 0", buffFlags)
	}
	config := buf[off]
	off++
	if config != 0 {
		t.Errorf("config = %d, want 0", config)
	}
	staminaVal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if staminaVal != 0.0 {
		t.Errorf("stamina = %f, want 0.0 (gunner has no stamina)", staminaVal)
	}
	shieldVal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if shieldVal != 0.0 {
		t.Errorf("shield = %f, want 0.0", shieldVal)
	}
	munitionsVal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if munitionsVal != 0.0 {
		t.Errorf("munitions = %f, want 0.0", munitionsVal)
	}
	resonanceVal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if resonanceVal != 0.0 {
		t.Errorf("resonance = %f, want 0.0", resonanceVal)
	}
	fluxVal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if fluxVal != 0.0 {
		t.Errorf("flux = %f, want 0.0", fluxVal)
	}
	maxFluxVal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if maxFluxVal != 0.0 {
		t.Errorf("max_flux = %f, want 0.0", maxFluxVal)
	}
	// onslaught_stacks (1 byte)
	if buf[off] != 0 {
		t.Errorf("onslaught_stacks = %d, want 0 (gunner)", buf[off])
	}
	off++
	// Gunner assault state (7 bytes — all zero, no assault state initialized)
	for i := range 7 {
		if buf[off+i] != 0 {
			t.Errorf("assault_byte[%d] = %d, want 0", i, buf[off+i])
		}
	}
	off += 7
	// speed_mult (1 byte, 255 = 1.0 for non-blocking player)
	if buf[off] != 255 {
		t.Errorf("speed_mult = %d, want 255", buf[off])
	}
	off++
	// flux_pool_count (0 for non-Arcanotechnicien)
	if buf[off] != 0 {
		t.Errorf("flux_pool_count = %d, want 0 (gunner)", buf[off])
	}
	off++

	// enemy count
	if buf[off] != 1 {
		t.Errorf("enemy_count = %d, want 1", buf[off])
	}
	off++
	// enemy alive
	if buf[off] != 1 {
		t.Errorf("enemy_alive = %d, want 1", buf[off])
	}
	off++
	// enemy ID
	off += 2
	// enemy position
	enemyX := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if enemyX != 5.0 {
		t.Errorf("enemy_x = %f, want 5.0", enemyX)
	}

	// Skip remaining enemy fields: pos_y(4) pos_z(4) rot_y(4) health(4)
	// state(1) phase(1) max_health(4) def_name(str8) ranged(3*4) charge(3*4)
	off += 4 + 4 + 4 + 4 + 1 + 1 + 4 // pos_y, pos_z, rot_y, health, state, phase, max_health
	defNameLen := int(buf[off])
	off++
	off += defNameLen // def_name string bytes
	off += 4 * 6      // ranged_target(3) + charge_dir(3)
	off += 4 * 2      // melee_cone_angle + melee_range

	// projectile count
	if buf[off] != 0 {
		t.Errorf("proj_count = %d, want 0", buf[off])
	}
}

func TestEncodeWorldStateZeroPlayers(t *testing.T) {
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	buf := EncodeWorldState(1, nil, []*entity.Enemy{e}, nil)
	// tick(4) + player_count(1) + enemy_count(1) + enemy_data + proj_count(1)
	if len(buf) < 6 {
		t.Errorf("len = %d, want 55", len(buf))
	}
	if buf[4] != 0 {
		t.Errorf("player_count = %d, want 0", buf[4])
	}
}

// =============================================================================
// EncodeWorldState — nil enemy (hub zones)
// =============================================================================

func TestEncodeWorldStateNoEnemies(t *testing.T) {
	buf := EncodeWorldState(1, nil, nil, nil)
	// tick(4) + player_count(1) + enemy_count(1) + proj_count(1) + npc_count(1) = 8
	if len(buf) != 8 {
		t.Errorf("len = %d, want 8", len(buf))
	}
	if buf[4] != 0 {
		t.Errorf("player_count = %d, want 0", buf[4])
	}
	if buf[5] != 0 {
		t.Errorf("enemy_count = %d, want 0", buf[5])
	}
	if buf[6] != 0 {
		t.Errorf("proj_count = %d, want 0", buf[6])
	}
	if buf[7] != 0 {
		t.Errorf("npc_count = %d, want 0", buf[7])
	}
}

// =============================================================================
// EncodeLobbyState — wire format verification
// =============================================================================

func TestEncodeLobbyStateWireFormat(t *testing.T) {
	infos := []LobbyPlayerInfo{
		{PeerID: 1, ClassName: entity.ClassGunner, Username: testName, Ready: true},
		{PeerID: 2, ClassName: entity.ClassVanguard, Username: testNameBob, Ready: false},
	}
	buf := EncodeLobbyState(LobbyPhaseCountdown, 3, infos)

	if buf[0] != LobbyPhaseCountdown {
		t.Fatalf("phase = %d, want %d", buf[0], LobbyPhaseCountdown)
	}
	if buf[1] != 3 {
		t.Fatalf("countdown_secs = %d, want 3", buf[1])
	}
	if buf[2] != 2 {
		t.Fatalf("player_count = %d, want 2", buf[2])
	}
	off := 3
	// Player 1
	p1 := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if p1 != 1 {
		t.Errorf("p1 peer_id = %d, want 1", p1)
	}
	classLen := int(buf[off])
	off++
	class := string(buf[off : off+classLen])
	off += classLen
	if class != entity.ClassGunner {
		t.Errorf("p1 class = %q, want %q", class, entity.ClassGunner)
	}
	specLen := int(buf[off])
	off++
	spec := string(buf[off : off+specLen])
	off += specLen
	if spec != "" {
		t.Errorf("p1 spec = %q, want %q", spec, "")
	}
	nameLen := int(buf[off])
	off++
	name := string(buf[off : off+nameLen])
	off += nameLen
	if name != testName {
		t.Errorf("p1 username = %q, want %q", name, testName)
	}
	if buf[off] != 1 {
		t.Errorf("p1 ready = %d, want 1", buf[off])
	}
	off++
	// Player 2
	p2 := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if p2 != 2 {
		t.Errorf("p2 peer_id = %d, want 2", p2)
	}
	classLen = int(buf[off])
	off++
	class = string(buf[off : off+classLen])
	off += classLen
	if class != entity.ClassVanguard {
		t.Errorf("p2 class = %q, want %q", class, entity.ClassVanguard)
	}
	specLen = int(buf[off])
	off++
	spec = string(buf[off : off+specLen])
	off += specLen
	if spec != "" {
		t.Errorf("p2 spec = %q, want %q", spec, "")
	}
	nameLen = int(buf[off])
	off++
	name = string(buf[off : off+nameLen])
	off += nameLen
	if name != testNameBob {
		t.Errorf("p2 username = %q, want %q", name, testNameBob)
	}
	if buf[off] != 0 {
		t.Errorf("p2 ready = %d, want 0", buf[off])
	}
}

// =============================================================================
// EncodeDamageEvent — wire format verification
// =============================================================================

func TestEncodeDamageEventWireFormat(t *testing.T) {
	buf := EncodeDamageEvent(0, 42, 10.0, 1.5, 2.0, -3.5, 0 /* SourcePlayerAttack */, 0)

	if len(buf) != 25 {
		t.Fatalf("len = %d, want 25", len(buf))
	}

	off := 0
	target := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	source := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	amount := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	hitX := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	hitY := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	hitZ := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	srcType := buf[off]
	off++
	overheal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))

	if target != 0 {
		t.Errorf("target = %d, want 0", target)
	}
	if source != 42 {
		t.Errorf("source = %d, want 42", source)
	}
	if amount != 10.0 {
		t.Errorf("amount = %f, want 10.0", amount)
	}
	if hitX != 1.5 {
		t.Errorf("hit_x = %f, want 1.5", hitX)
	}
	if hitY != 2.0 {
		t.Errorf("hit_y = %f, want 2.0", hitY)
	}
	if hitZ != -3.5 {
		t.Errorf("hit_z = %f, want -3.5", hitZ)
	}
	if srcType != 0 {
		t.Errorf("source_type = %d, want 0", srcType)
	}
	if overheal != 0 {
		t.Errorf("overheal = %f, want 0", overheal)
	}
}

// =============================================================================
// EncodeGameFlow — wire format verification
// =============================================================================

func TestEncodeGameFlowWireFormat(t *testing.T) {
	buf := EncodeGameFlow(2, "fight")
	if buf[0] != 2 {
		t.Errorf("flow_type = %d, want 2", buf[0])
	}
	if buf[1] != 5 {
		t.Errorf("text_len = %d, want 5", buf[1])
	}
	if string(buf[2:]) != "fight" {
		t.Errorf("text = %q, want %q", string(buf[2:]), "fight")
	}
}

func TestEncodeGameFlowEmpty(t *testing.T) {
	buf := EncodeGameFlow(7, "")
	if len(buf) != 2 {
		t.Errorf("len = %d, want 2", len(buf))
	}
	if buf[0] != 7 || buf[1] != 0 {
		t.Errorf("buf = %v, want [7, 0]", buf)
	}
}

// =============================================================================
// EncodePeerID
// =============================================================================

func TestEncodePeerID(t *testing.T) {
	tests := []struct {
		id   uint16
		want []byte
	}{
		{0, []byte{0, 0}},
		{1, []byte{0, 1}},
		{256, []byte{1, 0}},
		{0xFFFF, []byte{0xFF, 0xFF}},
		{0x1234, []byte{0x12, 0x34}},
	}
	for _, tc := range tests {
		buf := EncodePeerID(tc.id)
		if len(buf) != 2 {
			t.Errorf("EncodePeerID(%d) len = %d, want 2", tc.id, len(buf))
			continue
		}
		got := binary.BigEndian.Uint16(buf)
		if got != tc.id {
			t.Errorf("EncodePeerID(%d) roundtrip = %d", tc.id, got)
		}
	}
}

// =============================================================================
// EncodeCharacterState
// =============================================================================

func TestEncodeCharacterState(t *testing.T) {
	c := CharacterInfo{
		ID:        42,
		ClassName: entity.ClassGunner,
		Name:      testName,
		PosX:      1.5,
		PosY:      2.0,
		PosZ:      -3.5,
		RotY:      0.7,
	}
	buf := EncodeCharacterState(c)

	off := 0
	// charID
	charID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if charID != 42 {
		t.Errorf("charID = %d, want 42", charID)
	}
	// class
	classLen := int(buf[off])
	off++
	className := string(buf[off : off+classLen])
	off += classLen
	if className != entity.ClassGunner {
		t.Errorf("class = %q, want %q", className, entity.ClassGunner)
	}
	// name
	nameLen := int(buf[off])
	off++
	name := string(buf[off : off+nameLen])
	off += nameLen
	if name != testName {
		t.Errorf("name = %q, want %q", name, testName)
	}
	// position
	posX := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if posX != 1.5 {
		t.Errorf("posX = %f, want 1.5", posX)
	}
	posY := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if posY != 2.0 {
		t.Errorf("posY = %f, want 2.0", posY)
	}
	posZ := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if posZ != -3.5 {
		t.Errorf("posZ = %f, want -3.5", posZ)
	}
	rotY := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if rotY != 0.7 {
		t.Errorf("rotY = %f, want 0.7", rotY)
	}
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

func TestEncodeCharacterStateEmpty(t *testing.T) {
	c := CharacterInfo{ID: 1}
	buf := EncodeCharacterState(c)
	// 4 (id) + 1 (classLen=0) + 1 (nameLen=0) + 16 (4 floats) = 22
	if len(buf) != 22 {
		t.Errorf("len = %d, want 22", len(buf))
	}
}

// =============================================================================
// EncodeCharacterList
// =============================================================================

func TestEncodeCharacterList(t *testing.T) {
	chars := []CharacterInfo{
		{ID: 10, ClassName: entity.ClassGunner, Name: testName, PosX: 1, PosY: 2, PosZ: 3, RotY: 0.5},
		{ID: 20, ClassName: entity.ClassVanguard, Name: testNameBob},
	}
	buf := EncodeCharacterList("TestUser", chars, 10)

	off := 0
	// username
	usernameLen := int(buf[off])
	off++
	username := string(buf[off : off+usernameLen])
	off += usernameLen
	if username != "TestUser" {
		t.Errorf("username = %q, want %q", username, "TestUser")
	}
	// count
	count := int(buf[off])
	off++
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	// char 1
	c1ID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if c1ID != 10 {
		t.Errorf("char1 ID = %d, want 10", c1ID)
	}
	c1ClassLen := int(buf[off])
	off++
	c1Class := string(buf[off : off+c1ClassLen])
	off += c1ClassLen
	if c1Class != entity.ClassGunner {
		t.Errorf("char1 class = %q, want %q", c1Class, entity.ClassGunner)
	}
	c1NameLen := int(buf[off])
	off++
	c1Name := string(buf[off : off+c1NameLen])
	off += c1NameLen
	if c1Name != testName {
		t.Errorf("char1 name = %q, want %q", c1Name, testName)
	}
	off += 16 // skip 4 floats
	// char 2
	c2ID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if c2ID != 20 {
		t.Errorf("char2 ID = %d, want 20", c2ID)
	}
	c2ClassLen := int(buf[off])
	off++
	off += c2ClassLen // skip class
	c2NameLen := int(buf[off])
	off++
	off += c2NameLen // skip name
	off += 16        // skip 4 floats
	// lastCharID
	lastID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if lastID != 10 {
		t.Errorf("lastCharID = %d, want 10", lastID)
	}
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

func TestEncodeCharacterListEmpty(t *testing.T) {
	buf := EncodeCharacterList("User", nil, 0)
	off := 0
	usernameLen := int(buf[off])
	off++
	off += usernameLen
	if buf[off] != 0 {
		t.Errorf("count = %d, want 0", buf[off])
	}
	off++
	lastID := binary.LittleEndian.Uint32(buf[off:])
	if lastID != 0 {
		t.Errorf("lastCharID = %d, want 0", lastID)
	}
}

// =============================================================================
// EncodeCharacterError
// =============================================================================

func TestEncodeCharacterError(t *testing.T) {
	tests := []struct {
		code uint8
		msg  string
	}{
		{1, "Name already taken"},
		{3, "Name must be 2-20 characters"},
		{5, ""},
	}
	for _, tc := range tests {
		buf := EncodeCharacterError(tc.code, tc.msg)
		if buf[0] != tc.code {
			t.Errorf("code = %d, want %d", buf[0], tc.code)
		}
		msgLen := int(buf[1])
		if msgLen != len(tc.msg) {
			t.Errorf("msgLen = %d, want %d", msgLen, len(tc.msg))
		}
		if string(buf[2:2+msgLen]) != tc.msg {
			t.Errorf("msg = %q, want %q", string(buf[2:2+msgLen]), tc.msg)
		}
	}
}

// =============================================================================
// EncodeGroupState
// =============================================================================

func TestEncodeGroupState(t *testing.T) {
	members := []GroupMemberInfo{
		{PeerID: 1, Username: testName},
		{PeerID: 2, Username: testNameBob},
	}
	buf := EncodeGroupState(99, 1, members)

	off := 0
	groupID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if groupID != 99 {
		t.Errorf("groupID = %d, want 99", groupID)
	}
	leaderPeer := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if leaderPeer != 1 {
		t.Errorf("leaderPeer = %d, want 1", leaderPeer)
	}
	count := int(buf[off])
	off++
	if count != 2 {
		t.Fatalf("member count = %d, want 2", count)
	}
	// member 1
	m1Peer := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if m1Peer != 1 {
		t.Errorf("m1 peer = %d, want 1", m1Peer)
	}
	m1NameLen := int(buf[off])
	off++
	m1Name := string(buf[off : off+m1NameLen])
	off += m1NameLen
	if m1Name != testName {
		t.Errorf("m1 name = %q, want %q", m1Name, testName)
	}
	// member 2
	m2Peer := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if m2Peer != 2 {
		t.Errorf("m2 peer = %d, want 2", m2Peer)
	}
	m2NameLen := int(buf[off])
	off++
	m2Name := string(buf[off : off+m2NameLen])
	off += m2NameLen
	if m2Name != testNameBob {
		t.Errorf("m2 name = %q, want %q", m2Name, testNameBob)
	}
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

// =============================================================================
// Friend encoders
// =============================================================================

// readStr8 reads a [len:u8][bytes] string starting at off, returning the string
// and the new offset.
func readStr8(b []byte, off int) (string, int) {
	n := int(b[off])
	off++
	s := string(b[off : off+n])
	return s, off + n
}

func TestEncodeFriendList(t *testing.T) {
	friends := []FriendInfo{
		{UserID: "uuid-1", Name: testName, Online: true},
		{UserID: "uuid-2", Name: testNameBob, Online: false},
	}
	buf := EncodeFriendList(friends)

	off := 0
	count := int(buf[off])
	off++
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	for i, want := range friends {
		var uid, name string
		uid, off = readStr8(buf, off)
		name, off = readStr8(buf, off)
		online := buf[off] == 1
		off++
		if uid != want.UserID || name != want.Name || online != want.Online {
			t.Errorf("entry %d = (%q,%q,%v), want (%q,%q,%v)", i, uid, name, online, want.UserID, want.Name, want.Online)
		}
	}
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

func TestEncodeFriendRequestRecv(t *testing.T) {
	buf := EncodeFriendRequestRecv("uuid-req", testName)
	uid, off := readStr8(buf, 0)
	name, off := readStr8(buf, off)
	if uid != "uuid-req" || name != testName {
		t.Errorf("got (%q,%q), want (uuid-req,%q)", uid, name, testName)
	}
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

func TestEncodeFriendStatus(t *testing.T) {
	buf := EncodeFriendStatus("uuid-x", true)
	uid, off := readStr8(buf, 0)
	if uid != "uuid-x" {
		t.Errorf("uid = %q, want uuid-x", uid)
	}
	if buf[off] != 1 {
		t.Errorf("online byte = %d, want 1", buf[off])
	}
	off++
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

func TestEncodeFriendError(t *testing.T) {
	buf := EncodeFriendError("nope")
	if buf[0] != 1 {
		t.Errorf("code = %d, want 1", buf[0])
	}
	msg, off := readStr8(buf, 1)
	if msg != "nope" {
		t.Errorf("msg = %q, want nope", msg)
	}
	if off != len(buf) {
		t.Errorf("consumed %d bytes, buf has %d", off, len(buf))
	}
}

func TestEncodeGroupStateNoMembers(t *testing.T) {
	buf := EncodeGroupState(1, 0, nil)
	// 4 (groupID) + 2 (leader) + 1 (count=0) = 7
	if len(buf) != 7 {
		t.Errorf("len = %d, want 7", len(buf))
	}
}

// =============================================================================
// EncodeGroupError
// =============================================================================

func TestEncodeGroupError(t *testing.T) {
	buf := EncodeGroupError("player not found")
	if buf[0] != 1 {
		t.Errorf("error code = %d, want 1", buf[0])
	}
	msgLen := int(buf[1])
	msg := string(buf[2 : 2+msgLen])
	if msg != "player not found" {
		t.Errorf("msg = %q, want %q", msg, "player not found")
	}
}

func TestEncodeGroupErrorEmpty(t *testing.T) {
	buf := EncodeGroupError("")
	if len(buf) != 2 {
		t.Errorf("len = %d, want 2", len(buf))
	}
	if buf[0] != 1 || buf[1] != 0 {
		t.Errorf("buf = %v, want [1, 0]", buf)
	}
}

// =============================================================================
// EncodeGroupInviteRecv
// =============================================================================

func TestEncodeGroupInviteRecv(t *testing.T) {
	buf := EncodeGroupInviteRecv(42, testName)
	off := 0
	groupID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if groupID != 42 {
		t.Errorf("groupID = %d, want 42", groupID)
	}
	nameLen := int(buf[off])
	off++
	name := string(buf[off : off+nameLen])
	if name != testName {
		t.Errorf("name = %q, want %q", name, testName)
	}
}

// =============================================================================
// EncodeEmptyGroupState
// =============================================================================

func TestEncodeEmptyGroupState(t *testing.T) {
	buf := EncodeEmptyGroupState()
	if len(buf) != 7 {
		t.Errorf("len = %d, want 7", len(buf))
	}
	for i, b := range buf {
		if b != 0 {
			t.Errorf("buf[%d] = %d, want 0", i, b)
		}
	}
}

// =============================================================================
// EncodeInteractInput roundtrip
// =============================================================================

func TestEncodeInteractInputRoundtrip(t *testing.T) {
	buf := EncodeInteractInput(0, entity.ClassVanguard)
	msg, ok := DecodeInteractInput(buf)
	if !ok {
		t.Fatal("DecodeInteractInput returned !ok")
	}
	if msg.Action != 0 {
		t.Errorf("Action = %d, want 0", msg.Action)
	}
	if msg.ClassName != entity.ClassVanguard {
		t.Errorf("ClassName = %q, want %q", msg.ClassName, entity.ClassVanguard)
	}
}

func TestEncodeInteractInputEmpty(t *testing.T) {
	buf := EncodeInteractInput(1, "")
	msg, ok := DecodeInteractInput(buf)
	if !ok {
		t.Fatal("DecodeInteractInput returned !ok")
	}
	if msg.Action != 1 {
		t.Errorf("Action = %d, want 1", msg.Action)
	}
	if msg.ClassName != "" {
		t.Errorf("ClassName = %q, want empty", msg.ClassName)
	}
}

// =============================================================================
// EncodeRespawnRequest roundtrip
// =============================================================================

func TestEncodeRespawnRequestRoundtrip(t *testing.T) {
	for _, rt := range []uint8{0, 1, 255} {
		buf := EncodeRespawnRequest(rt)
		typ, ok := DecodeRespawnRequest(buf)
		if !ok {
			t.Errorf("DecodeRespawnRequest(%d) failed", rt)
		}
		if typ != rt {
			t.Errorf("type = %d, want %d", typ, rt)
		}
	}
}

// =============================================================================
// getU16 wire helper
// =============================================================================

// =============================================================================
// EncodeAbilityInputWithTarget roundtrip
// =============================================================================

func TestAbilityInputWithTarget_Roundtrip(t *testing.T) {
	buf := EncodeAbilityInputWithTarget(50, 0.1, 1.2, 42)
	if len(buf) != 11 {
		t.Fatalf("len = %d, want 11", len(buf))
	}
	msg := DecodeAbilityInput(buf)
	if msg == nil {
		t.Fatal("DecodeAbilityInput returned nil")
	}
	if msg.Action != 50 {
		t.Errorf("Action = %d, want 50", msg.Action)
	}
	if msg.AimPitch != 0.1 {
		t.Errorf("AimPitch = %f, want 0.1", msg.AimPitch)
	}
	if msg.RotY != 1.2 {
		t.Errorf("RotY = %f, want 1.2", msg.RotY)
	}
	if msg.TargetPeerID != 42 {
		t.Errorf("TargetPeerID = %d, want 42", msg.TargetPeerID)
	}
}

func TestAbilityInputWithoutTarget_BackwardCompat(t *testing.T) {
	// A 9-byte payload (no target) should decode with TargetPeerID == 0.
	buf := EncodeAbilityInput(7, -0.5, 2.0)
	if len(buf) != 9 {
		t.Fatalf("len = %d, want 9", len(buf))
	}
	msg := DecodeAbilityInput(buf)
	if msg == nil {
		t.Fatal("DecodeAbilityInput returned nil")
	}
	if msg.Action != 7 {
		t.Errorf("Action = %d, want 7", msg.Action)
	}
	if msg.AimPitch != -0.5 {
		t.Errorf("AimPitch = %f, want -0.5", msg.AimPitch)
	}
	if msg.RotY != 2.0 {
		t.Errorf("RotY = %f, want 2.0", msg.RotY)
	}
	if msg.TargetPeerID != 0 {
		t.Errorf("TargetPeerID = %d, want 0 (backward compat)", msg.TargetPeerID)
	}
}

func TestGetU16(t *testing.T) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, 0x1234)
	got := getU16(buf)
	if got != 0x1234 {
		t.Errorf("getU16 = 0x%04X, want 0x1234", got)
	}
}

// =============================================================================
// Arcanotechnicien WorldState encoding — Confluence stacks & channeling flag
// =============================================================================

func TestWorldStateArcanotechnicienEncoding(t *testing.T) {
	tests := []struct {
		name             string
		confluenceStacks int
		channelPhase     uint8
		wantStacks       uint8
		wantChannelFlag  bool
	}{
		{
			name:             "confluence 3 stacks, channeling",
			confluenceStacks: 3,
			channelPhase:     1,
			wantStacks:       3,
			wantChannelFlag:  true,
		},
		{
			name:             "confluence 5 stacks, not channeling",
			confluenceStacks: 5,
			channelPhase:     0,
			wantStacks:       5,
			wantChannelFlag:  false,
		},
		{
			name:             "zero stacks, channeling",
			confluenceStacks: 0,
			channelPhase:     1,
			wantStacks:       0,
			wantChannelFlag:  true,
		},
		{
			name:             "zero stacks, not channeling",
			confluenceStacks: 0,
			channelPhase:     0,
			wantStacks:       0,
			wantChannelFlag:  false,
		},
		{
			name:             "execute phase is not channeling",
			confluenceStacks: 2,
			channelPhase:     2,
			wantStacks:       2,
			wantChannelFlag:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &entity.Player{
				Combatant: entity.Combatant{
					ID:        1,
					Alive:     true,
					Health:    100,
					MaxHealth: 100,
				},
				ClassID:      entity.ClassArcanotechnicien,
				ChannelPhase: tc.channelPhase,
				Confluence:   &entity.ConfluenceState{Stacks: tc.confluenceStacks, MaxStacks: 5},
				Resources:    make(map[string]*entity.Resource),
				AbilityState: make(map[string]any),
			}

			buf := EncodeWorldState(1, map[uint16]*entity.Player{1: p}, nil, nil)
			ws, ok := DecodeWorldState(buf)
			if !ok {
				t.Fatal("DecodeWorldState failed")
			}
			if len(ws.Players) != 1 {
				t.Fatalf("players = %d, want 1", len(ws.Players))
			}

			dp := ws.Players[0]

			if dp.OnslaughtStacks != tc.wantStacks {
				t.Errorf("OnslaughtStacks = %d, want %d", dp.OnslaughtStacks, tc.wantStacks)
			}

			gotFlag := dp.BuffFlags&0x20 != 0
			if gotFlag != tc.wantChannelFlag {
				t.Errorf("channeling flag (0x20) = %v, want %v (BuffFlags=0x%02X)",
					gotFlag, tc.wantChannelFlag, dp.BuffFlags)
			}
		})
	}
}

// =============================================================================
// Loadout roundtrip tests
// =============================================================================

func TestLoadoutStateRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		slots [6]string
	}{
		{
			name:  "all slots filled",
			slots: [6]string{testFireball, "ice_lance", "arcane_blast", testShield, "blink", "meteor"},
		},
		{
			name:  "some empty slots",
			slots: [6]string{testFireball, "", "arcane_blast", "", "", "meteor"},
		},
		{
			name:  "all empty slots",
			slots: [6]string{"", "", "", "", "", ""},
		},
		{
			name:  "single slot filled",
			slots: [6]string{"", "", "", "", "", "heal"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := EncodeLoadoutState(tc.slots)
			got, ok := DecodeSetLoadout(buf)
			if !ok {
				t.Fatal("DecodeSetLoadout returned !ok")
			}
			for i := range 6 {
				if got[i] != tc.slots[i] {
					t.Errorf("slot[%d] = %q, want %q", i, got[i], tc.slots[i])
				}
			}
		})
	}
}

func TestLoadoutStateEmptySlots(t *testing.T) {
	slots := [6]string{"", "ice_lance", "", testShield, "", ""}
	buf := EncodeLoadoutState(slots)
	got, ok := DecodeSetLoadout(buf)
	if !ok {
		t.Fatal("DecodeSetLoadout returned !ok")
	}
	for i := range 6 {
		if got[i] != slots[i] {
			t.Errorf("slot[%d] = %q, want %q", i, got[i], slots[i])
		}
	}
	// Verify empty slots produce a 1-byte (length=0) encoding each.
	// Total: 4 empty slots * 1 byte + 2 filled slots * (1 + len) bytes
	wantLen := 4*1 + (1 + len("ice_lance")) + (1 + len(testShield))
	if len(buf) != wantLen {
		t.Errorf("encoded len = %d, want %d", len(buf), wantLen)
	}
}

func TestAbilityCatalogEncode(t *testing.T) {
	entries := []AbilityCatalogEntry{
		{
			ID:          testFireball,
			Name:        "Fireball",
			School:      "destruction",
			AbilityType: "damage",
			Delivery:    "projectile",
			FluxCost:    "30",
			Description: "Hurls a ball of fire at the target.",
			Cooldown:    1.5,
			CommitTime:  2.0,
			Implemented: true,
			Affinity:    "primary",
		},
		{
			ID:          "heal",
			Name:        "Heal",
			School:      "restoration",
			AbilityType: "heal",
			Delivery:    "direct",
			FluxCost:    "50",
			Description: "Restores health to the target.",
			Cooldown:    0.0,
			CommitTime:  1.0,
			Implemented: false,
			Affinity:    "secondary",
		},
		{
			ID:          testShield,
			Name:        "Shield",
			School:      "protection",
			AbilityType: "buff",
			Delivery:    "self",
			FluxCost:    "20",
			Description: "Grants a protective barrier.",
			Cooldown:    10.0,
			CommitTime:  0.0,
			Implemented: true,
			Affinity:    "off",
		},
	}

	buf := EncodeAbilityCatalog(entries)

	// First byte is the count.
	if buf[0] != 3 {
		t.Errorf("count byte = %d, want 3", buf[0])
	}

	// Verify total length is reasonable: count(1) + per entry overhead.
	// Each entry: 6 str8 fields + 1 str16 field + 2 f32 + 1 u8 + 1 str8
	// = (1+len)*7 for str8s + (2+len) for str16 + 4*2 for f32s + 1 for bool
	// Minimum per entry with all empty strings: 7 + 2 + 8 + 1 = 18 bytes.
	if len(buf) < 1+3*18 {
		t.Errorf("encoded len = %d, suspiciously small", len(buf))
	}

	// Verify we can read back the first entry's ID as a spot check.
	off := 1
	idLen := int(buf[off])
	off++
	id := string(buf[off : off+idLen])
	if id != testFireball {
		t.Errorf("first entry ID = %q, want %q", id, testFireball)
	}
}

func TestDecodeSetLoadoutTooShort(t *testing.T) {
	// A valid 6-slot loadout with all empty strings is 6 bytes (6 zero-length prefixes).
	// Anything shorter should fail.
	if _, ok := DecodeSetLoadout(nil); ok {
		t.Error("nil payload should return !ok")
	}
	if _, ok := DecodeSetLoadout([]byte{0, 0, 0, 0, 0}); ok {
		t.Error("5-byte payload (only 5 slots) should return !ok")
	}
	// Truncated string: length says 5 but only 2 bytes follow.
	if _, ok := DecodeSetLoadout([]byte{5, 'a', 'b'}); ok {
		t.Error("truncated string payload should return !ok")
	}
}

// TestWorldStateArcanotechnicienFieldOrder verifies the exact wire order:
// resonance(f32) → flux(f32) → masteryStacks(u8). A swap between flux and
// masteryStacks would cause the client to read a u8 where a f32 is expected,
// corrupting all subsequent per-player data (the bug that caused the freeze).
func TestWorldStateArcanotechnicienFieldOrder(t *testing.T) {
	p := &entity.Player{
		Combatant: entity.Combatant{
			ID:        1,
			Alive:     true,
			Health:    120.0,
			MaxHealth: 170.0,
		},
		ClassID:      entity.ClassArcanotechnicien,
		ChannelPhase: 1,
		Confluence:   &entity.ConfluenceState{Stacks: 3, MaxStacks: 5},
		Resources: map[string]*entity.Resource{
			"flux":                {Current: 87.5, Max: 160.0},
			"resonance":           {Current: 42.0},
			"stamina":             {Current: 0},
			entity.ResourceShield: {Current: 0},
			"munitions":           {Current: 0},
		},
		AbilityState: make(map[string]any),
	}

	buf := EncodeWorldState(1, map[uint16]*entity.Player{1: p}, nil, nil)
	ws, ok := DecodeWorldState(buf)
	if !ok {
		t.Fatal("DecodeWorldState failed")
	}
	if len(ws.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(ws.Players))
	}

	dp := ws.Players[0]

	// These three fields are adjacent on the wire. If their order is wrong,
	// the float values will be garbage (reading a u8 as part of a f32).
	if dp.Resonance != 42.0 {
		t.Errorf("Resonance = %f, want 42.0", dp.Resonance)
	}
	if dp.Flux != 87.5 {
		t.Errorf("Flux = %f, want 87.5", dp.Flux)
	}
	if dp.MaxFlux != 160.0 {
		t.Errorf("MaxFlux = %f, want 160.0", dp.MaxFlux)
	}
	if dp.OnslaughtStacks != 3 {
		t.Errorf("OnslaughtStacks (confluence) = %d, want 3", dp.OnslaughtStacks)
	}

	// Also verify health roundtrips (would be corrupted if earlier offset was wrong).
	if dp.Health != 120.0 {
		t.Errorf("Health = %f, want 120.0", dp.Health)
	}
	if dp.MaxHealth != 170.0 {
		t.Errorf("MaxHealth = %f, want 170.0", dp.MaxHealth)
	}
}

// =============================================================================
// DecodeMerchantInteract
// =============================================================================

func TestDecodeMerchantInteract_Valid(t *testing.T) {
	tier, ok := DecodeMerchantInteract([]byte{2})
	if !ok {
		t.Fatal("DecodeMerchantInteract returned !ok for valid 1-byte payload")
	}
	if tier != 2 {
		t.Errorf("tier = %d, want 2", tier)
	}
}

func TestDecodeMerchantInteract_ZeroTier(t *testing.T) {
	tier, ok := DecodeMerchantInteract([]byte{0})
	if !ok {
		t.Fatal("DecodeMerchantInteract returned !ok for tier 0")
	}
	if tier != 0 {
		t.Errorf("tier = %d, want 0", tier)
	}
}

func TestDecodeMerchantInteract_Empty(t *testing.T) {
	if _, ok := DecodeMerchantInteract(nil); ok {
		t.Error("nil payload should return !ok")
	}
	if _, ok := DecodeMerchantInteract([]byte{}); ok {
		t.Error("empty payload should return !ok")
	}
}

// =============================================================================
// DecodeMerchantBuy
// =============================================================================

func TestDecodeMerchantBuy_Valid(t *testing.T) {
	defID := "iron_helm"
	// [tier:u8][len:u8][defID:...]
	payload := []byte{1, byte(len(defID))}
	payload = append(payload, []byte(defID)...)

	tier, got, ok := DecodeMerchantBuy(payload)
	if !ok {
		t.Fatal("DecodeMerchantBuy returned !ok for valid payload")
	}
	if tier != 1 {
		t.Errorf("tier = %d, want 1", tier)
	}
	if got != defID {
		t.Errorf("defID = %q, want %q", got, defID)
	}
}

func TestDecodeMerchantBuy_TooShort(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"one byte", []byte{0}},
		{"two bytes", []byte{0, 3}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, ok := DecodeMerchantBuy(tc.payload); ok {
				t.Errorf("payload %v should return !ok", tc.payload)
			}
		})
	}
}

func TestDecodeMerchantBuy_TruncatedDefID(t *testing.T) {
	// Length byte claims 10 bytes but only 3 follow.
	payload := []byte{0, 10, 'a', 'b', 'c'}
	if _, _, ok := DecodeMerchantBuy(payload); ok {
		t.Error("truncated defID should return !ok")
	}
}

func TestDecodeMerchantBuy_EmptyDefID(t *testing.T) {
	// tier=0, len=0 (empty def ID) - valid minimal payload of 3+ bytes: [tier][len=0][nothing]
	// Minimum valid: [tier:1][len:1][len=0 means no bytes follow] = 2 bytes total but
	// the check is len < 3, so we need [tier][0x00] padded to 3 bytes.
	// Looking at decode.go: len < 3 check triggers before reading nameLen, so supply exactly 3.
	payload := []byte{5, 0, 0} // tier=5, nameLen=0, extra byte (ignored)
	tier, defID, ok := DecodeMerchantBuy(payload)
	if !ok {
		t.Fatal("DecodeMerchantBuy returned !ok for empty defID with 3-byte payload")
	}
	if tier != 5 {
		t.Errorf("tier = %d, want 5", tier)
	}
	if defID != "" {
		t.Errorf("defID = %q, want empty", defID)
	}
}

// =============================================================================
// EncodeMerchantBuySuccess
// =============================================================================

func TestEncodeMerchantBuySuccess_WireFormat(t *testing.T) {
	buf := EncodeMerchantBuySuccess(5000, 42)

	// [success:1][balance:u32 LE][itemID:u32 LE][errLen:0]
	wantLen := 1 + 4 + 4 + 1
	if len(buf) != wantLen {
		t.Fatalf("len = %d, want %d", len(buf), wantLen)
	}

	off := 0
	if buf[off] != 1 {
		t.Errorf("success byte = %d, want 1", buf[off])
	}
	off++

	balance := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if balance != 5000 {
		t.Errorf("balance = %d, want 5000", balance)
	}

	itemID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if itemID != 42 {
		t.Errorf("itemID = %d, want 42", itemID)
	}

	if buf[off] != 0 {
		t.Errorf("err_len = %d, want 0 (empty error string)", buf[off])
	}
}

func TestEncodeMerchantBuySuccess_ZeroBalance(t *testing.T) {
	buf := EncodeMerchantBuySuccess(0, 0)
	if buf[0] != 1 {
		t.Errorf("success byte = %d, want 1", buf[0])
	}
	balance := binary.LittleEndian.Uint32(buf[1:5])
	if balance != 0 {
		t.Errorf("balance = %d, want 0", balance)
	}
	itemID := binary.LittleEndian.Uint32(buf[5:9])
	if itemID != 0 {
		t.Errorf("itemID = %d, want 0", itemID)
	}
}

// =============================================================================
// EncodeMerchantBuyError
// =============================================================================

func TestEncodeMerchantBuyError_WireFormat(t *testing.T) {
	errMsg := "insufficient scrip"
	buf := EncodeMerchantBuyError(errMsg)

	// [success:0][balance:u32(0)][itemID:u32(0)][errLen:u8][errMsg:...]
	wantLen := 1 + 4 + 4 + 1 + len(errMsg)
	if len(buf) != wantLen {
		t.Fatalf("len = %d, want %d", len(buf), wantLen)
	}

	off := 0
	if buf[off] != 0 {
		t.Errorf("success byte = %d, want 0", buf[off])
	}
	off++

	balance := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if balance != 0 {
		t.Errorf("balance = %d, want 0 (error path)", balance)
	}

	itemID := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if itemID != 0 {
		t.Errorf("itemID = %d, want 0 (error path)", itemID)
	}

	errLen := int(buf[off])
	off++
	if errLen != len(errMsg) {
		t.Fatalf("errLen = %d, want %d", errLen, len(errMsg))
	}
	gotMsg := string(buf[off : off+errLen])
	if gotMsg != errMsg {
		t.Errorf("errMsg = %q, want %q", gotMsg, errMsg)
	}
}

func TestEncodeMerchantBuyError_Empty(t *testing.T) {
	buf := EncodeMerchantBuyError("")
	// [success:0][balance:4][itemID:4][errLen:0] = 10 bytes
	wantLen := 1 + 4 + 4 + 1
	if len(buf) != wantLen {
		t.Fatalf("len = %d, want %d", len(buf), wantLen)
	}
	if buf[0] != 0 {
		t.Errorf("success byte = %d, want 0", buf[0])
	}
	if buf[9] != 0 {
		t.Errorf("err_len = %d, want 0", buf[9])
	}
}

// =============================================================================
// EncodeScripAward
// =============================================================================

func TestEncodeScripAward_WireFormat(t *testing.T) {
	buf := EncodeScripAward(250, 3500)

	// Wire: [amount:u16 LE][new_balance:u32 LE]
	if len(buf) != 6 {
		t.Fatalf("len = %d, want 6", len(buf))
	}

	amount := binary.LittleEndian.Uint16(buf[0:2])
	if amount != 250 {
		t.Errorf("amount = %d, want 250", amount)
	}

	balance := binary.LittleEndian.Uint32(buf[2:6])
	if balance != 3500 {
		t.Errorf("balance = %d, want 3500", balance)
	}
}

func TestEncodeScripAward_ZeroValues(t *testing.T) {
	buf := EncodeScripAward(0, 0)
	if len(buf) != 6 {
		t.Fatalf("len = %d, want 6", len(buf))
	}
	for i, b := range buf {
		if b != 0 {
			t.Errorf("buf[%d] = %d, want 0", i, b)
		}
	}
}

func TestEncodeScripAward_MaxUint16Amount(t *testing.T) {
	buf := EncodeScripAward(65535, 100000)
	amount := binary.LittleEndian.Uint16(buf[0:2])
	if amount != 65535 {
		t.Errorf("amount = %d, want 65535", amount)
	}
	balance := binary.LittleEndian.Uint32(buf[2:6])
	if balance != 100000 {
		t.Errorf("balance = %d, want 100000", balance)
	}
}

// =============================================================================
// EncodeMerchantState roundtrip
// =============================================================================

func TestEncodeMerchantState_ByteLength(t *testing.T) {
	tiers := []MerchantTierInfo{
		{
			ILvl:     10,
			Unlocked: true,
			Price:    500,
			Items: []MerchantItemInfo{
				{
					DefID:  "iron_helm",
					Name:   "Iron Helm",
					SlotID: 1,
					StatLines: []InventoryStatLine{
						{Stat: 1, Value: 5.0},
					},
				},
				{
					DefID:  "iron_chestplate",
					Name:   "Iron Chestplate",
					SlotID: 2,
					StatLines: []InventoryStatLine{
						{Stat: 2, Value: 10.0},
						{Stat: 3, Value: 3.0},
					},
				},
			},
		},
	}

	buf := EncodeMerchantState(1200, 800, 1, 2000, tiers)

	// Header: balance(4) + watermark(2) + season(2) + max_score(2) + tier_count(1) = 11
	// Per tier: ilvl(1) + unlocked(1) + price(4) + req_score(2) + item_count(1) = 9
	// Item 0: defID str8(1+9) + name str8(1+9) + slot(1) + stat_count(1) + 1 stat(1+4) = 27
	// Item 1: defID str8(1+15) + name str8(1+15) + slot(1) + stat_count(1) + 2 stats(2*(1+4)) = 44
	// Total = 11 + 9 + 27 + 44 = 91
	wantMin := 11 + 9 + 27 + 44
	if len(buf) < wantMin {
		t.Errorf("encoded len = %d, want >= %d", len(buf), wantMin)
	}
}

func TestEncodeMerchantState_HeaderFields(t *testing.T) {
	tiers := []MerchantTierInfo{
		{ILvl: 5, Unlocked: false, Price: 100, ReqScore: 1000},
	}

	buf := EncodeMerchantState(9999, 1234, 3, 5000, tiers)

	off := 0

	// balance: u32 LE
	balance := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if balance != 9999 {
		t.Errorf("balance = %d, want 9999", balance)
	}

	// watermark: u16 LE
	watermark := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if watermark != 1234 {
		t.Errorf("watermark = %d, want 1234", watermark)
	}

	// season: u16 LE
	season := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if season != 3 {
		t.Errorf("season = %d, want 3", season)
	}

	// max_score: u16 LE
	maxScore := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if maxScore != 5000 {
		t.Errorf("max_score = %d, want 5000", maxScore)
	}

	// tier_count: u8
	tierCount := buf[off]
	off++
	if tierCount != 1 {
		t.Errorf("tier_count = %d, want 1", tierCount)
	}

	// First tier: ilvl(1) + unlocked(1) + price(4) + item_count(1)
	ilvl := buf[off]
	off++
	if ilvl != 5 {
		t.Errorf("tier[0].ilvl = %d, want 5", ilvl)
	}

	unlocked := buf[off]
	off++
	if unlocked != 0 {
		t.Errorf("tier[0].unlocked = %d, want 0", unlocked)
	}

	price := binary.LittleEndian.Uint32(buf[off:])
	off += 4
	if price != 100 {
		t.Errorf("tier[0].price = %d, want 100", price)
	}

	reqScore := binary.LittleEndian.Uint16(buf[off:])
	off += 2
	if reqScore != 1000 {
		t.Errorf("tier[0].req_score = %d, want 1000", reqScore)
	}

	itemCount := buf[off]
	if itemCount != 0 {
		t.Errorf("tier[0].item_count = %d, want 0", itemCount)
	}
}

func TestEncodeMerchantState_UnlockedFlag(t *testing.T) {
	tiers := []MerchantTierInfo{
		{ILvl: 1, Unlocked: true, Price: 0},
	}
	buf := EncodeMerchantState(0, 0, 0, 0, tiers)

	// Header is 11 bytes; unlocked byte is at offset 11+1 (after ilvl).
	unlockedOff := 11 + 1
	if buf[unlockedOff] != 1 {
		t.Errorf("unlocked = %d, want 1", buf[unlockedOff])
	}
}

func TestEncodeMerchantState_EmptyTiers(t *testing.T) {
	buf := EncodeMerchantState(0, 0, 0, 0, nil)
	// Header only: balance(4) + watermark(2) + season(2) + max_score(2) + tier_count(1) = 11
	if len(buf) != 11 {
		t.Errorf("len = %d, want 11 (header only, no tiers)", len(buf))
	}
	if buf[10] != 0 {
		t.Errorf("tier_count = %d, want 0", buf[10])
	}
}
