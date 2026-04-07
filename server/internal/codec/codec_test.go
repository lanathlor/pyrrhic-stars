package codec

import (
	"encoding/binary"
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

// =============================================================================
// Roundtrip: EncodePlayerInput → DecodePlayerInput
// =============================================================================

func TestPlayerInputRoundtrip(t *testing.T) {
	buf := EncodePlayerInput(1.5, 2.0, -3.5, 0.7, 42, "run", 1.5, -0.3)
	msg := DecodePlayerInput(buf)
	if msg == nil {
		t.Fatal("DecodePlayerInput returned nil")
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
	if msg.AnimName != "run" {
		t.Errorf("AnimName = %q, want %q", msg.AnimName, "run")
	}
	if msg.AnimSpeed != 1.5 {
		t.Errorf("AnimSpeed = %f, want 1.5", msg.AnimSpeed)
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

	msg := DecodePlayerInput(buf)
	if msg == nil {
		t.Fatal("DecodePlayerInput returned nil for 16-byte payload")
	}
	if msg.PosX != 1.0 || msg.PosY != 2.0 || msg.PosZ != 3.0 {
		t.Errorf("position = (%f,%f,%f), want (1,2,3)", msg.PosX, msg.PosY, msg.PosZ)
	}
	if msg.Tick != 0 {
		t.Errorf("Tick = %d, want 0", msg.Tick)
	}
	if msg.AnimName != "" {
		t.Errorf("AnimName = %q, want empty", msg.AnimName)
	}
}

func TestPlayerInputTooShort(t *testing.T) {
	if DecodePlayerInput(nil) != nil {
		t.Error("nil payload should return nil")
	}
	if DecodePlayerInput([]byte{1, 2, 3}) != nil {
		t.Error("3-byte payload should return nil")
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
	payload = append(payload, []byte("gunner")...)
	msg := DecodeInteractInput(payload)
	if msg == nil {
		t.Fatal("DecodeInteractInput returned nil")
	}
	if msg.Action != 0 {
		t.Errorf("Action = %d, want 0", msg.Action)
	}
	if msg.ClassName != "gunner" {
		t.Errorf("ClassName = %q, want %q", msg.ClassName, "gunner")
	}
}

func TestInteractInputReadyToggle(t *testing.T) {
	msg := DecodeInteractInput([]byte{1})
	if msg == nil {
		t.Fatal("DecodeInteractInput returned nil")
	}
	if msg.Action != 1 {
		t.Errorf("Action = %d, want 1", msg.Action)
	}
	if msg.ClassName != "" {
		t.Errorf("ClassName = %q, want empty", msg.ClassName)
	}
}

func TestInteractInputTooShort(t *testing.T) {
	if DecodeInteractInput(nil) != nil {
		t.Error("nil payload should return nil")
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

func TestEncodeWorldStateWireFormat(t *testing.T) {
	p := &entity.Player{
		PeerID:    7,
		Position:  entity.Vec3{X: 1.0, Y: 2.0, Z: 3.0},
		RotationY: 0.5,
		Health:    100.0,
		State:     entity.PlayerStateAttack,
		ClassName: "gunner",
		Username:  "Alice",
		AnimName:  "idle",
		AnimSpeed: 1.0,
		AimPitch:  -0.1,
	}
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
	e.Position = entity.Vec3{X: 5.0, Y: 0.0, Z: -3.0}

	buf := EncodeWorldState(10, []*entity.Player{p}, []*entity.Enemy{e}, nil)

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
	if className != "gunner" {
		t.Errorf("class = %q, want %q", className, "gunner")
	}
	// name:str8
	nameLen := int(buf[off])
	off++
	name := string(buf[off : off+nameLen])
	off += nameLen
	if name != "Alice" {
		t.Errorf("name = %q, want %q", name, "Alice")
	}
	// anim
	animLen := int(buf[off])
	off++
	anim := string(buf[off : off+animLen])
	off += animLen
	if anim != "idle" {
		t.Errorf("anim = %q, want %q", anim, "idle")
	}
	// anim_speed
	animSpeed := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if animSpeed != 1.0 {
		t.Errorf("anim_speed = %f, want 1.0", animSpeed)
	}
	// aim_pitch
	aimPitch := math.Float32frombits(binary.LittleEndian.Uint32(buf[off:]))
	off += 4
	if aimPitch != -0.1 {
		t.Errorf("aim_pitch = %f, want -0.1", aimPitch)
	}

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
	// tick(4) + player_count(1) + enemy_count(1) + proj_count(1) = 7
	if len(buf) != 7 {
		t.Errorf("len = %d, want 7", len(buf))
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
}

// =============================================================================
// EncodeLobbyState — wire format verification
// =============================================================================

func TestEncodeLobbyStateWireFormat(t *testing.T) {
	infos := []LobbyPlayerInfo{
		{PeerID: 1, ClassName: "gunner", Username: "Alice", Ready: true},
		{PeerID: 2, ClassName: "vanguard", Username: "Bob", Ready: false},
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
	if class != "gunner" {
		t.Errorf("p1 class = %q, want %q", class, "gunner")
	}
	nameLen := int(buf[off])
	off++
	name := string(buf[off : off+nameLen])
	off += nameLen
	if name != "Alice" {
		t.Errorf("p1 username = %q, want %q", name, "Alice")
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
	if class != "vanguard" {
		t.Errorf("p2 class = %q, want %q", class, "vanguard")
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
