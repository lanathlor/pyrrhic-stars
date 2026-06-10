package entity

import "testing"

func TestNewNPCAtFirstWaypoint(t *testing.T) {
	wps := []Vec3{{X: 5, Y: 0.1, Z: 10}, {X: 15, Y: 0.1, Z: 20}}
	npc := NewNPC(1, "merchant", 2.0, 3.0, wps, nil)

	if npc.ID != 1 {
		t.Errorf("ID = %d, want 1", npc.ID)
	}
	if npc.DefName != "merchant" {
		t.Errorf("DefName = %q, want %q", npc.DefName, "merchant")
	}
	if npc.Position != wps[0] {
		t.Errorf("Position = %v, want %v", npc.Position, wps[0])
	}
	if npc.Speed != 2.0 {
		t.Errorf("Speed = %f, want 2.0", npc.Speed)
	}
	if npc.IdleDuration != 3.0 {
		t.Errorf("IdleDuration = %f, want 3.0", npc.IdleDuration)
	}
	if npc.IdleTimer != 3.0 {
		t.Errorf("IdleTimer = %f, want 3.0 (starts idle)", npc.IdleTimer)
	}
	if npc.State != NPCIdle {
		t.Errorf("State = %d, want NPCIdle", npc.State)
	}
	if npc.WaypointIdx != 0 {
		t.Errorf("WaypointIdx = %d, want 0", npc.WaypointIdx)
	}
}

func TestNewNPCNoWaypoints(t *testing.T) {
	npc := NewNPC(2, "guard", 1.5, 2.0, nil, nil)
	if npc.Position != (Vec3{}) {
		t.Errorf("Position = %v, want zero with no waypoints", npc.Position)
	}
}
