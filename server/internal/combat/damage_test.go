package combat

import (
	"encoding/binary"
	"math"
	"testing"

	"codex-online/server/internal/codec"
	"codex-online/server/internal/entity"
)

func makeVanguard(peerID uint16, pos entity.Vec3, rotY float32) *entity.Player {
	p := entity.NewPlayer(peerID, entity.ClassVanguard)
	p.Position = pos
	p.RotationY = rotY
	return p
}

// TestDamageEventWireFormat verifies the exact byte layout that broadcastDamageEvents
// produces, matching what the client's decode_damage_event expects.
// Client expects: [target_peer_id:u16 LE][source_peer_id:u16 LE][amount:f32 LE][hit_x:f32 LE][hit_y:f32 LE][hit_z:f32 LE][source_type:u8]
func TestDamageEventWireFormat(t *testing.T) {
	evt := DamageEvent{
		TargetPeerID: 0,  // enemy
		SourcePeerID: 42, // player who dealt damage
		Amount:       10.0,
		HitPos:       entity.Vec3{X: 1.5, Y: 2.0, Z: -3.5},
		SourceType:   SourcePlayerAttack,
	}

	buf := codec.EncodeDamageEvent(evt.TargetPeerID, evt.SourcePeerID, evt.Amount, evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z, evt.SourceType, evt.Overheal)

	if len(buf) != 25 {
		t.Fatalf("wire length = %d, want 25 bytes", len(buf))
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
	off++
	gotOverheal := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))

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
	if gotOverheal != 0 {
		t.Errorf("overheal = %f, want 0", gotOverheal)
	}
}

// --- AoE Resolution Tests ---

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
	// Player facing +Z, wide 180 degree arc
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

// TestEnemyDamageEventWireFormat verifies enemy->player damage events.
func TestEnemyDamageEventWireFormat(t *testing.T) {
	evt := DamageEvent{
		TargetPeerID: 7, // player who got hit
		SourcePeerID: 0, // enemy (no peer id)
		Amount:       25.0,
		HitPos:       entity.Vec3{X: 0, Y: 1.0, Z: 0},
		SourceType:   SourceEnemyMelee,
	}

	buf := codec.EncodeDamageEvent(evt.TargetPeerID, evt.SourcePeerID, evt.Amount, evt.HitPos.X, evt.HitPos.Y, evt.HitPos.Z, evt.SourceType, evt.Overheal)

	if len(buf) != 25 {
		t.Fatalf("wire length = %d, want 25", len(buf))
	}

	off := 0
	gotTarget := binary.LittleEndian.Uint16(buf[off : off+2])
	off += 2
	gotSource := binary.LittleEndian.Uint16(buf[off : off+2])
	off += 2
	gotAmount := math.Float32frombits(binary.LittleEndian.Uint32(buf[off : off+4]))

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
