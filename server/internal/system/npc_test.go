package system

import (
	"math"
	"testing"

	"codex-online/server/internal/entity"
)

func TestNPCSystem_IdleCountdown(t *testing.T) {
	npc := entity.NewNPC(1, "citizen", 2.0, 3.0, []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
	})
	w := &World{NPCs: []*entity.NPC{npc}}
	sys := &NPCSystem{}

	// NPC starts idle with IdleTimer = IdleDuration
	if npc.State != entity.NPCIdle {
		t.Fatalf("initial state = %d, want NPCIdle", npc.State)
	}
	if npc.IdleTimer != 3.0 {
		t.Fatalf("initial idle timer = %f, want 3.0", npc.IdleTimer)
	}

	// Tick 2 seconds — still idle
	sys.Tick(w, 2.0)
	if npc.State != entity.NPCIdle {
		t.Fatalf("after 2s: state = %d, want NPCIdle", npc.State)
	}
	if npc.IdleTimer != 1.0 {
		t.Fatalf("after 2s: idle timer = %f, want 1.0", npc.IdleTimer)
	}

	// Tick 1.5 seconds — timer expires, switch to walk
	sys.Tick(w, 1.5)
	if npc.State != entity.NPCWalk {
		t.Fatalf("after 3.5s: state = %d, want NPCWalk", npc.State)
	}
	if npc.WaypointIdx != 1 {
		t.Fatalf("waypoint idx = %d, want 1", npc.WaypointIdx)
	}
}

func TestNPCSystem_WalkToWaypoint(t *testing.T) {
	npc := entity.NewNPC(1, "citizen", 5.0, 1.0, []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
	})
	npc.State = entity.NPCWalk
	npc.WaypointIdx = 1
	w := &World{NPCs: []*entity.NPC{npc}}
	sys := &NPCSystem{}

	// Tick 1 second at speed 5 — should move 5 units toward (10,0,0)
	sys.Tick(w, 1.0)
	if npc.State != entity.NPCWalk {
		t.Fatalf("after 1s: state = %d, want NPCWalk", npc.State)
	}
	if math.Abs(float64(npc.Position.X-5.0)) > 0.01 {
		t.Fatalf("after 1s: pos.X = %f, want 5.0", npc.Position.X)
	}

	// Tick 1 more second — should reach (10,0,0) and go idle
	sys.Tick(w, 1.0)
	if npc.State != entity.NPCIdle {
		t.Fatalf("after 2s: state = %d, want NPCIdle", npc.State)
	}
	if math.Abs(float64(npc.Position.X-10.0)) > 0.01 {
		t.Fatalf("after 2s: pos.X = %f, want 10.0", npc.Position.X)
	}
	if npc.IdleTimer != 1.0 {
		t.Fatalf("idle timer = %f, want 1.0", npc.IdleTimer)
	}
}

func TestNPCSystem_WaypointWrap(t *testing.T) {
	npc := entity.NewNPC(1, "citizen", 100.0, 0.0, []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 1, Y: 0, Z: 0},
		{X: 2, Y: 0, Z: 0},
	})
	w := &World{NPCs: []*entity.NPC{npc}}
	sys := &NPCSystem{}

	// Idle timer is 0, so first tick should switch to walk toward wp 1
	npc.IdleTimer = 0
	sys.Tick(w, 0.05)
	if npc.WaypointIdx != 1 {
		t.Fatalf("wp = %d, want 1", npc.WaypointIdx)
	}

	// Arrive at wp 1, idle expires immediately (duration=0), advance to wp 2
	npc.State = entity.NPCIdle
	npc.IdleTimer = 0
	sys.Tick(w, 0.05)
	if npc.WaypointIdx != 2 {
		t.Fatalf("wp = %d, want 2", npc.WaypointIdx)
	}

	// Arrive at wp 2, idle expires, should wrap to wp 0
	npc.State = entity.NPCIdle
	npc.IdleTimer = 0
	sys.Tick(w, 0.05)
	if npc.WaypointIdx != 0 {
		t.Fatalf("wp = %d, want 0", npc.WaypointIdx)
	}
}

func TestNPCSystem_FacingDirection(t *testing.T) {
	npc := entity.NewNPC(1, "citizen", 5.0, 1.0, []entity.Vec3{
		{X: 0, Y: 0, Z: 0},
		{X: 10, Y: 0, Z: 0},
	})
	npc.State = entity.NPCWalk
	npc.WaypointIdx = 1
	w := &World{NPCs: []*entity.NPC{npc}}
	sys := &NPCSystem{}

	sys.Tick(w, 0.1)

	// Walking +X: atan2(1,0) + PI ≈ PI/2 + PI = 3PI/2
	expected := float32(math.Atan2(1, 0)) + math.Pi
	if math.Abs(float64(npc.RotationY-expected)) > 0.01 {
		t.Fatalf("rotation = %f, want %f", npc.RotationY, expected)
	}
}

func TestNPCSystem_SingleWaypointStaysIdle(t *testing.T) {
	npc := entity.NewNPC(1, "merchant", 0.0, 999.0, []entity.Vec3{
		{X: 5, Y: 0, Z: 5},
	})
	w := &World{NPCs: []*entity.NPC{npc}}
	sys := &NPCSystem{}

	// Even after many ticks, single-waypoint NPC stays idle
	for i := 0; i < 100; i++ {
		sys.Tick(w, 0.05)
	}
	if npc.State != entity.NPCIdle {
		t.Fatalf("state = %d, want NPCIdle", npc.State)
	}
	if npc.Position.X != 5.0 || npc.Position.Z != 5.0 {
		t.Fatalf("pos = (%f,%f), want (5,5)", npc.Position.X, npc.Position.Z)
	}
}

func TestNPCSystem_YAxisIgnored(t *testing.T) {
	npc := entity.NewNPC(1, "citizen", 5.0, 0.0, []entity.Vec3{
		{X: 0, Y: -200, Z: 0},
		{X: 10, Y: -200, Z: 0},
	})
	npc.State = entity.NPCWalk
	npc.WaypointIdx = 1
	w := &World{NPCs: []*entity.NPC{npc}}
	sys := &NPCSystem{}

	sys.Tick(w, 1.0)
	// Y should stay at -200 (ground plane)
	if npc.Position.Y != -200.0 {
		t.Fatalf("pos.Y = %f, want -200.0", npc.Position.Y)
	}
}
