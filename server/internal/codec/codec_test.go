package codec

import (
	"encoding/binary"
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

const testName = "Alice"

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
		{ID: 10, Position: entity.Vec3{X: 3, Y: 1, Z: -2}, Direction: entity.Vec3{X: 0, Z: 1}, Speed: 22.0, AngularVelocity: 0.5, VisualTag: "fireball"},
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
	if proj0.VisualTag != "fireball" {
		t.Errorf("proj[0].VisualTag = %q, want %q", proj0.VisualTag, "fireball")
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
			"flux":      {Current: 75.5},
			"stamina":   {Current: 50.0},
			"shield":    {Current: 25.0},
			"munitions": {Current: 10.0},
			"resonance": {Current: 30.0},
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
		{PeerID: 2, ClassName: entity.ClassVanguard, Username: "Bob", Ready: false},
	}
	buf := EncodeLobbyState(infos)

	if buf[0] != 2 {
		t.Fatalf("player_count = %d, want 2", buf[0])
	}
	off := 1
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
	if name != "Bob" {
		t.Errorf("p2 username = %q, want %q", name, "Bob")
	}
	if buf[off] != 0 {
		t.Errorf("p2 ready = %d, want 0", buf[off])
	}
}

// =============================================================================
// EncodeDamageEvent — wire format verification
// =============================================================================

func TestEncodeDamageEventWireFormat(t *testing.T) {
	buf := EncodeDamageEvent(0, 42, 10.0, 1.5, 2.0, -3.5, 0 /* SourcePlayerAttack */)

	if len(buf) != 21 {
		t.Fatalf("len = %d, want 21", len(buf))
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
		{ID: 20, ClassName: entity.ClassVanguard, Name: "Bob"},
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
		{PeerID: 2, Username: "Bob"},
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
	if m2Name != "Bob" {
		t.Errorf("m2 name = %q, want %q", m2Name, "Bob")
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

func TestGetU16(t *testing.T) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, 0x1234)
	got := getU16(buf)
	if got != 0x1234 {
		t.Errorf("getU16 = 0x%04X, want 0x1234", got)
	}
}
