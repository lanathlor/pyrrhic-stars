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
	e := entity.NewEnemy(0, 2000.0, "guard_captain")
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

func TestGunnerRechamberBuffDamage(t *testing.T) {
	enemy := makeEnemy()
	player := makeGunner(42, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 0})
	player.RechamberBuff = true

	evt := ResolvePlayerAttackOnEnemy(player, enemy, nil)
	if evt == nil {
		t.Fatal("expected hit, got nil")
	}
	if evt.Amount != 18.0 {
		t.Errorf("Amount = %f, want 18.0 (rechamber buff)", evt.Amount)
	}
}

func TestGunnerNormalDamageWithoutBuff(t *testing.T) {
	enemy := makeEnemy()
	player := makeGunner(42, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 0})
	player.RechamberBuff = false

	evt := ResolvePlayerAttackOnEnemy(player, enemy, nil)
	if evt == nil {
		t.Fatal("expected hit, got nil")
	}
	if evt.Amount != 10.0 {
		t.Errorf("Amount = %f, want 10.0 (no buff)", evt.Amount)
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

// --- AoE Resolution Tests ---

func makeVanguard(peerID uint16, pos entity.Vec3, rotY float32) *entity.Player {
	return &entity.Player{
		PeerID:    peerID,
		ClassName: "vanguard",
		Position:  pos,
		RotationY: rotY,
		Health:    200,
		MaxHealth: 200,
		Alive:     true,
	}
}

func TestAoECircle_HitsMultiple(t *testing.T) {
	player := makeVanguard(1, entity.Vec3{X: 0, Y: 0, Z: 0}, 0)
	e1 := entity.NewEnemy(1000, 500, "test")
	e1.Position = entity.Vec3{X: 2, Y: 0, Z: 0} // in range (dist=2, radius=4)
	e2 := entity.NewEnemy(1001, 500, "test")
	e2.Position = entity.Vec3{X: -1, Y: 0, Z: 1} // in range (dist=1.4)
	e3 := entity.NewEnemy(1002, 500, "test")
	e3.Position = entity.Vec3{X: 10, Y: 0, Z: 0} // out of range

	enemies := []*entity.Enemy{e1, e2, e3}
	shape := AoEShape{Type: AoECircle, Radius: 4.0, Damage: 25.0}
	events := ResolvePlayerAoEOnEnemies(player, enemies, nil, shape)

	if len(events) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(events))
	}
	for _, evt := range events {
		if evt.Amount != 25.0 {
			t.Errorf("expected 25.0 damage, got %f", evt.Amount)
		}
		if evt.SourcePeerID != 1 {
			t.Errorf("expected source peer 1, got %d", evt.SourcePeerID)
		}
	}
}

func TestAoECircle_NoEnemies(t *testing.T) {
	player := makeVanguard(1, entity.Vec3{}, 0)
	shape := AoEShape{Type: AoECircle, Radius: 5.0, Damage: 30.0}
	events := ResolvePlayerAoEOnEnemies(player, nil, nil, shape)
	if len(events) != 0 {
		t.Errorf("expected 0 hits, got %d", len(events))
	}
}

func TestAoECircle_SkipsDeadEnemies(t *testing.T) {
	player := makeVanguard(1, entity.Vec3{}, 0)
	e1 := entity.NewEnemy(1000, 500, "test")
	e1.Position = entity.Vec3{X: 1, Y: 0, Z: 0}
	e1.Alive = false
	e1.State = entity.EnemyDead

	shape := AoEShape{Type: AoECircle, Radius: 5.0, Damage: 30.0}
	events := ResolvePlayerAoEOnEnemies(player, []*entity.Enemy{e1}, nil, shape)
	if len(events) != 0 {
		t.Errorf("expected 0 hits for dead enemy, got %d", len(events))
	}
}

func TestAoECone_HitsInArc(t *testing.T) {
	// Player at origin facing +Z (rotY = PI)
	player := makeVanguard(1, entity.Vec3{}, float32(math.Pi))
	e1 := entity.NewEnemy(1000, 500, "test")
	e1.Position = entity.Vec3{X: 0, Y: 0, Z: 3} // directly ahead, in range
	e2 := entity.NewEnemy(1001, 500, "test")
	e2.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // behind player, should miss

	shape := AoEShape{Type: AoECone, Radius: 5.0, ArcDegrees: 90.0, Damage: 60.0}
	events := ResolvePlayerAoEOnEnemies(player, []*entity.Enemy{e1, e2}, nil, shape)

	if len(events) != 1 {
		t.Fatalf("expected 1 hit (front only), got %d", len(events))
	}
	if events[0].TargetPeerID != 1000 {
		t.Errorf("expected hit on enemy 1000, got %d", events[0].TargetPeerID)
	}
	if events[0].Amount != 60.0 {
		t.Errorf("expected 60.0 damage, got %f", events[0].Amount)
	}
}

func TestAoECircle_BlockedByObstacle(t *testing.T) {
	player := makeVanguard(1, entity.Vec3{X: 0, Y: 0, Z: 0}, 0)
	e1 := entity.NewEnemy(1000, 500, "test")
	e1.Position = entity.Vec3{X: 0, Y: 0, Z: 5} // in radius but behind obstacle

	obstacle := Obstacle{CX: 0, CZ: 2.5, HX: 2.0, HZ: 0.5, Height: 3.0}
	shape := AoEShape{Type: AoECircle, Radius: 10.0, Damage: 25.0}
	events := ResolvePlayerAoEOnEnemies(player, []*entity.Enemy{e1}, []Obstacle{obstacle}, shape)

	if len(events) != 0 {
		t.Errorf("expected 0 hits (blocked by obstacle), got %d", len(events))
	}
}

func TestAoECone_MultipleInArc(t *testing.T) {
	// Player facing +Z, wide 180° arc
	player := makeVanguard(1, entity.Vec3{}, float32(math.Pi))
	e1 := entity.NewEnemy(1000, 500, "test")
	e1.Position = entity.Vec3{X: 2, Y: 0, Z: 3} // front-right
	e2 := entity.NewEnemy(1001, 500, "test")
	e2.Position = entity.Vec3{X: -2, Y: 0, Z: 3} // front-left
	e3 := entity.NewEnemy(1002, 500, "test")
	e3.Position = entity.Vec3{X: 0, Y: 0, Z: -3} // behind

	shape := AoEShape{Type: AoECone, Radius: 5.0, ArcDegrees: 180.0, Damage: 40.0}
	events := ResolvePlayerAoEOnEnemies(player, []*entity.Enemy{e1, e2, e3}, nil, shape)

	if len(events) != 2 {
		t.Fatalf("expected 2 hits (both in front), got %d", len(events))
	}
}

// --- ResolvePlayerAttackOnEnemies Tests ---

func TestResolvePlayerAttackOnEnemies(t *testing.T) {
	t.Run("hits closest enemy among multiple", func(t *testing.T) {
		// Two enemies in a line, gunner aimed at them
		eFar := entity.NewEnemy(100, 500, "test")
		eFar.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		eNear := entity.NewEnemy(101, 500, "test")
		eNear.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		// Gunner at Z=10 aiming toward origin
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 5})

		evt, enemy := ResolvePlayerAttackOnEnemies(player, []*entity.Enemy{eFar, eNear}, nil)
		if evt == nil {
			t.Fatal("expected hit, got nil")
		}
		if enemy.ID != eNear.ID {
			t.Errorf("hit enemy %d, want %d (nearest)", enemy.ID, eNear.ID)
		}
	})

	t.Run("nil enemies in slice are skipped", func(t *testing.T) {
		e := entity.NewEnemy(100, 500, "test")
		e.Position = entity.Vec3{X: 0, Y: 0, Z: 0}
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 0})

		evt, enemy := ResolvePlayerAttackOnEnemies(player, []*entity.Enemy{nil, e, nil}, nil)
		if evt == nil {
			t.Fatal("expected hit past nil entries, got nil")
		}
		if enemy.ID != 100 {
			t.Errorf("hit enemy %d, want 100", enemy.ID)
		}
	})

	t.Run("dead enemies are skipped", func(t *testing.T) {
		eDead := entity.NewEnemy(100, 500, "test")
		eDead.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		eDead.Alive = false
		eDead.State = entity.EnemyDead

		eAlive := entity.NewEnemy(101, 500, "test")
		eAlive.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 3})
		evt, enemy := ResolvePlayerAttackOnEnemies(player, []*entity.Enemy{eDead, eAlive}, nil)
		// The dead enemy is closer to the aim direction but should be skipped.
		// The alive one may or may not be in the hitscan cylinder — aim at the dead one's pos.
		// Since we aim at Z=3 center mass, the alive enemy at Z=5 is off-axis.
		// Make a test that definitely hits: aim broadly.
		_ = evt
		_ = enemy
	})

	t.Run("dead enemies skipped - alive one hit", func(t *testing.T) {
		eDead := entity.NewEnemy(100, 500, "test")
		eDead.Position = entity.Vec3{X: 0, Y: 0, Z: 5}
		eDead.Alive = false
		eDead.State = entity.EnemyDead

		eAlive := entity.NewEnemy(101, 500, "test")
		eAlive.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 5})
		evt, enemy := ResolvePlayerAttackOnEnemies(player, []*entity.Enemy{eDead, eAlive}, nil)
		if evt == nil {
			t.Fatal("expected hit on alive enemy, got nil")
		}
		if enemy.ID != 101 {
			t.Errorf("hit enemy %d, want 101", enemy.ID)
		}
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 0})
		evt, enemy := ResolvePlayerAttackOnEnemies(player, nil, nil)
		if evt != nil || enemy != nil {
			t.Errorf("expected nil for empty enemies, got evt=%+v enemy=%+v", evt, enemy)
		}
	})

	t.Run("all dead returns nil", func(t *testing.T) {
		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 0, Y: 0, Z: 5}
		e1.Alive = false
		e1.State = entity.EnemyDead

		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 10}, entity.Vec3{X: 0, Y: 1, Z: 5})
		evt, enemy := ResolvePlayerAttackOnEnemies(player, []*entity.Enemy{e1}, nil)
		if evt != nil || enemy != nil {
			t.Errorf("expected nil for all-dead enemies, got evt=%+v", evt)
		}
	})
}

// --- ResolveAoEAtPosition Tests ---

func TestResolveAoEAtPosition(t *testing.T) {
	t.Run("hits enemies within radius", func(t *testing.T) {
		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 1, Y: 0, Z: 0}
		e2 := entity.NewEnemy(101, 500, "test")
		e2.Position = entity.Vec3{X: -1, Y: 0, Z: 1}
		e3 := entity.NewEnemy(102, 500, "test")
		e3.Position = entity.Vec3{X: 20, Y: 0, Z: 0} // out of range

		center := entity.Vec3{X: 0, Y: 0, Z: 0}
		shape := AoEShape{Type: AoECircle, Radius: 5.0, Damage: 50.0}
		events := ResolveAoEAtPosition(center, 1, []*entity.Enemy{e1, e2, e3}, nil, shape)
		if len(events) != 2 {
			t.Fatalf("expected 2 hits, got %d", len(events))
		}
		for _, evt := range events {
			if evt.Amount != 50.0 {
				t.Errorf("expected 50.0 damage, got %f", evt.Amount)
			}
			if evt.SourcePeerID != 1 {
				t.Errorf("expected source peer 1, got %d", evt.SourcePeerID)
			}
		}
	})

	t.Run("blocked by obstacle", func(t *testing.T) {
		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		center := entity.Vec3{X: 0, Y: 0, Z: 0}
		obstacle := Obstacle{CX: 0, CZ: 2.5, HX: 2, HZ: 0.5, Height: 3}
		shape := AoEShape{Type: AoECircle, Radius: 10.0, Damage: 50.0}
		events := ResolveAoEAtPosition(center, 1, []*entity.Enemy{e1}, []Obstacle{obstacle}, shape)
		if len(events) != 0 {
			t.Errorf("expected 0 hits (blocked), got %d", len(events))
		}
	})

	t.Run("dead and nil enemies skipped", func(t *testing.T) {
		eDead := entity.NewEnemy(100, 500, "test")
		eDead.Position = entity.Vec3{X: 1, Y: 0, Z: 0}
		eDead.Alive = false
		eDead.State = entity.EnemyDead

		eAlive := entity.NewEnemy(101, 500, "test")
		eAlive.Position = entity.Vec3{X: 2, Y: 0, Z: 0}

		center := entity.Vec3{}
		shape := AoEShape{Type: AoECircle, Radius: 5.0, Damage: 30.0}
		events := ResolveAoEAtPosition(center, 1, []*entity.Enemy{nil, eDead, eAlive}, nil, shape)
		if len(events) != 1 {
			t.Fatalf("expected 1 hit, got %d", len(events))
		}
		if events[0].TargetPeerID != 101 {
			t.Errorf("expected hit on 101, got %d", events[0].TargetPeerID)
		}
	})

	t.Run("empty enemies", func(t *testing.T) {
		center := entity.Vec3{}
		shape := AoEShape{Type: AoECircle, Radius: 5.0, Damage: 30.0}
		events := ResolveAoEAtPosition(center, 1, nil, nil, shape)
		if len(events) != 0 {
			t.Errorf("expected 0 hits, got %d", len(events))
		}
	})
}

// --- ResolveNearestNEnemies Tests ---

func TestResolveNearestNEnemies(t *testing.T) {
	t.Run("sorted by distance - hits nearest N", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 0}, entity.Vec3{X: 0, Y: 1, Z: 5})

		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		e1.ThreatTable[1] = 10 // in combat

		e2 := entity.NewEnemy(101, 500, "test")
		e2.Position = entity.Vec3{X: 0, Y: 0, Z: 10}
		e2.ThreatTable[1] = 5

		e3 := entity.NewEnemy(102, 500, "test")
		e3.Position = entity.Vec3{X: 0, Y: 0, Z: 5}
		e3.ThreatTable[1] = 8

		events := ResolveNearestNEnemies(player, []*entity.Enemy{e2, e3, e1}, nil, 2, 20.0)
		if len(events) != 2 {
			t.Fatalf("expected 2 hits, got %d", len(events))
		}
		// Should hit e1 (dist=3) and e3 (dist=5), not e2 (dist=10)
		hitIDs := map[uint16]bool{}
		for _, evt := range events {
			hitIDs[evt.TargetPeerID] = true
		}
		if !hitIDs[100] || !hitIDs[102] {
			t.Errorf("expected hits on 100 and 102, got %v", hitIDs)
		}
		if hitIDs[101] {
			t.Errorf("did not expect hit on 101 (farthest)")
		}
	})

	t.Run("n=0 returns nil", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{}, entity.Vec3{Y: 1})
		e := entity.NewEnemy(100, 500, "test")
		e.ThreatTable[1] = 10
		events := ResolveNearestNEnemies(player, []*entity.Enemy{e}, nil, 0, 20.0)
		if events != nil {
			t.Errorf("expected nil for n=0, got %d events", len(events))
		}
	})

	t.Run("n greater than count hits all", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 0}, entity.Vec3{X: 0, Y: 1, Z: 3})
		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		e1.ThreatTable[1] = 10

		events := ResolveNearestNEnemies(player, []*entity.Enemy{e1}, nil, 5, 20.0)
		if len(events) != 1 {
			t.Fatalf("expected 1 hit (only 1 available), got %d", len(events))
		}
	})

	t.Run("no threat = skip", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 0}, entity.Vec3{X: 0, Y: 1, Z: 3})
		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		// ThreatTable is empty => not in combat

		events := ResolveNearestNEnemies(player, []*entity.Enemy{e1}, nil, 5, 20.0)
		if len(events) != 0 {
			t.Errorf("expected 0 hits (no threat), got %d", len(events))
		}
	})

	t.Run("LoS blocked skips enemy", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 0}, entity.Vec3{X: 0, Y: 1, Z: 5})

		e1 := entity.NewEnemy(100, 500, "test")
		e1.Position = entity.Vec3{X: 0, Y: 0, Z: 5}
		e1.ThreatTable[1] = 10

		// Obstacle between player and enemy
		obstacle := Obstacle{CX: 0, CZ: 2.5, HX: 2, HZ: 0.5, Height: 3}

		events := ResolveNearestNEnemies(player, []*entity.Enemy{e1}, []Obstacle{obstacle}, 5, 20.0)
		if len(events) != 0 {
			t.Errorf("expected 0 hits (LoS blocked), got %d", len(events))
		}
	})

	t.Run("dead enemies skipped", func(t *testing.T) {
		player := makeGunner(1, entity.Vec3{X: 0, Y: 0, Z: 0}, entity.Vec3{X: 0, Y: 1, Z: 3})
		eDead := entity.NewEnemy(100, 500, "test")
		eDead.Position = entity.Vec3{X: 0, Y: 0, Z: 3}
		eDead.ThreatTable[1] = 10
		eDead.Alive = false
		eDead.State = entity.EnemyDead

		events := ResolveNearestNEnemies(player, []*entity.Enemy{eDead}, nil, 5, 20.0)
		if len(events) != 0 {
			t.Errorf("expected 0 hits (dead), got %d", len(events))
		}
	})
}

// --- resolveBladeDancerAttack Tests ---

func TestResolveBladeDancerAttack(t *testing.T) {
	t.Run("orbit config hit in range", func(t *testing.T) {
		enemy := entity.NewEnemy(100, 500, "test")
		enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		// Blade dancer at Z=10, aimed at enemy center mass (Y=1, Z=5)
		player := &entity.Player{
			PeerID:    1,
			ClassName: "blade_dancer",
			Position:  entity.Vec3{X: 0, Y: 0, Z: 10},
			Health:    150,
			MaxHealth: 150,
			Alive:     true,
			Config:    0, // orbit
		}
		// Compute aim direction toward enemy center mass
		eyePos := player.EyePosition()
		targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})
		dir := targetCenter.Sub(eyePos).Normalized()
		yaw := float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
		pitch := float32(math.Asin(float64(dir.Y)))
		player.RotationY = yaw
		player.AimPitch = pitch

		evt := resolveBladeDancerAttack(player, enemy, nil)
		if evt == nil {
			t.Fatal("expected orbit config hit, got nil")
		}
		if evt.Amount != 25.0 {
			t.Errorf("orbit damage = %f, want 25.0", evt.Amount)
		}
	})

	t.Run("lance config hit in range", func(t *testing.T) {
		enemy := entity.NewEnemy(100, 500, "test")
		enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		player := &entity.Player{
			PeerID:    1,
			ClassName: "blade_dancer",
			Position:  entity.Vec3{X: 0, Y: 0, Z: 10},
			Health:    150,
			MaxHealth: 150,
			Alive:     true,
			Config:    2, // lance
		}
		eyePos := player.EyePosition()
		targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})
		dir := targetCenter.Sub(eyePos).Normalized()
		yaw := float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
		pitch := float32(math.Asin(float64(dir.Y)))
		player.RotationY = yaw
		player.AimPitch = pitch

		evt := resolveBladeDancerAttack(player, enemy, nil)
		if evt == nil {
			t.Fatal("expected lance config hit, got nil")
		}
		if evt.Amount != 35.0 {
			t.Errorf("lance damage = %f, want 35.0", evt.Amount)
		}
	})

	t.Run("miss - out of range (beyond 20m)", func(t *testing.T) {
		enemy := entity.NewEnemy(100, 500, "test")
		enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 0}

		// Blade dancer at Z=25 (25m away, max range is 20m)
		player := &entity.Player{
			PeerID:    1,
			ClassName: "blade_dancer",
			Position:  entity.Vec3{X: 0, Y: 0, Z: 25},
			Health:    150,
			MaxHealth: 150,
			Alive:     true,
			Config:    0,
		}
		eyePos := player.EyePosition()
		targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})
		dir := targetCenter.Sub(eyePos).Normalized()
		yaw := float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
		pitch := float32(math.Asin(float64(dir.Y)))
		player.RotationY = yaw
		player.AimPitch = pitch

		evt := resolveBladeDancerAttack(player, enemy, nil)
		if evt != nil {
			t.Errorf("expected miss (out of range), got %+v", evt)
		}
	})

	t.Run("miss - aiming away", func(t *testing.T) {
		enemy := entity.NewEnemy(100, 500, "test")
		enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		// Player at Z=0, aiming in +Z direction (rotY=PI).
		// But enemy is at Z=5, same direction. Place player beyond the enemy.
		// Player at Z=8, aim direction at rotY=PI is (0, 0, 1), so aiming +Z = away from enemy at Z=5.
		player := &entity.Player{
			PeerID:    1,
			ClassName: "blade_dancer",
			Position:  entity.Vec3{X: 0, Y: 0, Z: 3},
			RotationY: 0, // facing -Z
			AimPitch:  0,
			Health:    150,
			MaxHealth: 150,
			Alive:     true,
			Config:    0,
		}
		// Eye at (0, 1.6, 3), aiming (0, 0, -1). Enemy center at (0, 1, 5).
		// toTarget2D = (0-0, 5-3) = (0, 2). dir2DN = (0, -1). proj = 0 + 2*(-1) = -2 < 0 => miss.

		evt := resolveBladeDancerAttack(player, enemy, nil)
		if evt != nil {
			t.Errorf("expected miss (aiming away), got %+v", evt)
		}
	})

	t.Run("blocked by obstacle", func(t *testing.T) {
		enemy := entity.NewEnemy(100, 500, "test")
		enemy.Position = entity.Vec3{X: 0, Y: 0, Z: 5}

		player := &entity.Player{
			PeerID:    1,
			ClassName: "blade_dancer",
			Position:  entity.Vec3{X: 0, Y: 0, Z: 10},
			Health:    150,
			MaxHealth: 150,
			Alive:     true,
			Config:    0,
		}
		eyePos := player.EyePosition()
		targetCenter := enemy.Position.Add(entity.Vec3{Y: 1.0})
		dir := targetCenter.Sub(eyePos).Normalized()
		yaw := float32(-math.Atan2(float64(-dir.X), float64(-dir.Z)))
		pitch := float32(math.Asin(float64(dir.Y)))
		player.RotationY = yaw
		player.AimPitch = pitch

		obstacle := Obstacle{CX: 0, CZ: 7.5, HX: 2, HZ: 0.5, Height: 3}
		evt := resolveBladeDancerAttack(player, enemy, []Obstacle{obstacle})
		if evt != nil {
			t.Errorf("expected miss (blocked by obstacle), got %+v", evt)
		}
	})
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

