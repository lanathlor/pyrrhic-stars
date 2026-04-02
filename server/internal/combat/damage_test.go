package combat

import (
	"encoding/binary"
	"math"
	"testing"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
)

// makeGunner creates a gunner aimed directly at a target position.
// Aim is computed from eye position (pos + Y=1.6), not foot position.
func makeGunner(peerID uint16, pos entity.Vec3, targetPos entity.Vec3) *entity.Player {
	eyePos := pos.Add(entity.Vec3{Y: 1.6})
	dir := targetPos.Sub(eyePos).Normalized()
	yaw := float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
	pitch := float32(math.Asin(float64(dir.Y)))
	return &entity.Player{
		PeerID:    peerID,
		ClassName: "gunner",
		Position:  pos,
		RotationY: yaw,
		AimPitch:  pitch,
		Health:    100,
		MaxHealth: 100,
		Alive:     true,
	}
}

func makeEnemy() *entity.Enemy {
	e := entity.NewEnemy(0)
	e.Alive = true
	e.State = entity.EnemyIdle
	e.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
	return e
}

func TestGunnerHit(t *testing.T) {
	enemy := makeEnemy()
	// Gunner at Z=10 aiming at origin (enemy center mass is at Y=1)
	player := makeGunner(42, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 0})

	evt := ResolvePlayerAttackOnEnemy(player, enemy, nil)
	if evt == nil {
		t.Fatal("expected hit, got nil")
	}
	if evt.TargetPeerID != 0 {
		t.Errorf("TargetPeerID = %d, want 0 (enemy)", evt.TargetPeerID)
	}
	if evt.Amount != 10.0 {
		t.Errorf("Amount = %f, want 10.0", evt.Amount)
	}
	if evt.SourceType != SourcePlayerAttack {
		t.Errorf("SourceType = %d, want %d", evt.SourceType, SourcePlayerAttack)
	}
	// SourcePeerID is set by the zone, not by ResolvePlayerAttackOnEnemy.
	// Verify it defaults to 0.
	if evt.SourcePeerID != 0 {
		t.Errorf("SourcePeerID = %d, want 0 (not yet set by zone)", evt.SourcePeerID)
	}
}

func TestGunnerMiss(t *testing.T) {
	enemy := makeEnemy()
	// Gunner aimed away from enemy (looking in +X direction)
	player := &entity.Player{
		PeerID:    1,
		ClassName: "gunner",
		Position:  entity.Vec3{X: 0, Y: 0, Z: 10},
		RotationY: float32(math.Pi / 2), // facing +X
		AimPitch:  0,
		Health:    100,
		MaxHealth: 100,
		Alive:     true,
	}

	evt := ResolvePlayerAttackOnEnemy(player, enemy, nil)
	if evt != nil {
		t.Errorf("expected miss (nil), got %+v", evt)
	}
}

func TestDeadEnemyIgnored(t *testing.T) {
	enemy := makeEnemy()
	enemy.Alive = false
	player := makeGunner(1, entity.Vec3{Z: 10}, entity.Vec3{Y: 1})

	evt := ResolvePlayerAttackOnEnemy(player, enemy, nil)
	if evt != nil {
		t.Errorf("expected nil for dead enemy, got %+v", evt)
	}
}

func TestVanguardMelee(t *testing.T) {
	enemy := makeEnemy()
	enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 1.5} // within melee range
	player := &entity.Player{
		PeerID:    7,
		ClassName: "vanguard",
		Position:  entity.Vec3{X: 0, Y: 0, Z: 0},
		RotationY: 0, // facing -Z... wait, Forward() = {-sin(rotY), 0, -cos(rotY)}
		// rotY=0 => forward = {0, 0, -1}, but enemy is at Z=+1.5
		// rotY=PI => forward = {0, 0, 1} — facing +Z toward enemy
		Health:    100,
		MaxHealth: 100,
		Alive:     true,
		ComboStep: 0,
	}
	// Face toward the enemy (positive Z)
	player.RotationY = float32(math.Pi)

	evt := ResolvePlayerAttackOnEnemy(player, enemy, nil)
	if evt == nil {
		t.Fatal("expected melee hit, got nil")
	}
	if evt.Amount != 30.0 {
		t.Errorf("Amount = %f, want 30.0 (combo step 0)", evt.Amount)
	}
}

// TestDamageEventWireFormat verifies the exact byte layout that broadcastDamageEvents
// produces, matching what the client's decode_damage_event expects.
// Client expects: [target_peer_id:u16 LE][source_peer_id:u16 LE][amount:f32 LE][hit_x:f32 LE][hit_y:f32 LE][hit_z:f32 LE][source_type:u8]
func TestDamageEventWireFormat(t *testing.T) {
	evt := DamageEvent{
		TargetPeerID: 0,     // enemy
		SourcePeerID: 42,    // player who dealt damage
		Amount:       10.0,
		HitPos:       entity.Vec3{X: 1.5, Y: 2.0, Z: -3.5},
		SourceType:   SourcePlayerAttack,
	}

	buf := codec.EncodeDamageEvent(evt.TargetPeerID, evt.SourcePeerID, evt.Amount, evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z, evt.SourceType)

	if len(buf) != 21 {
		t.Fatalf("wire length = %d, want 21 bytes", len(buf))
	}

	// Now decode like the client does (StreamPeerBuffer, little-endian)
	off := 0
	gotTarget := binary.LittleEndian.Uint16(buf[off : off+2])
	off += 2
	gotSource := binary.LittleEndian.Uint16(buf[off : off+2])
	off += 2
	gotAmount := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))
	off += 4
	gotHitX := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))
	off += 4
	gotHitY := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))
	off += 4
	gotHitZ := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))
	off += 4
	gotType := buf[off]

	if gotTarget != 0 {
		t.Errorf("target_peer_id = %d, want 0", gotTarget)
	}
	if gotSource != 42 {
		t.Errorf("source_peer_id = %d, want 42", gotSource)
	}
	if gotAmount != 10.0 {
		t.Errorf("amount = %f, want 10.0", gotAmount)
	}
	if gotHitX != 1.5 {
		t.Errorf("hit_x = %f, want 1.5", gotHitX)
	}
	if gotHitY != 2.0 {
		t.Errorf("hit_y = %f, want 2.0", gotHitY)
	}
	if gotHitZ != -3.5 {
		t.Errorf("hit_z = %f, want -3.5", gotHitZ)
	}
	if gotType != SourcePlayerAttack {
		t.Errorf("source_type = %d, want %d", gotType, SourcePlayerAttack)
	}
}

// TestEnemyDamageEventWireFormat verifies enemy->player damage events.
func TestEnemyDamageEventWireFormat(t *testing.T) {
	evt := DamageEvent{
		TargetPeerID: 7,  // player who got hit
		SourcePeerID: 0,  // enemy (no peer id)
		Amount:       25.0,
		HitPos:       entity.Vec3{X: 0, Y: 1.0, Z: 0},
		SourceType:   SourceEnemyMelee,
	}

	buf := codec.EncodeDamageEvent(evt.TargetPeerID, evt.SourcePeerID, evt.Amount, evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z, evt.SourceType)

	if len(buf) != 21 {
		t.Fatalf("wire length = %d, want 21", len(buf))
	}

	off := 0
	gotTarget := binary.LittleEndian.Uint16(buf[off : off+2])
	off += 2
	gotSource := binary.LittleEndian.Uint16(buf[off : off+2])
	off += 2
	gotAmount := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))
	off += 4

	if gotTarget != 7 {
		t.Errorf("target = %d, want 7", gotTarget)
	}
	if gotSource != 0 {
		t.Errorf("source = %d, want 0", gotSource)
	}
	if gotAmount != 25.0 {
		t.Errorf("amount = %f, want 25.0", gotAmount)
	}
}

